package wire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
)

// CanonicalizeJSON applies RFC 8785 JSON Canonicalization Scheme (JCS).
// Uses json.Decoder.UseNumber() to preserve integer precision.
func CanonicalizeJSON(data []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var obj interface{}
	if err := dec.Decode(&obj); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	var buf bytes.Buffer
	if err := canonicalizeValue(&buf, obj); err != nil {
		return nil, fmt.Errorf("JCS canonicalization failed: %w", err)
	}

	return buf.Bytes(), nil
}

// canonicalizeValue recursively canonicalizes a JSON value.
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
	case json.Number:
		canonicalizeJSONNumber(buf, v)
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

// canonicalizeJSONNumber handles json.Number (from UseNumber()).
// This is critical: json.Number preserves the exact integer representation.
func canonicalizeJSONNumber(buf *bytes.Buffer, n json.Number) {
	// Try int64 first — this preserves integers exactly as they were written
	if i, err := n.Int64(); err == nil {
		buf.WriteString(strconv.FormatInt(i, 10))
		return
	}
	// Fall back to float64 for fractional numbers
	if f, err := n.Float64(); err == nil {
		canonicalizeNumber(buf, f)
		return
	}
	// Last resort: use the raw string representation
	buf.WriteString(n.String())
}

// canonicalizeNumber formats a float64 per RFC 8785 §3.2.2.
func canonicalizeNumber(buf *bytes.Buffer, f float64) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		buf.WriteString("null")
		return
	}

	if math.Abs(f) < float64(int64(1)<<53) {
		i := int64(f)
		if float64(i) == f {
			buf.WriteString(strconv.FormatInt(i, 10))
			return
		}
	}

	buf.WriteString(strconv.FormatFloat(f, 'g', -1, 64))
}

// canonicalizeString escapes a string per RFC 8785 §3.2.1.
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
			buf.WriteString(fmt.Sprintf(`\u%04x`, r))
		default:
			buf.WriteRune(r)
		}
	}
	buf.WriteByte('"')
}

// RemoveSignatureField removes the "signature" field from JSON for verification.
// CRITICAL: Uses json.Number to preserve integer precision, and calls
// canonicalizeValue directly instead of json.Marshal.
// json.Marshal would serialize float64(1786000000) as "1.786e+09" (scientific
// notation), which breaks JCS canonicalization and signature verification.
func RemoveSignatureField(data []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var obj map[string]interface{}
	if err := dec.Decode(&obj); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	delete(obj, "signature")

	var buf bytes.Buffer
	if err := canonicalizeValue(&buf, obj); err != nil {
		return nil, fmt.Errorf("canonicalize error: %w", err)
	}

	return buf.Bytes(), nil
}

// VerifyCanonicalization verifies that two JSON objects are canonically equivalent.
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
