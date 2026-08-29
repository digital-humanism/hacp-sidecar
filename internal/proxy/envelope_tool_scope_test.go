package proxy

import (
	"crypto/ed25519"
	"crypto/rand"
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

func TestServeHTTPRejectsToolOutsideEnvelopeScope(t *testing.T) {
	const keyID = "envelope-tool-scope-key"

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

	upstreamReached := make(chan struct{}, 1)

	upstream := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upstreamReached <- struct{}{}
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

	env, envJSON := signedEnvelopeWithToolScope(
		t,
		privateKey,
		keyID,
		[]string{"tool.allowed"},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"http://sidecar.test/transfer",
		nil,
	)

	req.Header.Set(
		"X-HACP-Tool-Name",
		"tool.denied",
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

	if got := rec.Header().Get("X-HACP-Decision"); got != "DENY" {
		t.Fatalf(
			"X-HACP-Decision = %q, want DENY; out-of-scope tool_name was authorized",
			got,
		)
	}

	if got := rec.Header().Get("X-HACP-Reason"); got != evaluate.ReasonScopeExceeded {
		t.Fatalf(
			"X-HACP-Reason = %q, want %q",
			got,
			evaluate.ReasonScopeExceeded,
		)
	}

	select {
	case <-upstreamReached:
		t.Fatal("request with tool_name outside envelope scope was forwarded upstream")
	default:
	}
}

func signedEnvelopeWithToolScope(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	keyID string,
	toolNames []string,
) (*wire.IntentEnvelope, []byte) {
	t.Helper()

	now := time.Now().Unix()

	unsigned := map[string]any{
		"hacp_version":     "0.9",
		"envelope_id":      "envelope-tool-scope-test",
		"principal":        "tool-scope-human",
		"principal_kind":   "human",
		"intent_statement": "Envelope tool scope enforcement test",
		"scope": map[string]any{
			"verbs":            []string{"write"},
			"resource_classes": []string{"test"},
			"audiences":        []string{"internal"},
			"reversibility":    []string{"reversible"},
			"externality":      []string{"internal"},
			"data_classes":     []string{"internal"},
			"tool_names":       toolNames,
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
