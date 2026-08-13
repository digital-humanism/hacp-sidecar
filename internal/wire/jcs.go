package wire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// CanonicalizeJSON applies RFC 8785 JSON Canonicalization Scheme (JCS)
// Implements minimal JCS without external dependencies
func CanonicalizeJSON(data []byte) ([]byte, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	var buf bytes.Buffer
	if err := canonicalizeValue(&buf, obj); err != nil {
		return nil, fmt.Errorf("JCS canonicalization failed: %w", err)
	}

	return buf.Bytes(), nil
}

// canonicalizeValue recursively canonicalizes a JSON value
func canonicalizeValue(buf *bytes.Buffer, val interface{}) error {
	switch v := val.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if v {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case float64:
		canonicalizeNumber(buf, v)
	case string:
		canonicalizeString(buf, v)
	case []interface{}:
		buf.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := canonicalizeValue(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]interface{}:
		buf.WriteByte('{')

		// Sort keys by Unicode code point (RFC 8785 §3.2.3)
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			canonicalizeString(buf, k)
			buf.WriteByte(':')
			if err := canonicalizeValue(buf, v[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		return fmt.Errorf("unsupported type: %T", val)
	}
	return nil
}

// canonicalizeNumber formats a number per RFC 8785 §3.2.2
func canonicalizeNumber(buf *bytes.Buffer, f float64) {
	// Special cases
	if f == 0 {
		buf.WriteByte('0')
		return
	}

	// Use strconv for consistent formatting
	// RFC 8785 requires shortest representation that round-trips
	s := strconv.FormatFloat(f, 'g', -1, 64)
	buf.WriteString(s)
}

// canonicalizeString escapes a string per RFC 8785 §3.2.1
func canonicalizeString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"':
			buf.WriteString(`\"`)
		case r == '\\':
			buf.WriteString(`\\`)
		case r == '\b':
			buf.WriteString(`\b`)
		case r == '\f':
			buf.WriteString(`\f`)
		case r == '\n':
			buf.WriteString(`\n`)
		case r == '\r':
			buf.WriteString(`\r`)
		case r == '\t':
			buf.WriteString(`\t`)
		case r < 0x20:
			// Control characters: use \uXXXX
			buf.WriteString(fmt.Sprintf(`\u%04x`, r))
		default:
			buf.WriteRune(r)
		}
	}
	buf.WriteByte('"')
}

// RemoveSignatureField removes the "signature" field from JSON for verification
func RemoveSignatureField(data []byte) ([]byte, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Remove signature field
	delete(obj, "signature")

	// Marshal back
	result, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("JSON marshal error: %w", err)
	}

	return result, nil
}

// VerifyCanonicalization verifies that two JSON objects are canonically equivalent
func VerifyCanonicalization(a, b []byte) (bool, error) {
	canonA, err := CanonicalizeJSON(a)
	if err != nil {
		return false, fmt.Errorf("canonicalize A error: %w", err)
	}

	canonB, err := CanonicalizeJSON(b)
	if err != nil {
		return false, fmt.Errorf("canonicalize B error: %w", err)
	}

	return bytes.Equal(canonA, canonB), nil
}
