package main

import (
	"time"

	"hacp-sidecar/internal/controlplane"
)

// controlStateFreshness is the minimal runtime contract required by
// authorization readiness.
//
// It intentionally mirrors the evaluator-facing freshness contract without
// exposing control-plane mutation or transport operations.
type controlStateFreshness interface {
	IsFresh(time.Time) bool
}

// Compile-time verification that the distributed ControlState satisfies the
// authorization-readiness freshness contract.
var _ controlStateFreshness = (*controlplane.ControlState)(nil)

// makeControlStateReadiness adapts distributed control-state freshness into
// the readiness predicate consumed by /readyz.
//
// Missing state or clock dependencies fail closed.
func makeControlStateReadiness(
	state controlStateFreshness,
	now func() time.Time,
) func() bool {

	return func() bool {
		if state == nil || now == nil {
			return false
		}

		return state.IsFresh(now())
	}
}
