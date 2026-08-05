package identity

import (
	"errors"
	"log/slog"
	"net/http"
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
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"code":"identity_not_verified","message":"` + message + `"}}`))
				return
			}
			if present {
				logger.Debug("request identity verified", "tenant_id", actor.TenantID, "principal_id", actor.PrincipalID, "assurance", actor.AssuranceLevel)
				r = r.WithContext(WithActor(r.Context(), actor))
			}
			next.ServeHTTP(w, r)
		})
	}
}
