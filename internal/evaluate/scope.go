package evaluate

import (
	"strings"

	"hacp-sidecar/internal/wire"
)

// DefaultScopeGuard implements ScopeGuard with boundary matrix evaluation
type DefaultScopeGuard struct {
	// Add any configuration needed for boundary matrix
}

// NewDefaultScopeGuard creates a new default scope guard
func NewDefaultScopeGuard() *DefaultScopeGuard {
	return &DefaultScopeGuard{}
}

// MatchRequestConstraints checks if request matches token constraints
func (g *DefaultScopeGuard) MatchRequestConstraints(constraints *wire.Constraints, req *RequestContext) bool {
	if constraints == nil {
		// No constraints = any request allowed (but action_hash still must match)
		return true
	}

	// Check method
	if constraints.Method != "" && !strings.EqualFold(constraints.Method, req.Method) {
		return false
	}

	// Check path
	if constraints.Path != "" && constraints.Path != req.Path {
		return false
	}

	// Check tool_name
	if constraints.ToolName != "" && constraints.ToolName != req.ToolName {
		return false
	}

	// Check payload_hash
	if constraints.PayloadHash != "" && constraints.PayloadHash != req.PayloadHash {
		return false
	}

	return true
}

// CheckBoundary evaluates boundary matrix per hacp-spec/boundary-matrix.md
func (g *DefaultScopeGuard) CheckBoundary(scope *wire.ScopeGrant, req *RequestContext) bool {
	if scope == nil {
		return false
	}

	// Boundary matrix checks would go here
	// For MVP, we accept if scope is present
	// Full implementation would check:
	// - audience crossing (internal → external)
	// - reversibility changes
	// - externality changes
	// - data_class escalation
	// - quantity limits
	// - destination allowlist

	return true
}
