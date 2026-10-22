package proxy

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hacp-sidecar/internal/budget"
	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/provenance"
	"hacp-sidecar/internal/wire"
)

func newR43SignedRequest(
	t *testing.T,
	privateKey ed25519.PrivateKey,
	keyID string,
	env *wire.IntentEnvelope,
	envJSON []byte,
) *http.Request {
	t.Helper()

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

	return req
}

func waitForR43ProvenanceEvidence(
	t *testing.T,
	rb *provenance.RingBuffer,
	path string,
) {
	t.Helper()

	deadline := time.Now().Add(time.Second)

	for time.Now().Before(deadline) {
		if rb.PendingCount() > 0 {
			return
		}

		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			return
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatal("real provenance writer showed no accepted or flushed event")
}

func TestR43H01ProvenanceAcceptedBeforeForward(t *testing.T) {
	const keyID = "r43-h01-key"

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}

	keyResolver := wire.NewStaticKeyResolver()
	if err := keyResolver.AddKey(keyID, publicKey); err != nil {
		t.Fatalf("AddKey() error = %v", err)
	}

	provenancePath := filepath.Join(
		t.TempDir(),
		"provenance.jsonl",
	)

	provenanceLog := provenance.NewRingBuffer(
		16,
		provenancePath,
	)
	defer provenanceLog.Stop()

	pipeline := evaluate.NewPipeline(
		keyResolver,
		evaluate.NewInMemoryRevocationStore(),
		budget.NewLedger(),
		evaluate.NewDefaultScopeGuard(),
		provenanceLog,
	)

	upstreamReached := make(chan struct{}, 1)

	upstream := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				// By the time forwarding begins, the real provenance writer
				// must already have accepted the ALLOW event. It may still
				// be pending or may already have been asynchronously flushed.
				waitForR43ProvenanceEvidence(
					t,
					provenanceLog,
					provenancePath,
				)

				upstreamReached <- struct{}{}
				w.WriteHeader(http.StatusOK)
			},
		),
	)
	defer upstream.Close()

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

	req := newR43SignedRequest(
		t,
		privateKey,
		keyID,
		env,
		envJSON,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status=%d decision=%q reason=%q body=%q, want HTTP 200",
			rec.Code,
			rec.Header().Get("X-HACP-Decision"),
			rec.Header().Get("X-HACP-Reason"),
			rec.Body.String(),
		)
	}

	if got := rec.Header().Get("X-HACP-Decision"); got != "ALLOW" {
		t.Fatalf("X-HACP-Decision=%q, want ALLOW", got)
	}

	select {
	case <-upstreamReached:
	// PASS
	default:
		t.Fatal("authorized request was not forwarded upstream")
	}
}

func TestR43H02FullProvenanceBufferFailsClosedWithoutForward(t *testing.T) {
	const keyID = "r43-h02-key"

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}

	keyResolver := wire.NewStaticKeyResolver()
	if err := keyResolver.AddKey(keyID, publicKey); err != nil {
		t.Fatalf("AddKey() error = %v", err)
	}

	// Zero capacity deterministically represents an already-full bounded
	// provenance sink. RingBuffer.Accept() must therefore fail immediately.
	provenanceLog := provenance.NewRingBuffer(
		0,
		filepath.Join(t.TempDir(), "provenance.jsonl"),
	)
	defer provenanceLog.Stop()

	pipeline := evaluate.NewPipeline(
		keyResolver,
		evaluate.NewInMemoryRevocationStore(),
		budget.NewLedger(),
		evaluate.NewDefaultScopeGuard(),
		provenanceLog,
	)

	upstreamReached := make(chan struct{}, 1)

	upstream := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				upstreamReached <- struct{}{}
				w.WriteHeader(http.StatusOK)
			},
		),
	)
	defer upstream.Close()

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

	req := newR43SignedRequest(
		t,
		privateKey,
		keyID,
		env,
		envJSON,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"status=%d decision=%q reason=%q body=%q, want HTTP 403",
			rec.Code,
			rec.Header().Get("X-HACP-Decision"),
			rec.Header().Get("X-HACP-Reason"),
			rec.Body.String(),
		)
	}

	if got := rec.Header().Get("X-HACP-Decision"); got != "DENY" {
		t.Fatalf("X-HACP-Decision=%q, want DENY", got)
	}

	if got := rec.Header().Get("X-HACP-Reason"); got != "TRACEABILITY_FAILURE" {
		t.Fatalf(
			"X-HACP-Reason=%q, want TRACEABILITY_FAILURE",
			got,
		)
	}

	select {
	case <-upstreamReached:
		t.Fatal("request was forwarded despite provenance acceptance failure")
	default:
		// PASS
	}
}
