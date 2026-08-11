package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type demoLoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *API) registerDemoRoutes(mux *http.ServeMux) {
	if !a.deps.DemoMode {
		return
	}
	authenticator, ok := a.deps.Identity.(identity.DemoSessionAuthenticator)
	if !ok {
		return
	}
	mux.HandleFunc("GET /api/v1/demo/accounts", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"accounts": authenticator.Accounts()})
	})
	mux.HandleFunc("POST /api/v1/demo/login", func(w http.ResponseWriter, r *http.Request) {
		var input demoLoginInput
		if err := httpx.DecodeJSON(w, r, &input); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Username and password are required.")
			return
		}
		account, err := authenticator.Login(w, strings.TrimSpace(input.Username), input.Password)
		if errors.Is(err, identity.ErrInvalidDemoCredentials) {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_demo_credentials", "The selected demo credentials are invalid.")
			return
		}
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "demo_login_failed", "The demo session could not be created.")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"account": account})
	})
	mux.HandleFunc("POST /api/v1/demo/logout", func(w http.ResponseWriter, _ *http.Request) {
		authenticator.Logout(w)
		w.WriteHeader(http.StatusNoContent)
	})
}
