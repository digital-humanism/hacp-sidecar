package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/provenance"
	"hacp-sidecar/internal/wire"
)

// Handler is the HTTP reverse proxy handler with HACP enforcement
type Handler struct {
	Pipeline      *evaluate.Pipeline
	ProvenanceLog *provenance.RingBuffer
	Upstream      string // upstream base URL (for MVP: just echo)
}

// NewHandler creates a new enforcement proxy handler
func NewHandler(pipeline *evaluate.Pipeline, prov *provenance.RingBuffer, upstream string) *Handler {
	return &Handler{
		Pipeline:      pipeline,
		ProvenanceLog: prov,
		Upstream:      upstream,
	}
}

// ServeHTTP implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := generateRequestID()

	// Step 1: Extract HACP headers
	env, tok, err := wire.ExtractHeaders(r)
	if err != nil {
		h.respondDeny(w, requestID, evaluate.ReasonInvalidEnvelope, err, nil, nil, start)
		return
	}

	// Step 2: Build request context
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondDeny(w, requestID, evaluate.ReasonInvalidAction, err, env, tok, start)
		return
	}
	r.Body = io.NopCloser(io.Reader(newBytesReader(bodyBytes))) // restore for forwarding

	payloadHash := wire.SHA256Hex(bodyBytes)
	reqCtx := &evaluate.RequestContext{
		Method:      r.Method,
		Path:        r.URL.Path,
		ToolName:    r.Header.Get("X-HACP-Tool-Name"), // optional deployment hint
		PayloadHash: payloadHash,
		Timestamp:   time.Now(),
		RequestID:   requestID,
		LatencyNs:   time.Since(start).Nanoseconds(),
	}

	// Step 3: Run evaluation pipeline
	decision := h.Pipeline.Evaluate(r.Context(), env, tok, reqCtx)

	if !decision.Allow {
		h.respondDeny(w, requestID, decision.ReasonCode, decision.Error, env, tok, start)
		return
	}

	// Step 4: Forward request upstream
	h.forwardUpstream(w, r, bodyBytes, requestID)

	latency := time.Since(start)
	log.Printf("ALLOW request_id=%s env=%s tok=%s latency=%s", requestID, env.EnvelopeID, tok.TokenID, latency)
}

// respondDeny sends a 403 with HACP diagnostic headers and records provenance
func (h *Handler) respondDeny(w http.ResponseWriter, requestID, reason string, err error, env *wire.IntentEnvelope, tok *wire.DecisionToken, start time.Time) {
	latency := time.Since(start)

	w.Header().Set("X-HACP-Decision", "DENY")
	w.Header().Set("X-HACP-Reason", reason)
	w.Header().Set("X-HACP-Request-Id", requestID)
	w.WriteHeader(http.StatusForbidden)

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	fmt.Fprintf(w, `{"decision":"DENY","reason":"%s","error":"%s","request_id":"%s"}`, reason, errMsg, requestID)

	// Record provenance (best-effort, do not fail the response)
	if env != nil && tok != nil {
		reqCtx := &evaluate.RequestContext{
			RequestID: requestID,
			LatencyNs: latency.Nanoseconds(),
		}
		if perr := h.ProvenanceLog.AcceptDenied(env, tok, reqCtx, reason); perr != nil {
			log.Printf("provenance record failed: %v", perr)
		}
	}

	log.Printf("DENY request_id=%s reason=%s err=%v latency=%s", requestID, reason, err, latency)
}

// forwardUpstream sends the request upstream.
// For MVP: we echo the request back with X-HACP-Decision: ALLOW.
// In production, this would proxy to the real upstream URL.
func (h *Handler) forwardUpstream(w http.ResponseWriter, r *http.Request, body []byte, requestID string) {
	w.Header().Set("X-HACP-Decision", "ALLOW")
	w.Header().Set("X-HACP-Request-Id", requestID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := map[string]interface{}{
		"forwarded":    true,
		"request_id":   requestID,
		"method":       r.Method,
		"path":         r.URL.Path,
		"payload_hash": wire.SHA256Hex(body),
		"upstream":     h.Upstream,
	}
	_ = writeJSON(w, resp)
}

// generateRequestID creates an opaque 16-byte hex request ID
func generateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "req-" + hex.EncodeToString(b)
}

// bytesReader helper
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func writeJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	return enc.Encode(v)
}
