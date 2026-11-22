package federation

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/alexedwards/scs/v2"
)

// OIDCSessionProof is ephemeral server-only data. It must never be returned or logged.
type OIDCSessionProof struct {
	TokenHash string
	Data      []byte
}

// ErrOIDCSessionProofUnavailable indicates a missing, mismatched or revoked session.
var ErrOIDCSessionProofUnavailable = errors.New("OIDC session proof unavailable")

// VerifiedOIDCSession binds the SCS-loaded session to the actor verified for
// this request. It does not itself prove that the database row remains active.
func VerifiedOIDCSession(ctx context.Context, actor identity.Actor) (OIDCSessionProof, error) {
	verified, ok := identity.FromContext(ctx)
	if !ok || verified.Valid(time.Now()) != nil || verified.AuthenticationMethod != "OIDC" || verified.Kind != "PERSON" || verified.Kind != actor.Kind || actor.IssuedAt.IsZero() || verified.TenantID != actor.TenantID || verified.PrincipalID != actor.PrincipalID || verified.LegalEntityID != actor.LegalEntityID || verified.SessionID != actor.SessionID || verified.AuthenticationMethod != actor.AuthenticationMethod || verified.AssuranceLevel != actor.AssuranceLevel || !verified.IssuedAt.Equal(actor.IssuedAt) || !verified.ExpiresAt.Equal(actor.ExpiresAt) || strings.TrimSpace(actor.SessionID) == "" {
		return OIDCSessionProof{}, ErrOIDCSessionProofUnavailable
	}
	state, ok := ctx.Value(originalSessionTokenKey{}).(*sessionRequestState)
	if !ok || state.hash == "" || state.hash == emptySessionTokenHash() {
		return OIDCSessionProof{}, ErrOIDCSessionProofUnavailable
	}
	state.mu.Lock()
	data := append([]byte(nil), state.loaded...)
	state.mu.Unlock()
	if len(data) == 0 {
		return OIDCSessionProof{}, ErrOIDCSessionProofUnavailable
	}
	deadline, values, err := (scs.GobCodec{}).Decode(data)
	if err != nil || !deadline.Equal(actor.ExpiresAt) {
		return OIDCSessionProof{}, ErrOIDCSessionProofUnavailable
	}
	tenant, tok := values[sessionTenantID].(string)
	principal, pok := values[sessionPrincipalID].(string)
	entity, eok := values[sessionLegalEntityID].(string)
	session, sok := values[sessionSessionID].(string)
	issued, iok := values[sessionIssuedAt].(time.Time)
	assurance, _ := values[sessionAssurance].(string)
	if !tok || !pok || !eok || !sok || !iok || tenant != actor.TenantID || principal != actor.PrincipalID || entity != actor.LegalEntityID || session != actor.SessionID || !issued.Equal(actor.IssuedAt) || assurance != actor.AssuranceLevel {
		return OIDCSessionProof{}, ErrOIDCSessionProofUnavailable
	}
	return OIDCSessionProof{TokenHash: state.hash, Data: data}, nil
}
