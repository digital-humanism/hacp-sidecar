package proxy

import (
	"encoding/json"
	"net/http"

	"hacp-sidecar/internal/wire"
)

// ProposedAction is the canonical form of a proposed action.
// The JSON shape is stable: it is what token.action_hash binds to.
type ProposedAction struct {
	HACPVersion   string `json:"hacp_version"`
	Verb          string `json:"verb"`
	ResourceClass string `json:"resource_class"`
	ResourceID    string `json:"resource_id"`
	Audience      string `json:"audience"`
	Reversibility string `json:"reversibility"`
	Externality   string `json:"externality"`
	DataClass     string `json:"data_class"`
	PayloadHash   string `json:"payload_hash"`
}

// SynthesizeProposedAction builds a ProposedAction from an HTTP request
// and the granted scope of the envelope.
//
// Verb is derived from HTTP method. Boundary attributes are taken from the
// first element of each scope list — a proxy cannot know the exact attribute
// the client intends; it uses the granted values, and the boundary matrix
// enforces whether the verb is allowed for that combination.
func SynthesizeProposedAction(r *http.Request, env *wire.IntentEnvelope, payloadHash string) *ProposedAction {
	return &ProposedAction{
		HACPVersion:   "0.9",
		Verb:          HTTPMethodToVerb(r.Method),
		ResourceClass: firstOrEmpty(env.Scope.ResourceClasses),
		ResourceID:    r.URL.Path,
		Audience:      firstOrEmpty(env.Scope.Audiences),
		Reversibility: firstOrEmpty(env.Scope.Reversibility),
		Externality:   firstOrEmpty(env.Scope.Externality),
		DataClass:     firstOrEmpty(env.Scope.DataClasses),
		PayloadHash:   payloadHash,
	}
}

// HTTPMethodToVerb maps an HTTP method to a HACP verb.
func HTTPMethodToVerb(method string) string {
	switch method {
	case "GET", "HEAD":
		return "read"
	case "POST":
		return "write"
	case "PUT", "PATCH":
		return "write"
	case "DELETE":
		return "delete"
	default:
		return "unknown"
	}
}

// MarshalCanonical returns the canonicalized JSON bytes of the proposed action.
func (p *ProposedAction) MarshalCanonical() ([]byte, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return wire.CanonicalizeJSON(raw)
}

// Hash returns SHA-256 of the canonical form — the action_hash for token binding.
func (p *ProposedAction) Hash() (string, error) {
	canonical, err := p.MarshalCanonical()
	if err != nil {
		return "", err
	}
	return wire.SHA256Hex(canonical), nil
}

func firstOrEmpty(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
