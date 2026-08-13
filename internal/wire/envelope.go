package wire

import (
	"encoding/json"
	"errors"
	"fmt"
)

// IntentEnvelope represents the HACP Intent Envelope schema
// per hacp-spec/schemas/intent_envelope.json
type IntentEnvelope struct {
	HACPVersion     string      `json:"hacp_version"`
	EnvelopeID      string      `json:"envelope_id"`
	Principal       string      `json:"principal"`
	PrincipalKind   string      `json:"principal_kind"`
	IntentStatement string      `json:"intent_statement"`
	Scope           *ScopeGrant `json:"scope"`
	IssuedAt        int64       `json:"issued_at"`
	ExpiresAt       int64       `json:"expires_at"`
	SignerKeyID     string      `json:"signer_key_id"`
	Signature       []byte      `json:"signature"` // base64url decoded
	AutonomyBudget  *Budget     `json:"autonomy_budget,omitempty"`

	// Computed fields (not in JSON)
	rawJSON       json.RawMessage // original JSON for canonicalization
	canonicalJSON []byte          // RFC 8785 canonicalized
	actionHash    []byte          // SHA-256 of canonical ProposedAction
}

type ScopeGrant struct {
	Verbs           []string `json:"verbs"`
	ResourceClasses []string `json:"resource_classes"`
	Audiences       []string `json:"audiences"`
	Reversibility   []string `json:"reversibility"`
	Externality     []string `json:"externality"`
	DataClasses     []string `json:"data_classes"`
	MaxQuantity     *int     `json:"max_quantity,omitempty"`
	Destinations    []string `json:"destinations,omitempty"`
	ToolNames       []string `json:"tool_names,omitempty"`
}

type Budget struct {
	MaxActions   int    `json:"max_actions"`
	MaxRiskClass string `json:"max_risk_class,omitempty"`
}

// envelopeJSON is the internal JSON representation for parsing
type envelopeJSON struct {
	HACPVersion     string          `json:"hacp_version"`
	EnvelopeID      string          `json:"envelope_id"`
	Principal       string          `json:"principal"`
	PrincipalKind   string          `json:"principal_kind"`
	IntentStatement string          `json:"intent_statement"`
	Scope           *ScopeGrant     `json:"scope"`
	IssuedAt        int64           `json:"issued_at"`
	ExpiresAt       int64           `json:"expires_at"`
	SignerKeyID     string          `json:"signer_key_id"`
	Signature       string          `json:"signature"`
	AutonomyBudget  *Budget         `json:"autonomy_budget,omitempty"`
	RawJSON         json.RawMessage `json:"-"`
}

// ParseIntentEnvelope parses and validates an IntentEnvelope from JSON bytes
func ParseIntentEnvelope(data []byte) (*IntentEnvelope, error) {
	var ej envelopeJSON

	// First pass: parse to check structure
	if err := json.Unmarshal(data, &ej); err != nil {
		return nil, fmt.Errorf("envelope JSON parse error: %w", err)
	}

	// Validate required fields
	if ej.HACPVersion == "" {
		return nil, errors.New("envelope missing hacp_version")
	}
	if ej.HACPVersion != "0.9" {
		return nil, fmt.Errorf("envelope unsupported version: %s", ej.HACPVersion)
	}
	if ej.EnvelopeID == "" {
		return nil, errors.New("envelope missing envelope_id")
	}
	if ej.Principal == "" {
		return nil, errors.New("envelope missing principal")
	}
	if ej.PrincipalKind != "human" && ej.PrincipalKind != "system" {
		return nil, fmt.Errorf("envelope invalid principal_kind: %s", ej.PrincipalKind)
	}
	if ej.IntentStatement == "" {
		return nil, errors.New("envelope missing intent_statement")
	}
	if ej.Scope == nil {
		return nil, errors.New("envelope missing scope")
	}
	if ej.IssuedAt == 0 {
		return nil, errors.New("envelope missing issued_at")
	}
	if ej.ExpiresAt == 0 {
		return nil, errors.New("envelope missing expires_at")
	}
	if ej.SignerKeyID == "" {
		return nil, errors.New("envelope missing signer_key_id")
	}
	if ej.Signature == "" {
		return nil, errors.New("envelope missing signature")
	}

	// Decode signature from base64url
	sigBytes, err := Base64URLDecode(ej.Signature)
	if err != nil {
		return nil, fmt.Errorf("envelope signature decode error: %w", err)
	}

	// Canonicalize JSON (RFC 8785)
	canonicalJSON, err := CanonicalizeJSON(data)
	if err != nil {
		return nil, fmt.Errorf("envelope canonicalization error: %w", err)
	}

	// Remove signature field for verification
	siglessJSON, err := RemoveSignatureField(data)
	if err != nil {
		return nil, fmt.Errorf("envelope sig removal error: %w", err)
	}
	siglessCanonical, err := CanonicalizeJSON(siglessJSON)
	if err != nil {
		return nil, fmt.Errorf("envelope sigless canonicalization error: %w", err)
	}

	env := &IntentEnvelope{
		HACPVersion:     ej.HACPVersion,
		EnvelopeID:      ej.EnvelopeID,
		Principal:       ej.Principal,
		PrincipalKind:   ej.PrincipalKind,
		IntentStatement: ej.IntentStatement,
		Scope:           ej.Scope,
		IssuedAt:        ej.IssuedAt,
		ExpiresAt:       ej.ExpiresAt,
		SignerKeyID:     ej.SignerKeyID,
		Signature:       sigBytes,
		AutonomyBudget:  ej.AutonomyBudget,
		rawJSON:         data,
		canonicalJSON:   canonicalJSON,
	}

	// Store sigless canonical for signature verification
	env.rawJSON = siglessCanonical

	return env, nil
}

// CanonicalPayload returns the canonicalized JSON without signature for verification
func (e *IntentEnvelope) CanonicalPayload() []byte {
	return e.rawJSON
}

// CanonicalHash returns the SHA-256 hash of the canonicalized envelope
func (e *IntentEnvelope) CanonicalHash() []byte {
	return SHA256(e.canonicalJSON)
}
