package evaluate

import (
	"strings"

	"hacp-sidecar/internal/scope"
	"hacp-sidecar/internal/wire"
)

// DefaultScopeGuard implements ScopeGuard with boundary matrix evaluation.
type DefaultScopeGuard struct{}

// NewDefaultScopeGuard creates a new default scope guard.
func NewDefaultScopeGuard() *DefaultScopeGuard {
	return &DefaultScopeGuard{}
}

// MatchRequestConstraints checks if request matches token constraints.
func (g *DefaultScopeGuard) MatchRequestConstraints(constraints *wire.Constraints, req *RequestContext) bool {
	if constraints == nil {
		return true
	}

	if constraints.Method != "" && !strings.EqualFold(constraints.Method, req.Method) {
		return false
	}

	if constraints.Path != "" && constraints.Path != req.Path {
		return false
	}

	if constraints.ToolName != "" && constraints.ToolName != req.ToolName {
		return false
	}

	if constraints.PayloadHash != "" && constraints.PayloadHash != req.PayloadHash {
		return false
	}

	return true
}

// CheckBoundary evaluates boundary matrix per hacp-spec/boundary-matrix.md.
// Returns false if any boundary crossing results in DENY.
// For MVP, CHECKPOINT and REAUTHORIZE are treated as ALLOW (future: proper handling).
func (g *DefaultScopeGuard) CheckBoundary(scopeGrant *wire.ScopeGrant, req *RequestContext) bool {
	if scopeGrant == nil {
		return false
	}

	// If no proposed action is available, skip boundary check
	// (this handles HTTP proxy mode without proposed_action)
	// Note: len() for nil slices is defined as zero, so nil check is redundant.
	if len(req.ProposedAction) == 0 {
		return true
	}

	// Parse proposed action attributes
	attrs, err := scope.ParseProposedActionAttributes(req.ProposedAction)
	if err != nil {
		return false // Fail closed on parse error
	}

	if attrs == nil {
		return true // No attributes to check
	}

	// Check each attribute against the boundary matrix
	checks := []struct {
		attr          scope.AttributeType
		scopeValues   []string
		proposedValue string
	}{
		{scope.AttrAudience, scopeGrant.Audiences, attrs.Audience},
		{scope.AttrReversibility, scopeGrant.Reversibility, attrs.Reversibility},
		{scope.AttrExternality, scopeGrant.Externality, attrs.Externality},
		{scope.AttrDataClass, scopeGrant.DataClasses, attrs.DataClass},
		{scope.AttrVerb, scopeGrant.Verbs, attrs.Verb},
		{scope.AttrResourceClass, scopeGrant.ResourceClasses, attrs.ResourceClass},
	}

	for _, check := range checks {
		action := scope.EvaluateBoundaryCrossing(check.attr, check.scopeValues, check.proposedValue)
		if action == scope.ActionDeny {
			return false
		}
		// For ALLOW, CHECKPOINT, REAUTHORIZE → continue
		// Future: return Action instead of bool for proper CHECKPOINT/REAUTHORIZE handling
	}

	return true
}
