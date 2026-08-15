package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"hacp-sidecar/internal/proxy"
	"hacp-sidecar/internal/wire"
)

// Generate N pre-signed tokens for load testing.
// Each token has a unique token_id but identical action_hash (same proposed action).
// This simulates a single user making repeated requests with the same scope.

func main() {
	count := flag.Int("count", 1000, "number of tokens to generate")
	outFile := flag.String("out", "tokens.jsonl", "output file (JSONL)")
	method := flag.String("method", "GET", "HTTP method")
	path := flag.String("path", "/api/test", "request path")
	flag.Parse()

	// Test key
	seedInput := []byte("hacp-conformance-v0.9-key-001")
	h := sha256.Sum256(seedInput)
	privKey := ed25519.NewKeyFromSeed(h[:])
	keyID := "key-ed25519-test-001"

	now := time.Now().Unix()
	expires := now + 7200 // 2 hours

	// Fixed scope for all tokens
	scope := map[string]interface{}{
		"verbs":            []string{"read"},
		"resource_classes": []string{"customer_record"},
		"audiences":        []string{"internal"},
		"reversibility":    []string{"reversible"},
		"externality":      []string{"internal"},
		"data_classes":     []string{"internal"},
	}

	// Fixed envelope (same for all tokens)
	envelope := map[string]interface{}{
		"hacp_version":     "0.9",
		"envelope_id":      "22222222-2222-2222-2222-222222222222",
		"principal":        "human_admin_01",
		"principal_kind":   "human",
		"intent_statement": "Benchmark envelope",
		"scope":            scope,
		"issued_at":        now,
		"expires_at":       expires,
		"signer_key_id":    keyID,
	}

	envJSON, _ := json.Marshal(envelope)
	envCanonical, _ := wire.CanonicalizeJSON(envJSON)
	envSig := ed25519.Sign(privKey, envCanonical)
	envelope["signature"] = wire.Base64URLEncode(envSig)
	envFinalJSON, _ := json.Marshal(envelope)
	envHeader := wire.Base64URLEncode(envFinalJSON)

	// Fixed proposed action (same for all tokens)
	payloadHash := wire.SHA256Hex(nil) // empty body
	pa := &proxy.ProposedAction{
		HACPVersion:   "0.9",
		Verb:          proxy.HTTPMethodToVerb(*method),
		ResourceClass: "customer_record",
		ResourceID:    *path,
		Audience:      "internal",
		Reversibility: "reversible",
		Externality:   "internal",
		DataClass:     "internal",
		PayloadHash:   payloadHash,
	}
	actionHash, _ := pa.Hash()

	// Open output file
	f, err := os.Create(*outFile)
	if err != nil {
		log.Fatalf("create output: %v", err)
	}
	defer f.Close()

	for i := 0; i < *count; i++ {
		// Unique token ID
		tokenUUID := make([]byte, 16)
		_, _ = rand.Read(tokenUUID)
		tokenUUID[6] = (tokenUUID[6] & 0x0F) | 0x40
		tokenUUID[8] = (tokenUUID[8] & 0x3F) | 0x80
		tokenID := fmt.Sprintf("%x-%x-%x-%x-%x",
			tokenUUID[0:4], tokenUUID[4:6], tokenUUID[6:8], tokenUUID[8:10], tokenUUID[10:16])

		token := map[string]interface{}{
			"hacp_version":  "0.9",
			"token_id":      tokenID,
			"envelope_id":   "22222222-2222-2222-2222-222222222222",
			"action_hash":   actionHash,
			"policy_digest": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			"principal":     "human_admin_01",
			"signer_key_id": keyID,
			"issued_at":     now,
			"expires_at":    expires,
			"decision":      "ALLOW",
			"constraints": map[string]interface{}{
				"method":   *method,
				"path":     *path,
				"max_uses": 99999,
			},
		}

		tokJSON, _ := json.Marshal(token)
		tokCanonical, _ := wire.CanonicalizeJSON(tokJSON)
		tokSig := ed25519.Sign(privKey, tokCanonical)
		token["signature"] = wire.Base64URLEncode(tokSig)
		tokFinalJSON, _ := json.Marshal(token)
		tokHeader := wire.Base64URLEncode(tokFinalJSON)

		fmt.Fprintf(f, `{"env":"%s","tok":"%s"}`+"\n", envHeader, tokHeader)
	}

	log.Printf("Generated %d tokens -> %s", *count, *outFile)
}
