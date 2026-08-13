package wire

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
)

// VerifyEd25519 verifies an Ed25519 signature (RFC 8032, pure mode)
func VerifyEd25519(publicKey ed25519.PublicKey, message, signature []byte) bool {
	log.Printf("[DEBUG VerifyEd25519] publicKey len=%d hex=%x", len(publicKey), publicKey)
	log.Printf("[DEBUG VerifyEd25519] message len=%d sha256=%x", len(message), sha256.Sum256(message))
	log.Printf("[DEBUG VerifyEd25519] signature len=%d hex=%x", len(signature), signature)

	if len(publicKey) != ed25519.PublicKeySize {
		log.Printf("[DEBUG VerifyEd25519] FAIL: publicKey size mismatch (expected %d, got %d)", ed25519.PublicKeySize, len(publicKey))
		return false
	}
	if len(signature) != ed25519.SignatureSize {
		log.Printf("[DEBUG VerifyEd25519] FAIL: signature size mismatch (expected %d, got %d)", ed25519.SignatureSize, len(signature))
		return false
	}

	result := ed25519.Verify(publicKey, message, signature)
	log.Printf("[DEBUG VerifyEd25519] ed25519.Verify result=%v", result)
	return result
}

// SHA256 computes SHA-256 hash and returns raw bytes
func SHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// SHA256Hex computes SHA-256 hash and returns lowercase hex string
func SHA256Hex(data []byte) string {
	hash := SHA256(data)
	return hex.EncodeToString(hash)
}

// Base64URLDecode decodes base64url without padding
func Base64URLDecode(s string) ([]byte, error) {
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

// Base64URLEncode encodes to base64url without padding
func Base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// ParseEd25519PublicKey parses a hex-encoded Ed25519 public key
func ParseEd25519PublicKey(hexKey string) (ed25519.PublicKey, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}

	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", ed25519.PublicKeySize, len(keyBytes))
	}

	return ed25519.PublicKey(keyBytes), nil
}
