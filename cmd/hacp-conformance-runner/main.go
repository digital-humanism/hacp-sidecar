package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"hacp-sidecar/internal/budget"
	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/provenance"
	"hacp-sidecar/internal/wire"
)

// truncate returns the first n bytes of b, appending "..." if truncated.
// Used for debug logging only.
func truncate(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return append(b[:n], []byte("...")...)
}

// ProtocolVersion matches runner_protocol.md.
const ProtocolVersion = "1"

// Request represents an incoming runner protocol request.
// CRITICAL: use json.RawMessage to preserve original bytes.
type Request struct {
	ProtocolVersion string          `json:"protocol_version"`
	Operation       string          `json:"operation"`
	VectorID        string          `json:"vector_id"`
	Input           json.RawMessage `json:"input"`
}

// InputData holds parsed input fields.
// Envelope, Token, Action are kept as raw JSON to preserve canonical bytes.
type InputData struct {
	IntentEnvelope  json.RawMessage `json:"intent_envelope"`
	ProposedAction  json.RawMessage `json:"proposed_action"`
	DecisionToken   json.RawMessage `json:"decision_token"`
	PolicyContext   json.RawMessage `json:"policy_context"`
	Checkpoint      json.RawMessage `json:"checkpoint,omitempty"`
	ProvenanceEvent json.RawMessage `json:"provenance_event,omitempty"`
}

type policyContextJSON struct {
	Clock              int64    `json:"clock"`
	HumanRequired      bool     `json:"human_required"`
	HumanRequiredVerbs []string `json:"human_required_verbs"`
	ConsequenceClass   string   `json:"consequence_class"`
	RiskClass          string   `json:"risk_class"`

	Checkpoint json.RawMessage `json:"checkpoint,omitempty"`
}

type checkpointJSON struct {
	ID           string `json:"id"`
	CheckpointID string `json:"checkpoint_id"`

	State  string `json:"state"`
	Status string `json:"status"`

	ResolvedBy     string `json:"resolved_by"`
	ResolvedByKind string `json:"resolved_by_kind"`

	ResolverPrincipal     string `json:"resolver_principal"`
	ResolverPrincipalKind string `json:"resolver_principal_kind"`

	CreatedAt  int64 `json:"created_at"`
	ExpiresAt  int64 `json:"expires_at"`
	ResolvedAt int64 `json:"resolved_at"`
}

