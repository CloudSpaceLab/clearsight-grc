package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestProtectedRecipientRoundTripUsesScopeBoundAAD(t *testing.T) {
	keyring := testRecipientKeyring(t, "key-v1", map[string][32]byte{"key-v1": repeatedRecipientKey(0x31)})
	value, err := keyring.Protect("tenant-a", "distribution-a", "recipient-a", "  Vendor.Owner@Example.Test  ")
	if err != nil {
		t.Fatal(err)
	}
	if value.KeyID != "key-v1" || value.HashHex == "" || value.Hint != "v***@example.test" || len(value.Ciphertext) <= 12 {
		t.Fatalf("incomplete protected value: %#v", value)
	}

	address, err := keyring.Reveal("tenant-a", "distribution-a", "recipient-a", value)
	if err != nil {
		t.Fatal(err)
	}
	if address != "Vendor.Owner@Example.Test" {
		t.Fatalf("round trip changed delivery address: %q", address)
	}

	for _, scope := range []struct {
		name, tenant, distribution, recipient string
	}{
		{name: "tenant", tenant: "tenant-b", distribution: "distribution-a", recipient: "recipient-a"},
		{name: "distribution", tenant: "tenant-a", distribution: "distribution-b", recipient: "recipient-a"},
		{name: "recipient", tenant: "tenant-a", distribution: "distribution-a", recipient: "recipient-b"},
	} {
		t.Run(scope.name, func(t *testing.T) {
			_, err := keyring.Reveal(scope.tenant, scope.distribution, scope.recipient, value)
			if !errors.Is(err, ErrProtectedRecipientInvalid) {
				t.Fatalf("wrong-scope reveal returned %v", err)
			}
		})
	}
}

func TestProtectedRecipientSupportsActiveKeyRotation(t *testing.T) {
	oldKey := repeatedRecipientKey(0x41)
	newKey := repeatedRecipientKey(0x52)
	oldRing := testRecipientKeyring(t, "key-v1", map[string][32]byte{"key-v1": oldKey})
	oldValue, err := oldRing.Protect("tenant-a", "distribution-a", "recipient-a", "owner@example.test")
	if err != nil {
		t.Fatal(err)
	}

	rotated := testRecipientKeyring(t, "key-v2", map[string][32]byte{"key-v1": oldKey, "key-v2": newKey})
	oldAddress, err := rotated.Reveal("tenant-a", "distribution-a", "recipient-a", oldValue)
	if err != nil || oldAddress != "owner@example.test" {
		t.Fatalf("rotated keyring could not reveal old ciphertext: %q %v", oldAddress, err)
	}
	newValue, err := rotated.Protect("tenant-a", "distribution-a", "recipient-b", "reviewer@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if newValue.KeyID != "key-v2" {
		t.Fatalf("new ciphertext did not use active key: %q", newValue.KeyID)
	}
	if _, err := oldRing.Reveal("tenant-a", "distribution-a", "recipient-b", newValue); !errors.Is(err, ErrProtectedRecipientInvalid) {
		t.Fatalf("retired keyring unexpectedly revealed new ciphertext: %v", err)
	}
}

func TestProtectedRecipientUsesNormalizedHashAndFreshNonce(t *testing.T) {
	keyring := testRecipientKeyring(t, "key-v1", map[string][32]byte{"key-v1": repeatedRecipientKey(0x63)})
	first, err := keyring.Protect("tenant-a", "distribution-a", "recipient-a", "OWNER@EXAMPLE.TEST")
	if err != nil {
		t.Fatal(err)
	}
	second, err := keyring.Protect("tenant-a", "distribution-a", "recipient-a", " owner@example.test ")
	if err != nil {
		t.Fatal(err)
	}
	if first.HashHex != second.HashHex {
		t.Fatalf("equivalent addresses produced different hashes: %q != %q", first.HashHex, second.HashHex)
	}
	if string(first.Ciphertext) == string(second.Ciphertext) {
		t.Fatal("AES-GCM ciphertext reused a nonce for the same address and scope")
	}
}

func TestProtectedRecipientDoesNotSerializeOrFormatPlaintext(t *testing.T) {
	const address = "highly.sensitive@example.test"
	keyring := testRecipientKeyring(t, "key-v1", map[string][32]byte{"key-v1": repeatedRecipientKey(0x74)})
	value, err := keyring.Protect("tenant-a", "distribution-a", "recipient-a", address)
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for name, output := range map[string]string{
		"json":        string(encoded),
		"string":      value.String(),
		"go-string":   fmt.Sprintf("%#v", value),
		"keyring":     fmt.Sprintf("%#v", keyring),
		"scope-error": protectedRecipientErrorText(keyring, value),
	} {
		if strings.Contains(strings.ToLower(output), address) {
			t.Fatalf("%s leaked plaintext address: %s", name, output)
		}
		if strings.Contains(output, string(value.Ciphertext)) {
			t.Fatalf("%s leaked raw ciphertext bytes", name)
		}
	}
}

func TestProtectedRecipientAdapterHonorsCancellation(t *testing.T) {
	keyring := testRecipientKeyring(t, "key-v1", map[string][32]byte{"key-v1": repeatedRecipientKey(0x85)})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := keyring.ProtectRecipientAddress(ctx, "tenant-a", "distribution-a", "recipient-a", "owner@example.test")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled protection, got %v", err)
	}
}

func TestProtectedRecipientRejectsUnconfiguredKeysAndAmbiguousInput(t *testing.T) {
	if _, err := NewRecipientKeyring("key-v1", map[string][32]byte{"key-v1": {}}); !errors.Is(err, ErrRecipientProtectionUnavailable) {
		t.Fatalf("zero encryption key was accepted: %v", err)
	}
	keyring := testRecipientKeyring(t, "key-v1", map[string][32]byte{"key-v1": repeatedRecipientKey(0x96)})
	cases := []struct {
		name, tenant, address string
	}{
		{name: "padded-scope", tenant: " tenant-a", address: "owner@example.test"},
		{name: "control-character", tenant: "tenant-a", address: "owner@example.test\n"},
		{name: "oversized", tenant: "tenant-a", address: strings.Repeat("a", 321)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := keyring.Protect(testCase.tenant, "distribution-a", "recipient-a", testCase.address)
			if !errors.Is(err, ErrRecipientProtectionUnavailable) {
				t.Fatalf("invalid protected-recipient input returned %v", err)
			}
		})
	}
}

func protectedRecipientErrorText(keyring RecipientKeyring, value ProtectedAddress) string {
	_, err := keyring.Reveal("tenant-b", "distribution-a", "recipient-a", value)
	if err == nil {
		return ""
	}
	return err.Error()
}

func testRecipientKeyring(t *testing.T, activeID string, keys map[string][32]byte) RecipientKeyring {
	t.Helper()
	keyring, err := NewRecipientKeyring(activeID, keys)
	if err != nil {
		t.Fatal(err)
	}
	return keyring
}

func repeatedRecipientKey(value byte) [32]byte {
	var key [32]byte
	for index := range key {
		key[index] = value
	}
	return key
}
