package httpapi

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

var errMaterialVersionUnavailable = errors.New("material aggregate version unavailable")

type bufferedCommandResponse struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newBufferedCommandResponse() *bufferedCommandResponse {
	return &bufferedCommandResponse{header: make(http.Header)}
}

func (w *bufferedCommandResponse) Header() http.Header { return w.header }

func (w *bufferedCommandResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *bufferedCommandResponse) Write(value []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(value)
}

func (w *bufferedCommandResponse) flushTo(target http.ResponseWriter) {
	for key, values := range w.header {
		for _, value := range values {
			target.Header().Add(key, value)
		}
	}
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	target.WriteHeader(status)
	_, _ = target.Write(w.body.Bytes())
}

type committedCommandReceipt struct {
	Status           string `json:"status"`
	AggregateType    string `json:"aggregate_type"`
	AggregateID      string `json:"aggregate_id"`
	Version          int64  `json:"version"`
	ResponseDegraded bool   `json:"response_degraded"`
}

func (a *API) executeMaterialHandler(w http.ResponseWriter, r *http.Request, policy commandPolicy, payload map[string]any, handler http.HandlerFunc) {
	objectID := commandObjectID(r, payload)
	if objectID == "" || objectID == "*" {
		handler(w, r)
		return
	}

	baseline, baselineErr := a.currentMaterialVersion(r, policy.ObjectType, objectID, stringValue(payload["tenant_id"]))
	if baselineErr != nil {
		handler(w, r)
		return
	}
	if expected, ok := int64Value(payload["expected_version"]); ok && expected > baseline {
		baseline = expected
	}

	buffered := newBufferedCommandResponse()
	handler(buffered, r)
	status := buffered.status
	if status == 0 {
		status = http.StatusOK
	}
	if status < http.StatusInternalServerError {
		buffered.flushTo(w)
		return
	}

	version, err := a.currentMaterialVersion(r, policy.ObjectType, objectID, stringValue(payload["tenant_id"]))
	if err != nil || version <= baseline {
		buffered.flushTo(w)
		return
	}

	w.Header().Set("X-ClearSight-Command-Outcome", "committed-response-degraded")
	w.Header().Set("X-ClearSight-Aggregate-Version", strconv.FormatInt(version, 10))
	httpx.WriteJSON(w, http.StatusOK, committedCommandReceipt{
		Status:           "COMMITTED",
		AggregateType:    policy.ObjectType,
		AggregateID:      objectID,
		Version:          version,
		ResponseDegraded: true,
	})
}

func (a *API) currentMaterialVersion(r *http.Request, objectType, objectID, tenantID string) (int64, error) {
	switch objectType {
	case "PROGRAM", "MATTER":
		if a.deps.Continuity == nil {
			return 0, errMaterialVersionUnavailable
		}
		return a.deps.Continuity.CurrentVersion(r.Context(), tenantID, objectType, objectID)
	case "VENDOR_RELATIONSHIP":
		if a.deps.ThirdParty == nil {
			return 0, errMaterialVersionUnavailable
		}
		actor, err := thirdPartyActor(r)
		if err != nil {
			return 0, err
		}
		value, err := a.deps.ThirdParty.GetRelationship(r.Context(), actor, objectID)
		return value.Relationship.Version, err
	case "THIRD_PARTY_ASSESSMENT":
		if a.deps.ThirdPartyAssessments == nil {
			return 0, errMaterialVersionUnavailable
		}
		actor, err := thirdPartyActor(r)
		if err != nil {
			return 0, err
		}
		value, err := a.deps.ThirdPartyAssessments.GetAssessment(r.Context(), actor, objectID)
		return value.Version, err
	default:
		return 0, errMaterialVersionUnavailable
	}
}

func commandObjectID(r *http.Request, payload map[string]any) string {
	if r != nil {
		if value := strings.TrimSpace(r.PathValue("id")); value != "" {
			return value
		}
	}
	if value := stringValue(payload["program_id"]); value != "" {
		return value
	}
	if value := stringValue(payload["matter_id"]); value != "" {
		return value
	}
	if value := stringValue(payload["subject_id"]); value != "" {
		return value
	}
	return "*"
}

func int64Value(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), true
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	default:
		return 0, false
	}
}
