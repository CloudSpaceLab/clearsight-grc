package identity

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

func Middleware(authenticator Authenticator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authenticator == nil {
				next.ServeHTTP(w, r)
				return
			}
			actor, present, err := authenticator.Authenticate(r)
			if err != nil {
				status := http.StatusUnauthorized
				message := "Your sign-in could not be verified. Sign in again and retry."
				if errors.Is(err, ErrExpiredIdentity) {
					message = "Your sign-in has expired. Sign in again and retry."
				}
				writeIdentityError(w, status, "identity_not_verified", message)
				return
			}
			if present {
				if !requestedScopeMatches(r, actor) {
					writeIdentityError(w, http.StatusNotFound, "scope_not_found", "The requested organization scope was not found.")
					return
				}
				if logger != nil {
					logger.Debug("request identity verified", "tenant_id", actor.TenantID, "principal_id", actor.PrincipalID, "assurance", actor.AssuranceLevel)
				}
				r = r.WithContext(WithActor(r.Context(), actor))
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requestedScopeMatches(r *http.Request, actor Actor) bool {
	queries := r.URL.Query()
	return optionalScopeValueMatches(queries.Get("tenant_id"), actor.TenantID) &&
		optionalScopeValueMatches(queries.Get("principal_id"), actor.PrincipalID) &&
		optionalScopeValueMatches(queries.Get("legal_entity_id"), actor.LegalEntityID)
}

func optionalScopeValueMatches(requested, verified string) bool {
	requested = strings.TrimSpace(requested)
	return requested == "" || requested == strings.TrimSpace(verified)
}

func writeIdentityError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + message + `"}}`))
}
