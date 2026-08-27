package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

const maxRecipientKeys = 32

type RecipientSecurityConfig struct {
	ExternalDeliveryEnabled bool                `json:"external_delivery_enabled"`
	ActiveKeyID             string              `json:"active_key_id,omitempty"`
	Keyring                 map[string][32]byte `json:"-"`
	AccessHMACKey           [32]byte            `json:"-"`
}

func (cfg RecipientSecurityConfig) String() string {
	return fmt.Sprintf("RecipientSecurityConfig{external_delivery_enabled:%t active_key_id:%q keyring:protected access_hmac_key:protected}", cfg.ExternalDeliveryEnabled, cfg.ActiveKeyID)
}

func (cfg RecipientSecurityConfig) GoString() string {
	return cfg.String()
}

func loadRecipientSecurityConfig() (RecipientSecurityConfig, error) {
	enabled, err := boolValue("CLEARSIGHT_EXTERNAL_DISTRIBUTION_DELIVERY_ENABLED", false)
	if err != nil {
		return RecipientSecurityConfig{}, err
	}
	keys, err := recipientKeyringValue("CLEARSIGHT_RECIPIENT_KEYRING")
	if err != nil {
		return RecipientSecurityConfig{}, err
	}
	activeID := strings.TrimSpace(env("CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID", ""))
	accessKey, accessKeyConfigured, err := optionalBase64Key("CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY")
	if err != nil {
		return RecipientSecurityConfig{}, err
	}
	cfg := RecipientSecurityConfig{
		ExternalDeliveryEnabled: enabled,
		ActiveKeyID:             activeID,
		Keyring:                 keys,
		AccessHMACKey:           accessKey,
	}
	if err := validateRecipientSecurityConfig(cfg, accessKeyConfigured); err != nil {
		return RecipientSecurityConfig{}, err
	}
	return cfg, nil
}

func validateRecipientSecurityConfig(cfg RecipientSecurityConfig, accessKeyConfigured bool) error {
	if len(cfg.Keyring) > maxRecipientKeys {
		return fmt.Errorf("CLEARSIGHT_RECIPIENT_KEYRING supports at most %d keys", maxRecipientKeys)
	}
	if len(cfg.Keyring) == 0 && cfg.ActiveKeyID != "" {
		return fmt.Errorf("CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID requires CLEARSIGHT_RECIPIENT_KEYRING")
	}
	if len(cfg.Keyring) > 0 {
		if cfg.ActiveKeyID == "" {
			return fmt.Errorf("CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID is required when CLEARSIGHT_RECIPIENT_KEYRING is configured")
		}
		if _, ok := cfg.Keyring[cfg.ActiveKeyID]; !ok {
			return fmt.Errorf("CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID does not identify a configured recipient key")
		}
	}
	if cfg.ExternalDeliveryEnabled {
		if len(cfg.Keyring) == 0 || cfg.ActiveKeyID == "" {
			return fmt.Errorf("external distribution delivery requires a valid recipient keyring and active key")
		}
		if !accessKeyConfigured {
			return fmt.Errorf("external distribution delivery requires CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY")
		}
	}
	return nil
}

func recipientKeyringValue(name string) (map[string][32]byte, error) {
	raw := strings.TrimSpace(env(name, ""))
	if raw == "" {
		return nil, nil
	}
	var encoded map[string]string
	if err := json.Unmarshal([]byte(raw), &encoded); err != nil {
		return nil, fmt.Errorf("%s must be a JSON object of base64 32-byte keys", name)
	}
	if len(encoded) == 0 {
		return nil, fmt.Errorf("%s must contain at least one key", name)
	}
	if len(encoded) > maxRecipientKeys {
		return nil, fmt.Errorf("%s supports at most %d keys", name, maxRecipientKeys)
	}
	keys := make(map[string][32]byte, len(encoded))
	for keyID, value := range encoded {
		trimmedID := strings.TrimSpace(keyID)
		if trimmedID == "" || trimmedID != keyID || len(trimmedID) > 128 {
			return nil, fmt.Errorf("%s contains an invalid key identifier", name)
		}
		key, err := decodeBase64Key(value)
		if err != nil {
			return nil, fmt.Errorf("%s key %q must be base64-encoded 32-byte material", name, keyID)
		}
		keys[keyID] = key
	}
	return keys, nil
}

func optionalBase64Key(name string) ([32]byte, bool, error) {
	raw := strings.TrimSpace(env(name, ""))
	if raw == "" {
		return [32]byte{}, false, nil
	}
	key, err := decodeBase64Key(raw)
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("%s must be base64-encoded 32-byte material", name)
	}
	return key, true, nil
}

func decodeBase64Key(value string) ([32]byte, error) {
	var key [32]byte
	value = strings.TrimSpace(value)
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(value)
	}
	if err != nil || len(decoded) != len(key) {
		return [32]byte{}, fmt.Errorf("invalid key material")
	}
	copy(key[:], decoded)
	if !configuredKeyMaterial(key) {
		return [32]byte{}, fmt.Errorf("invalid key material")
	}
	return key, nil
}

func configuredKeyMaterial(key [32]byte) bool {
	var aggregate byte
	for _, value := range key {
		aggregate |= value
	}
	return aggregate != 0
}

var (
	_ fmt.Stringer   = RecipientSecurityConfig{}
	_ fmt.GoStringer = RecipientSecurityConfig{}
)