// Response represents an outgoing runner protocol response.
type Response struct {
	ProtocolVersion string                 `json:"protocol_version"`
	Decision        string                 `json:"decision"`
	ReasonCodes     []string               `json:"reason_codes"`
	ActionHash      string                 `json:"action_hash,omitempty"`
	ProvenanceID    string                 `json:"provenance_id,omitempty"`
	Metrics         map[string]interface{} `json:"metrics,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	Explanation     map[string]interface{} `json:"explanation,omitempty"`
}

// Runner holds all dependencies for the conformance runner.
type Runner struct {
	pipeline     *evaluate.Pipeline
	revocation   *evaluate.InMemoryRevocationStore
	keyResolver  *wire.StaticKeyResolver
	budgetLedger *budget.Ledger

	// Tracks the currently executing conformance vector.
	//
	// State such as token consumption must persist for multiple operations
	// belonging to the same vector, but MUST NOT leak into another vector.
	currentVectorID string
}

// NewRunner creates a runner with test configuration.
func NewRunner() (*Runner, error) {
	keyResolver := wire.NewStaticKeyResolver()

	if err := keyResolver.AddKeyFromHex(
		"key-ed25519-test-001",
		"9d17f1bbcc0845865e670f526413fb7a510380798fe300b6c98e28f3a3b0fdb3",
	); err != nil {
		return nil, fmt.Errorf("load test key: %w", err)
	}

	revocation := evaluate.NewInMemoryRevocationStore()
	budgetLedger := budget.NewLedger()
	scopeGuard := evaluate.NewDefaultScopeGuard()
	provLog := provenance.NewNoopWriter()

	pipeline := evaluate.NewPipeline(
		keyResolver,
		revocation,
		budgetLedger,
		scopeGuard,
		provLog,
	)

	return &Runner{
		pipeline:        pipeline,
		revocation:      revocation,
		keyResolver:     keyResolver,
		budgetLedger:    budgetLedger,
		currentVectorID: "",
	}, nil
}

// isJSONNull reports whether raw contains the JSON literal null
// or is effectively empty.
func isJSONNull(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)

	return len(raw) == 0 || bytes.Equal(raw, []byte("null"))
}

// prepareVectorState initializes state for a new conformance vector.
//
// IMPORTANT:
// State is reset only when VectorID changes.
// Multiple operations with the same VectorID therefore share state.
func (r *Runner) prepareVectorState(vectorID string) {
	if vectorID == "" {
		return
	}

	if r.currentVectorID == vectorID {
		return
	}

	r.budgetLedger.Reset()

	r.currentVectorID = vectorID
}

// HandleRequest processes a single runner protocol request.
func (r *Runner) HandleRequest(req Request) Response {
	start := time.Now()

	// Isolate mutable conformance state between vectors.
	r.prepareVectorState(req.VectorID)

	if req.ProtocolVersion != ProtocolVersion {
		return r.errorResponse(
			fmt.Sprintf(
				"unsupported protocol version: %s (expected %s)",
				req.ProtocolVersion,
				ProtocolVersion,
			),
		)
	}

	switch req.Operation {
	case "evaluate":
		return r.handleEvaluate(req, start)

	case "revoke":
		return r.handleRevoke(req)

	case "explain":
		return r.handleExplain(req)

	default:
		return r.errorResponse(
			fmt.Sprintf("unknown operation: %s", req.Operation),
		)
	}
}

// handleEvaluate processes an evaluate operation.
func (r *Runner) handleEvaluate(req Request, start time.Time) Response {
	// Parse input as InputData to preserve raw JSON bytes.
	var input InputData

	if err := json.Unmarshal(req.Input, &input); err != nil {
		return r.errorResponse(
			fmt.Sprintf("input parse: %v", err),
		)
	}

	if len(input.IntentEnvelope) == 0 {
		return r.errorResponse("missing intent_envelope")
	}

	// Policy preflight runs before full envelope parsing.
	// It extracts only the fields required to determine whether
	// a system principal must stop for human authorization.
	var envPreflight struct {
		PrincipalKind string `json:"principal_kind"`
		IssuedAt      int64  `json:"issued_at"`
	}

	if err := json.Unmarshal(input.IntentEnvelope, &envPreflight); err != nil {

		return Response{
			ProtocolVersion: ProtocolVersion,
			Decision:        "DENY",
			ReasonCodes:     []string{evaluate.ReasonInvalidEnvelope},
		}
	}

	var actionPreflight struct {
		Verb       string `json:"verb"`
		ProposedAt int64  `json:"proposed_at"`
	}

	if err := json.Unmarshal(input.ProposedAction, &actionPreflight); err != nil {

		return Response{
			ProtocolVersion: ProtocolVersion,
			Decision:        "DENY",
			ReasonCodes:     []string{evaluate.ReasonInvalidAction},
		}
	}

	var policyPreflight struct {
		HumanRequiredVerbs       []string `json:"human_required_verbs"`
		CurrentTime              int64    `json:"current_time"`
		CheckpointTimeoutSeconds int64    `json:"checkpoint_timeout_seconds"`
	}

	if len(input.PolicyContext) > 0 {
		if err := json.Unmarshal(input.PolicyContext, &policyPreflight); err != nil {

			return Response{
				ProtocolVersion: ProtocolVersion,
				Decision:        "DENY",
				ReasonCodes:     []string{evaluate.ReasonPolicyDenied},
			}
		}
	}

	humanRequired := false

	for _, verb := range policyPreflight.HumanRequiredVerbs {
		if verb == actionPreflight.Verb {
			humanRequired = true
			break
		}
	}

	if envPreflight.PrincipalKind == "system" && humanRequired {
		// If the action has already waited longer than the configured
		// human-approval timeout, the checkpoint may no longer be opened.
		if policyPreflight.CurrentTime > 0 &&
			policyPreflight.CheckpointTimeoutSeconds > 0 {

			waitStartedAt := actionPreflight.ProposedAt
			if waitStartedAt == 0 {
				waitStartedAt = envPreflight.IssuedAt
			}

			if waitStartedAt > 0 &&
				policyPreflight.CurrentTime > waitStartedAt+policyPreflight.CheckpointTimeoutSeconds {

				return Response{
					ProtocolVersion: ProtocolVersion,
					Decision:        "DENY",
					ReasonCodes:     []string{evaluate.ReasonCheckpointExpired},
				}
			}
		}

		return Response{
			ProtocolVersion: ProtocolVersion,
			Decision:        "CHECKPOINT",
			ReasonCodes:     []string{evaluate.ReasonHumanRequired},
		}
	}

	// Parse envelope from RAW bytes.
	env, err := wire.ParseIntentEnvelope(input.IntentEnvelope)
	if err != nil {

		return Response{
			ProtocolVersion: ProtocolVersion,
			Decision:        "DENY",
			ReasonCodes:     []string{evaluate.ReasonInvalidEnvelope},
		}
	}

	// Signature/canonicalization diagnostics.

	// Parse decision token from RAW bytes, if actually present.
	var tok *wire.DecisionToken

	if !isJSONNull(input.DecisionToken) {

		tok, err = wire.ParseDecisionToken(input.DecisionToken)
		if err != nil {

			return Response{
				ProtocolVersion: ProtocolVersion,
				Decision:        "DENY",
				ReasonCodes:     []string{evaluate.ReasonInvalidAction},
			}
		}

	} else {
	}

	// ============================================================
	// Parse policy_context
	// ============================================================

	var policy evaluate.PolicyContext

	if len(bytes.TrimSpace(input.PolicyContext)) > 0 &&
		!isJSONNull(input.PolicyContext) {

		var pc policyContextJSON

		if err := json.Unmarshal(input.PolicyContext, &pc); err != nil {

			return Response{
				ProtocolVersion: ProtocolVersion,
				Decision:        "DENY",
				ReasonCodes:     []string{evaluate.ReasonInvalidAction},
			}
		}

		policy.Clock = pc.Clock
		policy.HumanRequired = pc.HumanRequired
		policy.HumanRequiredVerbs = pc.HumanRequiredVerbs
		policy.ConsequenceClass = pc.ConsequenceClass
		policy.RiskClass = pc.RiskClass

	}
	// ============================================================
	// Parse checkpoint
	// ============================================================

	var checkpoint *evaluate.CheckpointContext

	checkpointRaw := input.Checkpoint

	// If no explicit checkpoint is provided, try to read it from
	// policy_context. This keeps the runner compatible with both
	// conformance-context representations.
	if (len(bytes.TrimSpace(checkpointRaw)) == 0 ||
		isJSONNull(checkpointRaw)) &&
		len(bytes.TrimSpace(input.PolicyContext)) > 0 &&
		!isJSONNull(input.PolicyContext) {

		var pc policyContextJSON

		if err := json.Unmarshal(input.PolicyContext, &pc); err == nil {
			if len(bytes.TrimSpace(pc.Checkpoint)) > 0 &&
				!isJSONNull(pc.Checkpoint) {

				checkpointRaw = pc.Checkpoint
			}
		}
	}

	if len(bytes.TrimSpace(checkpointRaw)) > 0 &&
		!isJSONNull(checkpointRaw) {

		var cp checkpointJSON

		if err := json.Unmarshal(checkpointRaw, &cp); err != nil {

			return Response{
				ProtocolVersion: ProtocolVersion,
				Decision:        "DENY",
				ReasonCodes:     []string{evaluate.ReasonCheckpointInvalid},
			}
		}

		checkpointID := cp.ID
		if checkpointID == "" {
			checkpointID = cp.CheckpointID
		}

		state := cp.State
		if state == "" {
			state = cp.Status
		}

		resolvedBy := cp.ResolvedBy
		if resolvedBy == "" {
			resolvedBy = cp.ResolverPrincipal
		}

		resolvedByKind := cp.ResolvedByKind
		if resolvedByKind == "" {
			resolvedByKind = cp.ResolverPrincipalKind
		}

		checkpoint = &evaluate.CheckpointContext{
			ID:             checkpointID,
			State:          evaluate.CheckpointState(state),
			ResolvedBy:     resolvedBy,
			ResolvedByKind: resolvedByKind,
			CreatedAt:      cp.CreatedAt,
			ExpiresAt:      cp.ExpiresAt,
			ResolvedAt:     cp.ResolvedAt,
		}

	}

	// ============================================================
	// Build request context
	// ============================================================

	reqCtx := &evaluate.RequestContext{
		Method:         "EVALUATE",
		Path:           "/conformance",
		RequestID:      req.VectorID,
		Timestamp:      time.Now(),
		Clock:          policy.Clock,
		ProposedAction: input.ProposedAction,
		Policy:         &policy,
		Checkpoint:     checkpoint,
	}

	// ============================================================
	// Evaluate
	// ============================================================

	decision := r.pipeline.Evaluate(
		context.Background(),
		env,
		tok,
		reqCtx,
	)

	latency := time.Since(start).Nanoseconds()

	// ============================================================
	// Build runner protocol response
	// ============================================================

	resp := Response{
		ProtocolVersion: ProtocolVersion,
		Decision:        string(decision.Outcome),
		ReasonCodes:     []string{},
		Metrics: map[string]interface{}{
			"latency_ns": latency,
		},
	}

	// Compatibility fallback for decisions produced by legacy code.
	if resp.Decision == "" {
		if decision.Allow {
			resp.Decision = "ALLOW"
		} else {
			resp.Decision = "DENY"
		}
	}

	if decision.ReasonCode != "" {
		resp.ReasonCodes = []string{
			decision.ReasonCode,
		}
	}

	// Return the canonical action hash when a valid token is present.
	if tok != nil {
		resp.ActionHash = tok.ActionHash
	}
	// Bind the evaluation response to the provenance event supplied
	// by the conformance vector.
	if !isJSONNull(input.ProvenanceEvent) {
		var prov struct {
			EventID string `json:"event_id"`
		}

		if err := json.Unmarshal(input.ProvenanceEvent, &prov); err != nil {
		} else if prov.EventID != "" {
			resp.ProvenanceID = prov.EventID

		}
	}

	return resp
}

// handleRevoke processes a revoke operation.
func (r *Runner) handleRevoke(req Request) Response {
	var input map[string]interface{}

	if err := json.Unmarshal(req.Input, &input); err != nil {
		return r.errorResponse(
			fmt.Sprintf("input parse: %v", err),
		)
	}

	targetID, _ := input["target_id"].(string)
	targetKind, _ := input["target_kind"].(string)

	if targetID == "" || targetKind == "" {
		return r.errorResponse(
			"missing target_id or target_kind",
		)
	}

	switch targetKind {
	case "token":
		r.revocation.RevokeToken(targetID)

	case "envelope":
		r.revocation.RevokeEnvelope(targetID)

	case "key":
		r.revocation.RevokeKey(targetID)

	default:
		return r.errorResponse(
			fmt.Sprintf(
				"unknown target_kind: %s",
				targetKind,
			),
		)
	}

	return Response{
		ProtocolVersion: ProtocolVersion,
		Decision:        "OK",
		ReasonCodes:     []string{},
	}
}

func (r *Runner) handleExplain(_ Request) Response {

	return Response{
		ProtocolVersion: ProtocolVersion,
		Decision:        "OK",
		ReasonCodes:     []string{},
		Explanation: map[string]interface{}{
			"human_readable": "Explanation not yet implemented",
			"policy_rules":   []string{},
		},
	}
}

func (r *Runner) errorResponse(msg string) Response {
	log.Printf(
		"[ERROR] runner internal error: %s",
		msg,
	)

	return Response{
		ProtocolVersion: ProtocolVersion,
		Decision:        "ERROR",
		ReasonCodes:     []string{"INTERNAL_ERROR"},
		ErrorMessage:    msg,
	}
}

func main() {
	// All diagnostic output MUST go to stderr.
	// stdout is reserved exclusively for runner-protocol JSON.
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	runner, err := NewRunner()
	if err != nil {
		log.Fatalf(
			"failed to initialize runner: %v",
			err,
		)
	}

	scanner := bufio.NewScanner(os.Stdin)

	// Increase maximum request size for conformance vectors.
	scanner.Buffer(
		make([]byte, 64*1024),
		16*1024*1024,
	)

	stdout := bufio.NewWriter(os.Stdout)

	log.Println(
		"hacp-conformance-runner started, protocol version:",
		ProtocolVersion,
	)

	for scanner.Scan() {
		line := scanner.Bytes()

		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var req Request

		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf(
				"malformed request: %v",
				err,
			)

			resp := runner.errorResponse(
				fmt.Sprintf(
					"malformed request: %v",
					err,
				),
			)

			if err := writeResponse(stdout, resp); err != nil {
				log.Printf(
					"failed to write malformed-request response: %v",
					err,
				)
				break
			}

			continue
		}

		resp := runner.HandleRequest(req)

		if err := writeResponse(stdout, resp); err != nil {
			break
		}

	}

	if err := scanner.Err(); err != nil {
		log.Printf(
			"stdin read error: %v",
			err,
		)
		os.Exit(3)
	}

	if err := stdout.Flush(); err != nil {
		log.Printf(
			"stdout final flush error: %v",
			err,
		)
		os.Exit(4)
	}

	log.Println(
		"hacp-conformance-runner shutting down (stdin closed)",
	)
}

func writeResponse(
	w *bufio.Writer,
	resp Response,
) error {

	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf(
			"marshal response: %w",
			err,
		)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf(
			"write response: %w",
			err,
		)
	}

	if err := w.WriteByte('\n'); err != nil {
		return fmt.Errorf(
			"write response newline: %w",
			err,
		)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf(
			"flush response: %w",
			err,
		)
	}

	return nil
}
