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

	if constraints.Path != "" && !equalRequestTarget(constraints.Path, req.Path) {
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
// Returns whether the request is inside the granted envelope scope and,
// on failure, the normative primary reason code.
func (g *DefaultScopeGuard) CheckBoundary(
	scopeGrant *wire.ScopeGrant,
	req *RequestContext,
) (bool, string) {
	if scopeGrant == nil {
		return false, ReasonScopeExceeded
	}

	// If no proposed action is available, skip boundary check
	// (this handles HTTP proxy mode without proposed_action)
	// Note: len() for nil slices is defined as zero, so nil check is redundant.
	if len(req.ProposedAction) == 0 {
		return true, ""
	}

	// Parse proposed action attributes
	attrs, err := scope.ParseProposedActionAttributes(req.ProposedAction)
	if err != nil {
		return false, ReasonScopeExceeded
	}

	if attrs == nil {
		return true, ""
	}

	// Check each attribute against the boundary matrix.
	// Existing reason-code behavior is preserved here; this review
	// changes only the proven tool_name allowlist violation.
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
		action := scope.EvaluateBoundaryCrossing(
			check.attr,
			check.scopeValues,
			check.proposedValue,
		)

		if action != scope.ActionAllow {
			return false, ReasonScopeExceeded
		}
	}

	if len(scopeGrant.ToolNames) > 0 {
		allowed := false

		for _, toolName := range scopeGrant.ToolNames {
			if toolName == req.ToolName {
				allowed = true
				break
			}
		}

		if !allowed {
			return false, ReasonBoundaryCrossing
		}
	}

	return true, ""
}

func equalRequestTarget(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] == '%' && b[i] == '%' && i+2 < len(a) &&
			isHex(a[i+1]) && isHex(a[i+2]) &&
			isHex(b[i+1]) && isHex(b[i+2]) {

			if !equalHex(a[i+1], b[i+1]) ||
				!equalHex(a[i+2], b[i+2]) {
				return false
			}

			i += 2
			continue
		}

		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' ||
		c >= 'a' && c <= 'f' ||
		c >= 'A' && c <= 'F'
}

func equalHex(a, b byte) bool {
	if a >= 'A' && a <= 'F' {
		a += 'a' - 'A'
	}
	if b >= 'A' && b <= 'F' {
		b += 'a' - 'A'
	}
	return a == b
}
