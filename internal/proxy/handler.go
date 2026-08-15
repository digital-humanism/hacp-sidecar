package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/provenance"
	"hacp-sidecar/internal/wire"
)

// Handler is the HTTP reverse proxy handler with HACP enforcement.
type Handler struct {
	Pipeline      *evaluate.Pipeline
	ProvenanceLog *provenance.RingBuffer
	Upstream      string
	Client        *http.Client
}

// NewHandler creates a new HACP enforcement reverse proxy.
//
// A dedicated long-lived HTTP client is used for upstream communication.
// Its transport is shared across requests so persistent connections can
// be reused under concurrent workloads.
func NewHandler(
	pipeline *evaluate.Pipeline,
	prov *provenance.RingBuffer,
	upstream string,
) *Handler {

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,

		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 32,
		MaxConnsPerHost:     0,

		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		DisableKeepAlives: false,
		ForceAttemptHTTP2: false,
	}

	client := &http.Client{
		Transport: transport,
	}

	return &Handler{
		Pipeline:      pipeline,
		ProvenanceLog: prov,
		Upstream:      strings.TrimRight(upstream, "/"),
		Client:        client,
	}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	start := time.Now()
	requestID := generateRequestID()

	// ============================================================
	// 1. Extract HACP headers
	// ============================================================

	env, tok, err := wire.ExtractHeaders(r)

	if err != nil {
		h.respondDeny(
			w,
			requestID,
			evaluate.ReasonInvalidEnvelope,
			err,
			nil,
			nil,
			start,
		)
		return
	}

	// ============================================================
	// 2. Read request body
	// ============================================================

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		h.respondDeny(
			w,
			requestID,
			evaluate.ReasonInvalidAction,
			err,
			env,
			tok,
			start,
		)
		return
	}

	r.Body = io.NopCloser(
		newBytesReader(bodyBytes),
	)

	payloadHash := wire.SHA256Hex(bodyBytes)

	// ============================================================
	// 3. Synthesize proposed action
	// ============================================================

	proposedAction := SynthesizeProposedAction(
		r,
		env,
		payloadHash,
	)

	proposedActionJSON, err := json.Marshal(
		proposedAction,
	)

	if err != nil {
		h.respondDeny(
			w,
			requestID,
			evaluate.ReasonInvalidAction,
			err,
			env,
			tok,
			start,
		)
		return
	}

	// ============================================================
	// 4. Build evaluation context
	// ============================================================

	reqCtx := &evaluate.RequestContext{
		Method:      r.Method,
		Path:        r.URL.Path,
		ToolName:    r.Header.Get("X-HACP-Tool-Name"),
		PayloadHash: payloadHash,

		Timestamp: time.Now(),
		RequestID: requestID,

		LatencyNs: time.Since(start).Nanoseconds(),

		ProposedAction: proposedActionJSON,
	}

	// ============================================================
	// 5. HACP enforcement
	// ============================================================

	decision := h.Pipeline.Evaluate(
		r.Context(),
		env,
		tok,
		reqCtx,
	)

	if !decision.Allow {
		h.respondDeny(
			w,
			requestID,
			decision.ReasonCode,
			decision.Error,
			env,
			tok,
			start,
		)
		return
	}

	// ============================================================
	// 6. Forward authorized request
	// ============================================================

	if err := h.forwardUpstream(
		w,
		r,
		bodyBytes,
		requestID,
	); err != nil {
		return
	}

	// IMPORTANT:
	// No synchronous per-request ALLOW logging here.
	//
	// Gate D demonstrated that synchronous logging on every
	// successful request materially increases p99 tail latency.
}

// respondDeny sends a DENY response and records denied provenance.
func (h *Handler) respondDeny(
	w http.ResponseWriter,
	requestID string,
	reason string,
	err error,
	env *wire.IntentEnvelope,
	tok *wire.DecisionToken,
	start time.Time,
) {
	latency := time.Since(start)

	w.Header().Set(
		"X-HACP-Decision",
		"DENY",
	)

	w.Header().Set(
		"X-HACP-Reason",
		reason,
	)

	w.Header().Set(
		"X-HACP-Request-Id",
		requestID,
	)

	w.WriteHeader(
		http.StatusForbidden,
	)

	errMsg := ""

	if err != nil {
		errMsg = err.Error()
	}

	_, _ = fmt.Fprintf(
		w,
		`{"decision":"DENY","reason":"%s","error":"%s","request_id":"%s"}`,
		reason,
		errMsg,
		requestID,
	)

	// Record denied provenance best-effort.
	if env != nil && tok != nil {

		reqCtx := &evaluate.RequestContext{
			RequestID: requestID,
			LatencyNs: latency.Nanoseconds(),
		}

		if perr := h.ProvenanceLog.AcceptDenied(
			env,
			tok,
			reqCtx,
			reason,
		); perr != nil {

			log.Printf(
				"provenance record failed request_id=%s err=%v",
				requestID,
				perr,
			)
		}
	}

	// DENY is exceptional and remains synchronously logged.
	log.Printf(
		"DENY request_id=%s reason=%s err=%v latency=%s",
		requestID,
		reason,
		err,
		latency,
	)
}

