package trust_test

import (
	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/trust"
	"hacp-sidecar/internal/wire"
)

// Compile-time compatibility assertions.
// AtomicTrustStore must remain a drop-in resolver for both existing
// resolver interfaces without either interface importing trust lifecycle code.
var _ evaluate.KeyResolver = (*trust.AtomicTrustStore)(nil)
var _ wire.KeyResolver = (*trust.AtomicTrustStore)(nil)
