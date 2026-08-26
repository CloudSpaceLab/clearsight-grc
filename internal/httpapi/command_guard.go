package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

const maxCommandBody = 1 << 20

const noActorField = "-"

type commandPolicy struct {
	ObjectType      string
	Responsibility  authority.Responsibility
	Materiality     int
	AllowService    bool
	BindLegalEntity bool
	ActorField      string
	DecisionType    string
}

// command binds verified identity, resolves the lifecycle-specific authority
// requirement from current aggregate state, and only then executes the command.
// ModeOff disables only authority resolution; tenant/actor fields remain
// server-bound and lifecycle validity is still enforced.
func (a *API) command(name string, policy commandPolicy, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, _, err := commandPayload(r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The command body must be valid JSON.")
			return
		}
		actor, err := identity.Require(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
			return
		}
		if (name == "program.create" || name == "matter.create") && actor.LegalEntityID != "" && actor.LegalEntityID != "*" {
			payload["legal_entity_id"] = actor.LegalEntityID
		}
		if !bindPayloadIdentity(w, payload, actor, policy.BindLegalEntity) {
			return
		}
		policy, err = a.lifecycleCommandPolicy(r.Context(), r, actor.TenantID, name, payload, policy)
		if err != nil {
			if errors.Is(err, commandauth.ErrIdentityRequired) || errors.Is(err, commandauth.ErrTenantMismatch) || errors.Is(err, commandauth.ErrLegalEntityMismatch) || errors.Is(err, commandauth.ErrNotAuthorized) || errors.Is(err, commandauth.ErrGuardUnavailable) {
				writeCommandAuthorizationError(w, err)
				return
			}
			writeContinuityError(w, err)
			return
		}
		// Existing-record lifecycle binding may narrow a tenant-wide verified
		// identity to the loaded record's exact legal entity. Use that narrowed
		// identity for the authority guard and the material handler.
		actor, err = identity.Require(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
			return
		}
		if field := commandActorField(policy); field != "" {
			payload[field] = actor.PrincipalID
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The command body could not be processed.")
			return
		}
		restoreJSONBody(r, raw)

		if a.deps.CommandGuard == nil || a.deps.CommandGuard.Mode() == commandauth.ModeOff {
			a.executeMaterialHandler(w, r, policy, payload, handler)
			return
		}

		objectID := commandObjectID(r, payload)
		materiality := policy.Materiality
		if value, ok := numberValue(payload["materiality"]); ok {
			materiality = max(materiality, value)
		} else if value, ok := numberValue(payload["priority"]); ok {
			materiality = max(materiality, value)
		}
		legalEntityID := actor.LegalEntityID
		decisionType := strings.TrimSpace(policy.DecisionType)
		if decisionType == "" {
			decisionType = name
		}
		decision, authErr := a.deps.CommandGuard.Authorize(r.Context(), commandauth.Request{
			TenantID:       actor.TenantID,
			LegalEntityID:  legalEntityID,
			ObjectType:     policy.ObjectType,
			ObjectID:       objectID,
			Responsibility: policy.Responsibility,
			DecisionType:   decisionType,
			Materiality:    materiality,
			AllowService:   policy.AllowService,
		})
		if authErr != nil {
			writeCommandAuthorizationError(w, authErr)
			return
		}
		if decision.Enforced {
			w.Header().Set("X-ClearSight-Command-Authorization", "enforced")
		} else {
			w.Header().Set("X-ClearSight-Command-Authorization", "audit")
		}
		a.executeMaterialHandler(w, r, policy, payload, handler)
	}
}

func commandActorField(policy commandPolicy) string {
	if policy.ActorField == noActorField {
		return ""
	}
	if strings.TrimSpace(policy.ActorField) != "" {
		return policy.ActorField
	}
	return "actor_id"
}

func commandPayload(r *http.Request) (map[string]any, []byte, error) {
	if r.Body == nil {
		return map[string]any{}, []byte("{}"), nil
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxCommandBody+1))
	if err != nil || len(raw) > maxCommandBody {
		return nil, nil, errors.New("command body exceeds the supported size")
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = []byte("{}")
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, nil, err
	}
	return payload, raw, nil
}

func restoreJSONBody(r *http.Request, raw []byte) {
	r.Body = io.NopCloser(bytes.NewReader(raw))
	r.ContentLength = int64(len(raw))
}

func bindPayloadIdentity(w http.ResponseWriter, payload map[string]any, actor identity.Actor, injectLegalEntity bool) bool {
	if tenant := stringValue(payload["tenant_id"]); tenant != "" && tenant != actor.TenantID {
		httpx.WriteError(w, http.StatusForbidden, "tenant_not_allowed", "This command is outside your signed-in bank scope.")
		return false
	}
	payload["tenant_id"] = actor.TenantID

	if requested := stringValue(payload["legal_entity_id"]); requested != "" {
		if actor.LegalEntityID != "" && actor.LegalEntityID != "*" && requested != actor.LegalEntityID {
			httpx.WriteError(w, http.StatusForbidden, "legal_entity_not_allowed", "This command is outside your signed-in legal-entity scope.")
			return false
		}
		if actor.LegalEntityID != "" && actor.LegalEntityID != "*" {
			payload["legal_entity_id"] = actor.LegalEntityID
		}
	} else if injectLegalEntity && actor.LegalEntityID != "" && actor.LegalEntityID != "*" {
		payload["legal_entity_id"] = actor.LegalEntityID
	}
	return true
}

func writeCommandAuthorizationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, commandauth.ErrIdentityRequired):
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
	case errors.Is(err, commandauth.ErrTenantMismatch):
		httpx.WriteError(w, http.StatusForbidden, "tenant_not_allowed", "This command is outside your signed-in bank scope.")
	case errors.Is(err, commandauth.ErrLegalEntityMismatch):
		httpx.WriteError(w, http.StatusForbidden, "legal_entity_not_allowed", "This command is outside your signed-in legal-entity scope.")
	case errors.Is(err, commandauth.ErrGuardUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "authority_unavailable", "The approval route could not be checked. No change was made.")
	default:
		httpx.WriteError(w, http.StatusForbidden, "approval_required", "You are not the person currently authorized for this change.")
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func numberValue(value any) (int, bool) {
	number, ok := value.(float64)
	if !ok {
		return 0, false
	}
	return int(number), true
}
