package provenance

import (
	"hacp-sidecar/internal/evaluate"
	"hacp-sidecar/internal/wire"
)

// NoopWriter is a provenance writer that discards all events.
// Used for conformance runners where provenance is not under test.
type NoopWriter struct{}

// NewNoopWriter creates a noop provenance writer
func NewNoopWriter() *NoopWriter {
	return &NoopWriter{}
}

// Accept implements evaluate.ProvenanceWriter
func (n *NoopWriter) Accept(env *wire.IntentEnvelope, tok *wire.DecisionToken, req *evaluate.RequestContext) error {
	// Always accept (never fails)
	return nil
}
