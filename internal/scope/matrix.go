package scope

import (
	"fmt"
)

// AttributeType defines the type of boundary attribute being checked.
type AttributeType string

const (
	AttrAudience      AttributeType = "audience"
	AttrReversibility AttributeType = "reversibility"
	AttrExternality   AttributeType = "externality"
	AttrDataClass     AttributeType = "data_class"
	AttrVerb          AttributeType = "verb"
	AttrResourceClass AttributeType = "resource_class"
)

// Action defines the outcome of a boundary transition.
type Action string

const (
	ActionAllow       Action = "ALLOW"
	ActionDeny        Action = "DENY"
	ActionCheckpoint  Action = "CHECKPOINT"
	ActionReauthorize Action = "REAUTHORIZE"
)

// BoundaryTransition defines a single transition rule in the boundary matrix.
// If an action transitions from 'From' to 'To' for the given 'Attribute',
// the specified 'Action' is taken.
type BoundaryTransition struct {
	Attribute AttributeType
	From      string
	To        string
	Action    Action
}

// DefaultMatrix is the data-driven boundary matrix for HACP.
// This is the normative reference for boundary crossing evaluation.
var DefaultMatrix = []BoundaryTransition{
	// --- Audience transitions ---
	{AttrAudience, "internal", "external", ActionReauthorize},
	{AttrAudience, "external", "internal", ActionAllow},

	// --- Reversibility transitions ---
	{AttrReversibility, "reversible", "irreversible", ActionReauthorize},
	{AttrReversibility, "irreversible", "reversible", ActionAllow},

	// --- Externality transitions ---
	{AttrExternality, "internal", "external", ActionReauthorize},
	{AttrExternality, "external", "internal", ActionAllow},

	// --- Data class transitions ---
	{AttrDataClass, "public", "internal", ActionAllow},
	{AttrDataClass, "internal", "confidential", ActionReauthorize},
	{AttrDataClass, "confidential", "internal", ActionAllow},
	{AttrDataClass, "internal", "public", ActionAllow},
	{AttrDataClass, "confidential", "public", ActionDeny}, // data leak

	// --- Verb transitions ---
	{AttrVerb, "read", "write", ActionReauthorize},
	{AttrVerb, "read", "delete", ActionReauthorize},
	{AttrVerb, "analysis", "action", ActionReauthorize},
	{AttrVerb, "write", "read", ActionAllow},
	{AttrVerb, "delete", "read", ActionAllow},

	// --- Resource class transitions ---
	{AttrResourceClass, "internal", "customer_record", ActionReauthorize},
	{AttrResourceClass, "customer_record", "internal", ActionAllow},
}

// TransitionKey uniquely identifies a transition in the matrix.
type TransitionKey struct {
	Attribute AttributeType
	From      string
	To        string
}

// matrixIndex provides fast lookup for transitions.
var matrixIndex map[TransitionKey]Action

func init() {
	matrixIndex = make(map[TransitionKey]Action, len(DefaultMatrix))
	for _, t := range DefaultMatrix {
		key := TransitionKey{
			Attribute: t.Attribute,
			From:      t.From,
			To:        t.To,
		}
		matrixIndex[key] = t.Action
	}
}

// EvaluateTransition checks if a boundary transition is allowed.
// Returns the action to take for the given transition.
func EvaluateTransition(attr AttributeType, from, to string) Action {
	// Same boundary → ALLOW
	if from == to {
		return ActionAllow
	}

	key := TransitionKey{
		Attribute: attr,
		From:      from,
		To:        to,
	}

	if action, ok := matrixIndex[key]; ok {
		return action
	}

	// Unknown transition → fail closed
	return ActionDeny
}

// EvaluateBoundaryCrossing checks if a proposed action crosses a boundary
// relative to the granted scope.
//
// scopeValues: the list of allowed values for the attribute in the scope grant
// proposedValue: the value in the proposed action
//
// Returns the action to take.
func EvaluateBoundaryCrossing(attr AttributeType, scopeValues []string, proposedValue string) Action {
	// Empty scope → fail closed (DENY)
	if len(scopeValues) == 0 {
		return ActionDeny
	}

	// Missing semantic field → fail closed (DENY)
	if proposedValue == "" {
		return ActionDeny
	}

	// If proposed value is within scope → ALLOW
	for _, v := range scopeValues {
		if v == proposedValue {
			return ActionAllow
		}
	}

	// Proposed value is outside scope → check matrix for each scope value
	// Take the most restrictive action across all possible transitions
	mostRestrictive := ActionAllow

	for _, scopeValue := range scopeValues {
		action := EvaluateTransition(attr, scopeValue, proposedValue)
		if isMoreRestrictive(action, mostRestrictive) {
			mostRestrictive = action
		}
	}

	return mostRestrictive
}

// isMoreRestrictive returns true if a is more restrictive than b.
// Order: ALLOW < CHECKPOINT < REAUTHORIZE < DENY
func isMoreRestrictive(a, b Action) bool {
	return actionRank(a) > actionRank(b)
}

func actionRank(a Action) int {
	switch a {
	case ActionAllow:
		return 0
	case ActionCheckpoint:
		return 1
	case ActionReauthorize:
		return 2
	case ActionDeny:
		return 3
	default:
		return 4 // Unknown → most restrictive
	}
}

// String returns a human-readable representation of the transition.
func (t BoundaryTransition) String() string {
	return fmt.Sprintf("%s: %s → %s = %s", t.Attribute, t.From, t.To, t.Action)
}
