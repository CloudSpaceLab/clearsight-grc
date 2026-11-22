package federation

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
)

type originalSessionTokenKey struct{}

// The hash classifies the token already presented for this request. It is not
// an identity or authorization proof and is never exposed to application code.
func withOriginalSessionToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, originalSessionTokenKey{}, hashSessionToken(token))
}
func hashSessionToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
func emptySessionTokenHash() string { return hashSessionToken("") }
