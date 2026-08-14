package scope

import (
	"encoding/json"
)

// ProposedActionAttributes holds the boundary-relevant attributes
// extracted from a proposed action JSON.
type ProposedActionAttributes struct {
	Verb          string
	ResourceClass string
	Audience      string
	Reversibility string
	Externality   string
	DataClass     string
}

// ParseProposedActionAttributes extracts boundary-relevant attributes
// from raw proposed action JSON bytes.
func ParseProposedActionAttributes(data []byte) (*ProposedActionAttributes, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var action struct {
		Verb          string `json:"verb"`
		ResourceClass string `json:"resource_class"`
		Audience      string `json:"audience"`
		Reversibility string `json:"reversibility"`
		Externality   string `json:"externality"`
		DataClass     string `json:"data_class"`
	}

	if err := json.Unmarshal(data, &action); err != nil {
		return nil, err
	}

	return &ProposedActionAttributes{
		Verb:          action.Verb,
		ResourceClass: action.ResourceClass,
		Audience:      action.Audience,
		Reversibility: action.Reversibility,
		Externality:   action.Externality,
		DataClass:     action.DataClass,
	}, nil
}
