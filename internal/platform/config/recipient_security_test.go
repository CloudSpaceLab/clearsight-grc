package config

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestLoadRecipientSecurityConfig(t *testing.T) {
	setRecipientSecurityEnv(t)
	encryptionKey := encodedConfigKey(0x31)
	accessKey := encodedConfigKey(0x42)
	t.Setenv("CLEARSIGHT_EXTERNAL_DISTRIBUTION_DELIVERY_ENABLED", "true")
	t.Setenv("CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL", "http://localhost:5173/respond")
	t.Setenv("CLEARSIGHT_RECIPIENT_KEYRING", fmt.Sprintf(`{"key-v1":%q,"key-v0":%q}`, encryptionKey, encodedConfigKey(0x20)))
	t.Setenv("CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID", "key-v1")
	t.Setenv("CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY", accessKey)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	security := cfg.RecipientSecurity
	if !security.ExternalDeliveryEnabled || security.ActiveKeyID != "key-v1" || len(security.Keyring) != 2 {
		t.Fatalf("unexpected recipient security config: %#v", security)
	}
	if security.Keyring["key-v1"][0] != 0x31 || security.AccessHMACKey[0] != 0x42 {
		t.Fatal("recipient security keys were decoded incorrectly")
	}
	for _, output := range []string{security.String(), fmt.Sprintf("%#v", security)} {
		if strings.Contains(output, encryptionKey) || strings.Contains(output, accessKey) || strings.Contains(output, "49 49 49") {
			t.Fatalf("recipient security formatting leaked key material: %s", output)
		}
	}
}

func TestRecipientSecurityRejectsInvalidKeyMaterial(t *testing.T) {
	cases := []struct {
		name, keyring, active, accessKey, want string
	}{
		{name: "invalid-json", keyring: `{`, active: "key-v1", want: "JSON object"},
		{name: "short-encryption-key", keyring: `{"key-v1":"c2hvcnQ="}`, active: "key-v1", want: "32-byte"},
		{name: "zero-encryption-key", keyring: fmt.Sprintf(`{"key-v1":%q}`, encodedConfigKey(0x00)), active: "key-v1", want: "32-byte"},
		{name: "missing-active-key", keyring: fmt.Sprintf(`{"key-v1":%q}`, encodedConfigKey(0x31)), want: "ACTIVE_KEY_ID is required"},
		{name: "unknown-active-key", keyring: fmt.Sprintf(`{"key-v1":%q}`, encodedConfigKey(0x31)), active: "key-v2", want: "does not identify"},
		{name: "short-access-key", keyring: fmt.Sprintf(`{"key-v1":%q}`, encodedConfigKey(0x31)), active: "key-v1", accessKey: "c2hvcnQ=", want: "DISTRIBUTION_ACCESS_HMAC_KEY"},
		{name: "zero-access-key", keyring: fmt.Sprintf(`{"key-v1":%q}`, encodedConfigKey(0x31)), active: "key-v1", accessKey: encodedConfigKey(0x00), want: "DISTRIBUTION_ACCESS_HMAC_KEY"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			setRecipientSecurityEnv(t)
			t.Setenv("CLEARSIGHT_RECIPIENT_KEYRING", testCase.keyring)
			t.Setenv("CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID", testCase.active)
			t.Setenv("CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY", testCase.accessKey)
			_, err := loadRecipientSecurityConfig()
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("expected error containing %q, got %v", testCase.want, err)
			}
		})
	}
}

func TestExternalDistributionDeliveryFailsClosedWithoutSecurityConfiguration(t *testing.T) {
	setRecipientSecurityEnv(t)
	t.Setenv("CLEARSIGHT_ENV", "production")
	t.Setenv("CLEARSIGHT_IDENTITY_MODE", "signed")
	t.Setenv("CLEARSIGHT_IDENTITY_HMAC_SECRET", strings.Repeat("s", 32))
	t.Setenv("CLEARSIGHT_COMMAND_AUTHORIZATION", "enforce")
	t.Setenv("CLEARSIGHT_DEMO_MODE", "false")
	t.Setenv("CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL", "https://capture.example.test/respond")
	t.Setenv("CLEARSIGHT_EXTERNAL_DISTRIBUTION_DELIVERY_ENABLED", "true")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "valid recipient keyring") {
		t.Fatalf("production external delivery did not fail closed: %v", err)
	}
}

func TestExternalDistributionDeliveryRequiresPublicBaseURL(t *testing.T) {
	setRecipientSecurityEnv(t)
	t.Setenv("CLEARSIGHT_EXTERNAL_DISTRIBUTION_DELIVERY_ENABLED", "true")
	t.Setenv("CLEARSIGHT_RECIPIENT_KEYRING", fmt.Sprintf(`{"key-v1":%q}`, encodedConfigKey(0x31)))
	t.Setenv("CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID", "key-v1")
	t.Setenv("CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY", encodedConfigKey(0x42))

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "CAPTURE_PUBLIC_BASE_URL") {
		t.Fatalf("external delivery without a public base URL returned %v", err)
	}
}

func setRecipientSecurityEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CLEARSIGHT_ENV", "development")
	t.Setenv("CLEARSIGHT_IDENTITY_MODE", "development")
	t.Setenv("CLEARSIGHT_EXTERNAL_DISTRIBUTION_DELIVERY_ENABLED", "")
	t.Setenv("CLEARSIGHT_RECIPIENT_KEYRING", "")
	t.Setenv("CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID", "")
	t.Setenv("CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY", "")
	t.Setenv("CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL", "")
}

func encodedConfigKey(value byte) string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Repeat(string([]byte{value}), 32)))
}
