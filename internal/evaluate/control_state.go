package evaluate

import "time"

// ControlStateGuard is the minimal freshness contract used by the evaluator.
//
// Implementations may live in runtime/control-plane packages.
// evaluate intentionally does not depend on gRPC or controlplane internals.
type ControlStateGuard interface {
	IsFresh(time.Time) bool
}
