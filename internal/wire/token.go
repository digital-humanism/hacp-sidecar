package wire

import (
	"encoding/json"
	"errors"
	"fmt"
)

// DecisionToken represents the HACP Decision Token schema
// per hacp-spec/schemas/decision_token.json
type DecisionToken struct {
	HACPVersion  string       `json:"hacp_version"`
	TokenID      string       `json:"token_id"`
	EnvelopeID   string       `json:"envelope_id"`
	ActionHash   string       `json:"action_hash"`
	PolicyDigest string       `json:"policy_digest"`
	Principal    string       `json:"principal"`
	SignerKeyID  string       `json:"signer_key_id"`
	IssuedAt     int64        `json:"issued_at"`
	ExpiresAt    int64        `json:"expires_at"`
	Decision     string       `json:"decision"`
	Constraints  *Constraints `json:"constraints,omitempty"`
	Signature    []byte       `json:"signature"` // base64url decoded

	// Computed fields
	rawJSON       json.RawMessage
	canonicalJSON []byte
}

type Constraints struct {
	Method      string `json:"method,omitempty"`
	Path        string `json:"path,omitempty"`
	ToolName    string `json:"tool_name,omitempty"`
	PayloadHash string `json:"payload_hash,omitempty"`
	MaxUses     *int   `json:"max_uses,omitempty"`
}

type tokenJSON struct {
	HACPVersion  string          `json:"hacp_version"`
	TokenID      string          `json:"token_id"`
	EnvelopeID   string          `json:"envelope_id"`
	ActionHash   string          `json:"action_hash"`
	PolicyDigest string          `json:"policy_digest"`
	Principal    string          `json:"principal"`
	SignerKeyID  string          `json:"signer_key_id"`
	IssuedAt     int64           `json:"issued_at"`
	ExpiresAt    int64           `json:"expires_at"`
	Decision     string          `json:"decision"`
	Constraints  *Constraints    `json:"constraints,omitempty"`
	Signature    string          `json:"signature"`
	RawJSON      json.RawMessage `json:"-"`
}

// ParseDecisionToken parses and validates a DecisionToken from JSON bytes
func ParseDecisionToken(data []byte) (*DecisionToken, error) {
	var tj tokenJSON

	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, fmt.Errorf("token JSON parse error: %w", err)
	}

	// Validate required fields
	if tj.HACPVersion == "" {
		return nil, errors.New("token missing hacp_version")
	}
	if tj.HACPVersion != "0.9" {
		return nil, fmt.Errorf("token unsupported version: %s", tj.HACPVersion)
	}
	if tj.TokenID == "" {
		return nil, errors.New("token missing token_id")
	}
	if tj.EnvelopeID == "" {
		return nil, errors.New("token missing envelope_id")
	}
	if tj.ActionHash == "" {
		return nil, errors.New("token missing action_hash")
	}
	if tj.PolicyDigest == "" {
		return nil, errors.New("token missing policy_digest")
	}
	if tj.Principal == "" {
		return nil, errors.New("token missing principal")
	}
	if tj.SignerKeyID == "" {
		return nil, errors.New("token missing signer_key_id")
	}
	if tj.IssuedAt == 0 {
		return nil, errors.New("token missing issued_at")
	}
	if tj.ExpiresAt == 0 {
		return nil, errors.New("token missing expires_at")
	}
	if tj.Decision == "" {
		return nil, errors.New("token missing decision")
	}
	if tj.Decision != "ALLOW" && tj.Decision != "DENY" && tj.Decision != "CHECKPOINT" {
		return nil, fmt.Errorf("token invalid decision: %s", tj.Decision)
	}
	if tj.Signature == "" {
		return nil, errors.New("token missing signature")
	}

	// Decode signature from base64url
	sigBytes, err := Base64URLDecode(tj.Signature)
	if err != nil {
		return nil, fmt.Errorf("token signature decode error: %w", err)
	}

	// Canonicalize JSON
	canonicalJSON, err := CanonicalizeJSON(data)
	if err != nil {
		return nil, fmt.Errorf("token canonicalization error: %w", err)
	}

	// Remove signature field for verification
	siglessJSON, err := RemoveSignatureField(data)
	if err != nil {
		return nil, fmt.Errorf("token sig removal error: %w", err)
	}
	siglessCanonical, err := CanonicalizeJSON(siglessJSON)
	if err != nil {
		return nil, fmt.Errorf("token sigless canonicalization error: %w", err)
	}

	tok := &DecisionToken{
		HACPVersion:   tj.HACPVersion,
		TokenID:       tj.TokenID,
		EnvelopeID:    tj.EnvelopeID,
		ActionHash:    tj.ActionHash,
		PolicyDigest:  tj.PolicyDigest,
		Principal:     tj.Principal,
		SignerKeyID:   tj.SignerKeyID,
		IssuedAt:      tj.IssuedAt,
		ExpiresAt:     tj.ExpiresAt,
		Decision:      tj.Decision,
		Constraints:   tj.Constraints,
		Signature:     sigBytes,
		canonicalJSON: canonicalJSON,
	}

	tok.rawJSON = siglessCanonical

	return tok, nil
}

// CanonicalPayload returns the canonicalized JSON without signature for verification
func (t *DecisionToken) CanonicalPayload() []byte {
	return t.rawJSON
}

// CanonicalHash returns the SHA-256 hash of the canonicalized token
func (t *DecisionToken) CanonicalHash() []byte {
	return SHA256(t.canonicalJSON)
}
