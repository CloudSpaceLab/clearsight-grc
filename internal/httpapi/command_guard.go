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

type commandPolicy struct {
	ObjectType     string
	Responsibility authority.Responsibility
	Materiality    int
	AllowService   bool
}

var commandPolicies = map[string]commandPolicy{
	"program.create":               {ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2},
	"program.transition":           {ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3},
	"program.requirement.add":      {ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2},
	"program.applicability.decide": {ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3},
	"program.safeguard.define":     {ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2},
	"program.evidence.define":      {ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2},
	"program.evidence.assess":      {ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityReviewer, Materiality: 3},
	"program.trigger.ingest":       {ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityPerformer, Materiality: 2, AllowService: true},
	"matter.create":                {ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3},
	"matter.link":                  {ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2},
	"matter.transition":            {ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3},
	"matter.decision.record":       {ObjectType: "MATTER", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4},
	"matter.action.add":            {ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2},
	"matter.action.transition":     {ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2},
	"matter.outcome.define":        {ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 3},
	"matter.outcome.record":        {ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 4},
	"matter.response.add":          {ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3},
	"matter.response.transition":   {ObjectType: "MATTER", Responsibility: authority.ResponsibilitySignatory, Materiality: 4},
	"projection.reconcile":         {ObjectType: "PROJECTION", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, AllowService: true},
	"projection.rebuild":           {ObjectType: "PROJECTION", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4},
}

func (a *API) command(name string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		policy, known := commandPolicies[name]
		if !known || a.deps.CommandGuard == nil || a.deps.CommandGuard.Mode() == commandauth.ModeOff {
			handler(w, r)
			return
		}
		payload, raw, err := commandPayload(r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The command body must be valid JSON.")
			return
		}
		actor, hasActor := identity.FromContext(r.Context())
		tenant := stringValue(payload["tenant_id"])
		if hasActor {
			if tenant != "" && tenant != actor.TenantID {
				httpx.WriteError(w, http.StatusForbidden, "tenant_not_allowed", "This command is outside your signed-in bank scope.")
				return
			}
			tenant = actor.TenantID
			payload["tenant_id"] = actor.TenantID
			bindCommandActor(name, payload, actor.PrincipalID)
			raw, err = json.Marshal(payload)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The command body could not be processed.")
				return
			}
		}
		objectID := r.PathValue("id")
		if objectID == "" {
			objectID = "*"
		}
		materiality := policy.Materiality
		if value, ok := numberValue(payload["materiality"]); ok {
			materiality = value
		} else if value, ok := numberValue(payload["priority"]); ok {
			materiality = value
		}
		responsibility := policy.Responsibility
		if name == "matter.transition" {
			to := strings.ToUpper(stringValue(payload["to"]))
			if to == "CLOSED" || to == "CANCELLED" || to == "DECISION_REQUIRED" {
				responsibility = authority.ResponsibilityAuthorizer
				materiality = max(materiality, 4)
			}
		}
		decision, authErr := a.deps.CommandGuard.Authorize(r.Context(), commandauth.Request{
			TenantID: tenant, LegalEntityID: stringValue(payload["legal_entity_id"]),
			ObjectType: policy.ObjectType, ObjectID: objectID,
			Responsibility: responsibility, DecisionType: name,
			Materiality: materiality, AllowService: policy.AllowService,
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
		r.Body = io.NopCloser(bytes.NewReader(raw))
		r.ContentLength = int64(len(raw))
		handler(w, r)
	}
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

func bindCommandActor(name string, payload map[string]any, principalID string) {
	payload["actor_id"] = principalID
	switch name {
	case "program.applicability.decide":
		payload["approved_by"] = principalID
	case "program.evidence.assess":
		payload["assessed_by"] = principalID
	case "matter.decision.record":
		payload["authority_principal_id"] = principalID
	case "matter.outcome.record":
		payload["reviewer_principal_id"] = principalID
	}
}

func writeCommandAuthorizationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, commandauth.ErrIdentityRequired):
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
	case errors.Is(err, commandauth.ErrTenantMismatch):
		httpx.WriteError(w, http.StatusForbidden, "tenant_not_allowed", "This command is outside your signed-in bank scope.")
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
