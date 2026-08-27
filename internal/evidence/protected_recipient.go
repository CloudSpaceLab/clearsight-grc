package evidence

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

var (
	ErrRecipientProtectionUnavailable = errors.New("recipient protection is unavailable")
	ErrProtectedRecipientInvalid      = errors.New("protected recipient is unavailable")
)

// RecipientKeyring is purpose-specific key material for protected external
// distribution recipients. Keys are selected by identifier so old ciphertext
// remains decryptable while ActiveID rotates forward.
type RecipientKeyring struct {
	ActiveID string              `json:"active_key_id"`
	Keys     map[string][32]byte `json:"-"`
	random   io.Reader
}

// ProtectedAddress is safe to move through storage boundaries. Ciphertext is
// intentionally omitted from JSON and formatted output; callers receive the
// plaintext only from Reveal after all scope-bound AAD checks pass.
type ProtectedAddress struct {
	Ciphertext []byte `json:"-"`
	KeyID      string `json:"key_id"`
	HashHex    string `json:"hash_hex"`
	Hint       string `json:"hint"`
}

func NewRecipientKeyring(activeID string, keys map[string][32]byte) (RecipientKeyring, error) {
	keyring := RecipientKeyring{
		ActiveID: strings.TrimSpace(activeID),
		Keys:     cloneRecipientKeys(keys),
		random:   rand.Reader,
	}
	if err := keyring.validate(); err != nil {
		return RecipientKeyring{}, err
	}
	return keyring, nil
}

func (keyring RecipientKeyring) String() string {
	return fmt.Sprintf("RecipientKeyring{active_id:%q keys:protected}", keyring.ActiveID)
}

func (keyring RecipientKeyring) GoString() string {
	return keyring.String()
}

func (value ProtectedAddress) String() string {
	return fmt.Sprintf("ProtectedAddress{key_id:%q hash:%q hint:%q ciphertext:protected}", value.KeyID, value.HashHex, value.Hint)
}

func (value ProtectedAddress) GoString() string {
	return value.String()
}

func (keyring RecipientKeyring) Protect(tenantID, distributionID, recipientID, address string) (ProtectedAddress, error) {
	if err := keyring.validate(); err != nil {
		return ProtectedAddress{}, err
	}
	if !validRecipientScope(tenantID, distributionID, recipientID) {
		return ProtectedAddress{}, ErrRecipientProtectionUnavailable
	}
	trimmed := strings.TrimSpace(address)
	normalized := normalizeAudience(trimmed)
	if normalized == "" || len(trimmed) > 320 || strings.IndexFunc(address, unicode.IsControl) >= 0 {
		return ProtectedAddress{}, ErrRecipientProtectionUnavailable
	}

	key := keyring.Keys[keyring.ActiveID]
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return ProtectedAddress{}, ErrRecipientProtectionUnavailable
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ProtectedAddress{}, ErrRecipientProtectionUnavailable
	}
	nonce := make([]byte, gcm.NonceSize())
	random := keyring.random
	if random == nil {
		random = rand.Reader
	}
	if _, err := io.ReadFull(random, nonce); err != nil {
		return ProtectedAddress{}, ErrRecipientProtectionUnavailable
	}

	ciphertext := make([]byte, 0, len(nonce)+len(trimmed)+gcm.Overhead())
	ciphertext = append(ciphertext, nonce...)
	ciphertext = gcm.Seal(ciphertext, nonce, []byte(trimmed), recipientAAD(tenantID, distributionID, recipientID))
	digest := sha256.Sum256([]byte(normalized))
	return ProtectedAddress{
		Ciphertext: ciphertext,
		KeyID:      keyring.ActiveID,
		HashHex:    hex.EncodeToString(digest[:]),
		Hint:       audienceHint(normalized),
	}, nil
}