// forwardUpstream forwards an already-authorized request to the real
// upstream service.
func (h *Handler) forwardUpstream(
	w http.ResponseWriter,
	r *http.Request,
	body []byte,
	requestID string,
) error {

	upstreamURL :=
		h.Upstream +
			r.URL.RequestURI()

	upstreamReq, err := http.NewRequestWithContext(
		r.Context(),
		r.Method,
		upstreamURL,
		newBytesReader(body),
	)

	if err != nil {

		log.Printf(
			"upstream request creation failed request_id=%s err=%v",
			requestID,
			err,
		)

		writeUpstreamError(
			w,
			requestID,
			"failed to create upstream request",
		)

		return err
	}

	// Client -> Sidecar and Sidecar -> Upstream are separate HTTP
	// transport hops. Only end-to-end headers are propagated.
	copyEndToEndHeaders(
		upstreamReq.Header,
		r.Header,
	)

	resp, err := h.Client.Do(
		upstreamReq,
	)

	if err != nil {

		log.Printf(
			"upstream request failed request_id=%s upstream=%s err=%v",
			requestID,
			upstreamURL,
			err,
		)

		writeUpstreamError(
			w,
			requestID,
			"upstream unavailable",
		)

		return err
	}

	defer resp.Body.Close()

	// Reading the complete body ensures the HTTP transport can
	// safely return the persistent connection to its idle pool.
	responseBody, err := io.ReadAll(
		resp.Body,
	)

	if err != nil {

		log.Printf(
			"upstream response read failed request_id=%s err=%v",
			requestID,
			err,
		)

		return err
	}

	copyEndToEndHeaders(
		w.Header(),
		resp.Header,
	)

	w.Header().Set(
		"X-HACP-Decision",
		"ALLOW",
	)

	w.Header().Set(
		"X-HACP-Request-Id",
		requestID,
	)

	w.WriteHeader(
		resp.StatusCode,
	)

	if _, err := w.Write(
		responseBody,
	); err != nil {

		log.Printf(
			"upstream response write failed request_id=%s err=%v",
			requestID,
			err,
		)

		return err
	}

	return nil
}

// copyEndToEndHeaders copies HTTP headers while removing hop-by-hop
// transport headers.
func copyEndToEndHeaders(
	dst http.Header,
	src http.Header,
) {

	hopByHop := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Connection":    true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailer":             true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}

	// Connection may nominate additional hop-by-hop headers.
	if connection := src.Get("Connection"); connection != "" {

		for _, name := range strings.Split(
			connection,
			",",
		) {
			name = strings.TrimSpace(name)

			if name == "" {
				continue
			}

			hopByHop[http.CanonicalHeaderKey(name)] = true
		}
	}

	for name, values := range src {

		canonicalName :=
			http.CanonicalHeaderKey(name)

		if hopByHop[canonicalName] {
			continue
		}

		for _, value := range values {

			dst.Add(
				canonicalName,
				value,
			)
		}
	}
}

func writeUpstreamError(
	w http.ResponseWriter,
	requestID string,
	message string,
) {

	w.Header().Set(
		"X-HACP-Decision",
		"ALLOW",
	)

	w.Header().Set(
		"X-HACP-Request-Id",
		requestID,
	)

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(
		http.StatusBadGateway,
	)

	_, _ = fmt.Fprintf(
		w,
		`{"error":"%s","request_id":"%s"}`,
		message,
		requestID,
	)
}

// generateRequestID creates an opaque request identifier.
func generateRequestID() string {

	b := make(
		[]byte,
		16,
	)

	if _, err := rand.Read(b); err != nil {

		// rand.Read failure is exceptionally unlikely.
		// Fall back to timestamp-based opaque ID rather than failing
		// request processing.
		return fmt.Sprintf(
			"req-%d",
			time.Now().UnixNano(),
		)
	}

	return "req-" +
		hex.EncodeToString(b)
}

// bytesReader is a minimal reader for an already-buffered request body.
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(
	data []byte,
) *bytesReader {

	return &bytesReader{
		data: data,
	}
}

func (r *bytesReader) Read(
	p []byte,
) (int, error) {

	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	n := copy(
		p,
		r.data[r.pos:],
	)

	r.pos += n

	return n, nil
}
