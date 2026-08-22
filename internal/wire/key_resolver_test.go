package wire

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"testing"
)

func staticTestKey(fill byte) ed25519.PublicKey {
	key := make(ed25519.PublicKey, ed25519.PublicKeySize)
	for i := range key {
		key[i] = fill
	}
	return key
}

func staticHexKey(fill byte) string {
	const hex = "0123456789abcdef"
	raw := staticTestKey(fill)
	out := make([]byte, len(raw)*2)
	for i, b := range raw {
		out[i*2] = hex[b>>4]
		out[i*2+1] = hex[b&0x0f]
	}
	return string(out)
}

func TestStaticKeyResolverAddKeyRegistersNewBinding(t *testing.T) {
	resolver := NewStaticKeyResolver()
	want := staticTestKey(1)

	if err := resolver.AddKey("key-a", want); err != nil {
		t.Fatalf("AddKey() error = %v", err)
	}

	got, err := resolver.ResolveKey("key-a")
	if err != nil {
		t.Fatalf("ResolveKey() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("ResolveKey() = %x, want %x", got, want)
	}
}

func TestStaticKeyResolverAddKeySameBindingIsIdempotent(t *testing.T) {
	resolver := NewStaticKeyResolver()
	key := staticTestKey(1)

	if err := resolver.AddKey("key-a", key); err != nil {
		t.Fatalf("first AddKey() error = %v", err)
	}
	if err := resolver.AddKey("key-a", append(ed25519.PublicKey(nil), key...)); err != nil {
		t.Fatalf("second AddKey() error = %v", err)
	}
}

func TestStaticKeyResolverAddKeyRejectsConflictingBinding(t *testing.T) {
	resolver := NewStaticKeyResolver()
	original := staticTestKey(1)
	conflicting := staticTestKey(2)

	if err := resolver.AddKey("key-a", original); err != nil {
		t.Fatalf("first AddKey() error = %v", err)
	}

	err := resolver.AddKey("key-a", conflicting)
	if !errors.Is(err, ErrConflictingKeyBinding) {
		t.Fatalf("conflicting AddKey() error = %v, want ErrConflictingKeyBinding", err)
	}

	got, err := resolver.ResolveKey("key-a")
	if err != nil {
		t.Fatalf("ResolveKey() error = %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("conflicting AddKey() changed binding: got %x want %x", got, original)
	}
}

func TestStaticKeyResolverAddKeyFromHexSameBindingIsIdempotent(t *testing.T) {
	resolver := NewStaticKeyResolver()
	hexKey := staticHexKey(1)

	if err := resolver.AddKeyFromHex("key-a", hexKey); err != nil {
		t.Fatalf("first AddKeyFromHex() error = %v", err)
	}
	if err := resolver.AddKeyFromHex("key-a", hexKey); err != nil {
		t.Fatalf("second AddKeyFromHex() error = %v", err)
	}
}

func TestStaticKeyResolverAddKeyFromHexRejectsConflictingBinding(t *testing.T) {
	resolver := NewStaticKeyResolver()

	if err := resolver.AddKeyFromHex("key-a", staticHexKey(1)); err != nil {
		t.Fatalf("first AddKeyFromHex() error = %v", err)
	}

	err := resolver.AddKeyFromHex("key-a", staticHexKey(2))
	if !errors.Is(err, ErrConflictingKeyBinding) {
		t.Fatalf("conflicting AddKeyFromHex() error = %v, want ErrConflictingKeyBinding", err)
	}
}
