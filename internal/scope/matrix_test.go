package scope

import (
	"testing"
)

func TestEvaluateTransition_SameBoundary(t *testing.T) {
	tests := []struct {
		attr AttributeType
		from string
		to   string
		want Action
	}{
		{AttrAudience, "internal", "internal", ActionAllow},
		{AttrAudience, "external", "external", ActionAllow},
		{AttrReversibility, "reversible", "reversible", ActionAllow},
		{AttrVerb, "read", "read", ActionAllow},
	}

	for _, tt := range tests {
		got := EvaluateTransition(tt.attr, tt.from, tt.to)
		if got != tt.want {
			t.Errorf("EvaluateTransition(%s, %s, %s) = %s, want %s",
				tt.attr, tt.from, tt.to, got, tt.want)
		}
	}
}

func TestEvaluateTransition_KnownTransitions(t *testing.T) {
	tests := []struct {
		attr AttributeType
		from string
		to   string
		want Action
	}{
		// Audience
		{AttrAudience, "internal", "external", ActionReauthorize},
		{AttrAudience, "external", "internal", ActionAllow},

		// Reversibility
		{AttrReversibility, "reversible", "irreversible", ActionReauthorize},
		{AttrReversibility, "irreversible", "reversible", ActionAllow},

		// Externality
		{AttrExternality, "internal", "external", ActionReauthorize},
		{AttrExternality, "external", "internal", ActionAllow},

		// Data class
		{AttrDataClass, "public", "internal", ActionAllow},
		{AttrDataClass, "internal", "confidential", ActionReauthorize},
		{AttrDataClass, "confidential", "public", ActionDeny},

		// Verb
		{AttrVerb, "read", "write", ActionReauthorize},
		{AttrVerb, "analysis", "action", ActionReauthorize},
	}

	for _, tt := range tests {
		got := EvaluateTransition(tt.attr, tt.from, tt.to)
		if got != tt.want {
			t.Errorf("EvaluateTransition(%s, %s, %s) = %s, want %s",
				tt.attr, tt.from, tt.to, got, tt.want)
		}
	}
}

func TestEvaluateTransition_UnknownTransition(t *testing.T) {
	tests := []struct {
		attr AttributeType
		from string
		to   string
	}{
		{AttrAudience, "internal", "unknown_value"},
		{AttrAudience, "unknown_value", "external"},
		{AttrDataClass, "top_secret", "public"},
		{AttrVerb, "teleport", "read"},
	}

	for _, tt := range tests {
		got := EvaluateTransition(tt.attr, tt.from, tt.to)
		if got != ActionDeny {
			t.Errorf("EvaluateTransition(%s, %s, %s) = %s, want DENY (fail closed)",
				tt.attr, tt.from, tt.to, got)
		}
	}
}

func TestEvaluateBoundaryCrossing_WithinScope(t *testing.T) {
	tests := []struct {
		attr          AttributeType
		scopeValues   []string
		proposedValue string
		want          Action
	}{
		// Proposed value is in scope → ALLOW
		{AttrAudience, []string{"internal", "external"}, "internal", ActionAllow},
		{AttrAudience, []string{"internal", "external"}, "external", ActionAllow},
		{AttrReversibility, []string{"reversible"}, "reversible", ActionAllow},
		{AttrVerb, []string{"read", "write"}, "read", ActionAllow},
	}

	for _, tt := range tests {
		got := EvaluateBoundaryCrossing(tt.attr, tt.scopeValues, tt.proposedValue)
		if got != tt.want {
			t.Errorf("EvaluateBoundaryCrossing(%s, %v, %s) = %s, want %s",
				tt.attr, tt.scopeValues, tt.proposedValue, got, tt.want)
		}
	}
}

func TestEvaluateBoundaryCrossing_OutsideScope(t *testing.T) {
	tests := []struct {
		attr          AttributeType
		scopeValues   []string
		proposedValue string
		want          Action
	}{
		// Scope: internal, Proposed: external → REAUTHORIZE
		{AttrAudience, []string{"internal"}, "external", ActionReauthorize},

		// Scope: reversible, Proposed: irreversible → REAUTHORIZE
		{AttrReversibility, []string{"reversible"}, "irreversible", ActionReauthorize},

		// Scope: internal, Proposed: external (externality) → REAUTHORIZE
		{AttrExternality, []string{"internal"}, "external", ActionReauthorize},

		// Scope: confidential, Proposed: public → DENY (data leak)
		{AttrDataClass, []string{"confidential"}, "public", ActionDeny},

		// Scope: read, Proposed: write → REAUTHORIZE
		{AttrVerb, []string{"read"}, "write", ActionReauthorize},
	}

	for _, tt := range tests {
		got := EvaluateBoundaryCrossing(tt.attr, tt.scopeValues, tt.proposedValue)
		if got != tt.want {
			t.Errorf("EvaluateBoundaryCrossing(%s, %v, %s) = %s, want %s",
				tt.attr, tt.scopeValues, tt.proposedValue, got, tt.want)
		}
	}
}

func TestEvaluateBoundaryCrossing_EmptyScope(t *testing.T) {
	// Empty scope → fail closed (DENY)
	tests := []struct {
		attr          AttributeType
		scopeValues   []string
		proposedValue string
	}{
		{AttrAudience, []string{}, "internal"},
		{AttrAudience, nil, "internal"},
		{AttrVerb, []string{}, "read"},
	}

	for _, tt := range tests {
		got := EvaluateBoundaryCrossing(tt.attr, tt.scopeValues, tt.proposedValue)
		if got != ActionDeny {
			t.Errorf("EvaluateBoundaryCrossing(%s, %v, %s) = %s, want DENY (empty scope)",
				tt.attr, tt.scopeValues, tt.proposedValue, got)
		}
	}
}

func TestEvaluateBoundaryCrossing_MissingField(t *testing.T) {
	// Missing semantic field → fail closed (DENY)
	got := EvaluateBoundaryCrossing(AttrAudience, []string{"internal"}, "")
	if got != ActionDeny {
		t.Errorf("EvaluateBoundaryCrossing with empty proposedValue = %s, want DENY", got)
	}
}

func TestIsMoreRestrictive(t *testing.T) {
	tests := []struct {
		a, b Action
		want bool
	}{
		{ActionDeny, ActionAllow, true},
		{ActionReauthorize, ActionAllow, true},
		{ActionCheckpoint, ActionAllow, true},
		{ActionDeny, ActionReauthorize, true},
		{ActionReauthorize, ActionCheckpoint, true},
		{ActionAllow, ActionDeny, false},
		{ActionAllow, ActionAllow, false},
	}

	for _, tt := range tests {
		got := isMoreRestrictive(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("isMoreRestrictive(%s, %s) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
