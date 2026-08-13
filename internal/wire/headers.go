package wire

import (
	"fmt"
	"net/http"
)

const (
	HeaderIntentEnvelope = "X-HACP-Intent-Envelope"
	HeaderDecisionToken  = "X-HACP-Decision-Token"
	MaxHeaderSize        = 8 * 1024 // 8 KB per wire/encoding.md
)

// ExtractHeaders extracts and decodes HACP headers from an HTTP request
func ExtractHeaders(r *http.Request) (*IntentEnvelope, *DecisionToken, error) {
	// Extract envelope header
	envHeader := r.Header.Get(HeaderIntentEnvelope)
	if envHeader == "" {
		return nil, nil, fmt.Errorf("missing %s header", HeaderIntentEnvelope)
	}

	// Check size limit
	if len(envHeader) > MaxHeaderSize {
		return nil, nil, fmt.Errorf("envelope header too large: %d > %d", len(envHeader), MaxHeaderSize)
	}

	// Decode base64url
	envBytes, err := Base64URLDecode(envHeader)
	if err != nil {
		return nil, nil, fmt.Errorf("envelope header decode error: %w", err)
	}

	// Parse envelope
	env, err := ParseIntentEnvelope(envBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("envelope parse error: %w", err)
	}

	// Extract token header
	tokHeader := r.Header.Get(HeaderDecisionToken)
	if tokHeader == "" {
		return nil, nil, fmt.Errorf("missing %s header", HeaderDecisionToken)
	}

	// Check size limit
	if len(tokHeader) > MaxHeaderSize {
		return nil, nil, fmt.Errorf("token header too large: %d > %d", len(tokHeader), MaxHeaderSize)
	}

	// Decode base64url
	tokBytes, err := Base64URLDecode(tokHeader)
	if err != nil {
		return nil, nil, fmt.Errorf("token header decode error: %w", err)
	}

	// Parse token
	tok, err := ParseDecisionToken(tokBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("token parse error: %w", err)
	}

	return env, tok, nil
}
