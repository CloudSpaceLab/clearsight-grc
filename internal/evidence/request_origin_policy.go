package evidence

import (
	"context"
	"strings"
)

type requestOriginAuthorityKey struct{}

var reservedRequestOrigins = map[string]struct{}{
	"THIRD_PARTY_ASSESSMENT":           {},
	"THIRD_PARTY_WORK":                 {},
	"THIRD_PARTY_ADDRESS_VERIFICATION": {},
}

// WithRequestOriginAuthority is for the domain orchestrator that owns a
// reserved origin namespace. HTTP request creation never applies this marker.
func WithRequestOriginAuthority(ctx context.Context, originType string) context.Context {
	return context.WithValue(ctx, requestOriginAuthorityKey{}, strings.ToUpper(strings.TrimSpace(originType)))
}

func requestOriginAllowed(ctx context.Context, origin RequestOrigin) bool {
	typeName := strings.ToUpper(strings.TrimSpace(origin.Type))
	if _, reserved := reservedRequestOrigins[typeName]; !reserved {
		return true
	}
	trusted, _ := ctx.Value(requestOriginAuthorityKey{}).(string)
	return trusted == typeName
}
