package proxy

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"hacp-sidecar/internal/budget"
	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/provenance"
	"hacp-sidecar/internal/wire"
)

func TestHTTPRequestPathBindingIncludesQueryString(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"http://example.test/transfer?account=A",
		nil,
	)

	constraints := &wire.Constraints{
		Method: http.MethodPost,
		Path:   "/transfer?account=A",
	}

	reqCtx := &evaluate.RequestContext{
		Method: req.Method,
		Path:   requestBindingPath(req),
	}

	guard := evaluate.NewDefaultScopeGuard()

	if !guard.MatchRequestConstraints(constraints, reqCtx) {
		t.Fatalf(
			"spec-conformant request binding rejected: constraint path=%q request URI=%q request context path=%q",
			constraints.Path,
			req.URL.RequestURI(),
			reqCtx.Path,
		)
	}
}

func TestHTTPRequestPathBindingRejectsDifferentQueryString(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"http://example.test/transfer?account=B",
		nil,
	)

	constraints := &wire.Constraints{
		Method: http.MethodPost,
		Path:   "/transfer?account=A",
	}

	reqCtx := &evaluate.RequestContext{
		Method: req.Method,
		Path:   requestBindingPath(req),
	}

	guard := evaluate.NewDefaultScopeGuard()

	if guard.MatchRequestConstraints(constraints, reqCtx) {
		t.Fatalf(
			"query substitution accepted: constraint path=%q request URI=%q request context path=%q",
			constraints.Path,
			req.URL.RequestURI(),
			reqCtx.Path,
		)
	}
}

func TestServeHTTPUsesQueryInclusivePathBinding(t *testing.T) {
	const keyID = "ph1c-query-binding-key"

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}

	keyResolver := wire.NewStaticKeyResolver()
	if err := keyResolver.AddKey(keyID, publicKey); err != nil {
		t.Fatalf("AddKey() error = %v", err)
	}

	pipeline := evaluate.NewPipeline(
		keyResolver,
		evaluate.NewInMemoryRevocationStore(),
		budget.NewLedger(),
		evaluate.NewDefaultScopeGuard(),
		provenance.NewNoopWriter(),
	)

	upstreamRequestURI := make(chan string, 1)

	upstream := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upstreamRequestURI <- r.URL.RequestURI()
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer upstream.Close()

	provenanceLog := provenance.NewRingBuffer(
		16,
		filepath.Join(t.TempDir(), "provenance.jsonl"),
	)
	defer provenanceLog.Stop()

	handler := NewHandler(
		pipeline,
		provenanceLog,
		upstream.URL,
	)

	env, envJSON := signedBindingEnvelope(
		t,
		privateKey,
		keyID,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"http://sidecar.test/transfer?account=A",
		nil,
	)

	payloadHash := wire.SHA256Hex(nil)

	proposedAction := SynthesizeProposedAction(
		req,
		env,
		payloadHash,
	)

	actionHash, err := proposedAction.Hash()
	if err != nil {
		t.Fatalf("ProposedAction.Hash() error = %v", err)
	}

	tokenJSON := signedBindingToken(
		t,
		privateKey,
		keyID,
		env.EnvelopeID,
		actionHash,
		req.URL.RequestURI(),
		payloadHash,
	)

	req.Header.Set(
		wire.HeaderIntentEnvelope,
		wire.Base64URLEncode(envJSON),
	)
	req.Header.Set(
		wire.HeaderDecisionToken,
		wire.Base64URLEncode(tokenJSON),
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d body=%q decision=%q reason=%q, want %d",
			rec.Code,
			rec.Body.String(),
			rec.Header().Get("X-HACP-Decision"),
			rec.Header().Get("X-HACP-Reason"),
			http.StatusOK,
		)
	}

	if got := rec.Header().Get("X-HACP-Decision"); got != "ALLOW" {
		t.Fatalf("X-HACP-Decision = %q, want ALLOW", got)
	}

	select {
	case got := <-upstreamRequestURI:
		if got != "/transfer?account=A" {
			t.Fatalf(
				"upstream RequestURI = %q, want %q",
				got,
				"/transfer?account=A",
			)
		}
	case <-time.After(time.Second):
		t.Fatal("authorized request was not forwarded upstream")
	}
}

func signedBindingEnvelope(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	keyID string,
) (*wire.IntentEnvelope, []byte) {
	t.Helper()

	now := time.Now().Unix()

	unsigned := map[string]any{
		"hacp_version":     "0.9",
		"envelope_id":      "ph1c-query-binding-envelope",
		"principal":        "ph1c-human",
		"principal_kind":   "human",
		"intent_statement": "PH-1C request path binding test",
		"scope": map[string]any{
			"verbs":            []string{"write"},
			"resource_classes": []string{"test"},
			"audiences":        []string{"internal"},
			"reversibility":    []string{"reversible"},
			"externality":      []string{"internal"},
			"data_classes":     []string{"internal"},
		},
		"issued_at":     now - 60,
		"expires_at":    now + 600,
		"signer_key_id": keyID,
	}

	signedJSON := signBindingObject(
		t,
		privateKey,
		unsigned,
	)

	env, err := wire.ParseIntentEnvelope(signedJSON)
	if err != nil {
		t.Fatalf("ParseIntentEnvelope() error = %v", err)
	}

	return env, signedJSON
}

func signedBindingToken(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	keyID string,
	envelopeID string,
	actionHash string,
	path string,
	payloadHash string,
) []byte {
	t.Helper()

	now := time.Now().Unix()
	maxUses := 1

	unsigned := map[string]any{
		"hacp_version":  "0.9",
		"token_id":      "ph1c-query-binding-token",
		"envelope_id":   envelopeID,
		"action_hash":   actionHash,
		"policy_digest": "ph1c-query-binding-policy",
		"principal":     "ph1c-human",
		"signer_key_id": keyID,
		"issued_at":     now - 60,
		"expires_at":    now + 600,
		"decision":      "ALLOW",
		"constraints": map[string]any{
			"method":       http.MethodPost,
			"path":         path,
			"payload_hash": payloadHash,
			"max_uses":     maxUses,
		},
	}

	return signBindingObject(
		t,
		privateKey,
		unsigned,
	)
}

func signBindingObject(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	unsigned map[string]any,
) []byte {
	t.Helper()

	unsignedJSON, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatalf("json.Marshal(unsigned) error = %v", err)
	}

	canonical, err := wire.CanonicalizeJSON(unsignedJSON)
	if err != nil {
		t.Fatalf("CanonicalizeJSON(unsigned) error = %v", err)
	}

	unsigned["signature"] = wire.Base64URLEncode(
		ed25519.Sign(privateKey, canonical),
	)

	signedJSON, err := json.Marshal(unsigned)
	if err != nil {
		t.Fatalf("json.Marshal(signed) error = %v", err)
	}

	return signedJSON
}