func (keyring RecipientKeyring) Reveal(tenantID, distributionID, recipientID string, value ProtectedAddress) (string, error) {
	if err := keyring.validate(); err != nil {
		return "", ErrProtectedRecipientInvalid
	}
	if !validRecipientScope(tenantID, distributionID, recipientID) {
		return "", ErrProtectedRecipientInvalid
	}
	keyID := strings.TrimSpace(value.KeyID)
	key, ok := keyring.Keys[keyID]
	if !ok || keyID != value.KeyID {
		return "", ErrProtectedRecipientInvalid
	}
	expectedHash, err := hex.DecodeString(value.HashHex)
	if err != nil || len(expectedHash) != sha256.Size {
		return "", ErrProtectedRecipientInvalid
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", ErrProtectedRecipientInvalid
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(value.Ciphertext) < gcm.NonceSize()+gcm.Overhead() {
		return "", ErrProtectedRecipientInvalid
	}
	nonce := value.Ciphertext[:gcm.NonceSize()]
	sealed := value.Ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, recipientAAD(tenantID, distributionID, recipientID))
	if err != nil {
		return "", ErrProtectedRecipientInvalid
	}
	address := strings.TrimSpace(string(plaintext))
	digest := sha256.Sum256([]byte(normalizeAudience(address)))
	if subtle.ConstantTimeCompare(digest[:], expectedHash) != 1 {
		return "", ErrProtectedRecipientInvalid
	}
	return address, nil
}

func (keyring RecipientKeyring) ProtectRecipientAddress(ctx context.Context, tenantID, distributionID, recipientID, address string) (protectedRecipientAddress, error) {
	if err := ctx.Err(); err != nil {
		return protectedRecipientAddress{}, err
	}
	value, err := keyring.Protect(tenantID, distributionID, recipientID, address)
	if err != nil {
		return protectedRecipientAddress{}, err
	}
	digest, err := hex.DecodeString(value.HashHex)
	if err != nil || len(digest) != sha256.Size {
		return protectedRecipientAddress{}, ErrRecipientProtectionUnavailable
	}
	return protectedRecipientAddress{
		Hash:       digest,
		Ciphertext: append([]byte(nil), value.Ciphertext...),
		KeyID:      value.KeyID,
	}, nil
}

func (keyring RecipientKeyring) RevealRecipientAddress(ctx context.Context, tenantID, distributionID, recipientID string, value protectedRecipientAddress) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return keyring.Reveal(tenantID, distributionID, recipientID, ProtectedAddress{
		Ciphertext: append([]byte(nil), value.Ciphertext...),
		KeyID:      value.KeyID,
		HashHex:    hex.EncodeToString(value.Hash),
	})
}

func (keyring RecipientKeyring) validate() error {
	activeID := strings.TrimSpace(keyring.ActiveID)
	if activeID == "" || activeID != keyring.ActiveID || len(keyring.Keys) == 0 {
		return ErrRecipientProtectionUnavailable
	}
	if _, ok := keyring.Keys[activeID]; !ok {
		return ErrRecipientProtectionUnavailable
	}
	for keyID, key := range keyring.Keys {
		if strings.TrimSpace(keyID) == "" || strings.TrimSpace(keyID) != keyID || len(keyID) > 128 || !securityKeyConfigured(key) {
			return ErrRecipientProtectionUnavailable
		}
	}
	return nil
}

func validRecipientScope(tenantID, distributionID, recipientID string) bool {
	for _, value := range []string{tenantID, distributionID, recipientID} {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || trimmed != value {
			return false
		}
	}
	return true
}

func recipientAAD(tenantID, distributionID, recipientID string) []byte {
	return []byte(tenantID + "|" + distributionID + "|" + recipientID)
}

func cloneRecipientKeys(keys map[string][32]byte) map[string][32]byte {
	if len(keys) == 0 {
		return nil
	}
	cloned := make(map[string][32]byte, len(keys))
	for keyID, key := range keys {
		cloned[keyID] = key
	}
	return cloned
}

var (
	_ recipientAddressProtector = RecipientKeyring{}
	_ fmt.Stringer              = RecipientKeyring{}
	_ fmt.GoStringer            = RecipientKeyring{}
	_ fmt.Stringer              = ProtectedAddress{}
	_ fmt.GoStringer            = ProtectedAddress{}
)
