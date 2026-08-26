package httpapi

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

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
	objectType, objectID := commandOutcomeObject(r, policy, payload)
	if objectID == "" || objectID == "*" {
		handler(w, r)
		return
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

	version, confirmed := a.exactMaterialOutcomeVersion(r, objectType, objectID, payload)
	if !confirmed {
		buffered.flushTo(w)
		return
	}

	w.Header().Set("X-ClearSight-Command-Outcome", "committed-response-degraded")
	w.Header().Set("X-ClearSight-Aggregate-Version", strconv.FormatInt(version, 10))
	httpx.WriteJSON(w, http.StatusOK, committedCommandReceipt{
		Status:           "COMMITTED",
		AggregateType:    objectType,
		AggregateID:      objectID,
		Version:          version,
		ResponseDegraded: true,
	})
}

func (a *API) exactMaterialOutcomeVersion(r *http.Request, objectType, objectID string, payload map[string]any) (int64, bool) {
	if objectType != "VENDOR_BRAND" || a.deps.VendorBrands == nil || r == nil {
		return 0, false
	}
	expectedVersion, ok := int64Value(payload["expected_version"])
	if !ok || expectedVersion < 0 {
		return 0, false
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		return 0, false
	}
	command := ""
	switch r.Method {
	case http.MethodPut:
		command = thirdparty.VendorBrandApproveCommand
	case http.MethodDelete:
		command = thirdparty.VendorBrandRemoveCommand
	default:
		return 0, false
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		return 0, false
	}
	version, err := a.deps.VendorBrands.CommandReceiptVersion(r.Context(), actor, objectID, idempotencyKey, command, expectedVersion)
	return version, err == nil
}

func commandOutcomeObject(r *http.Request, policy commandPolicy, payload map[string]any) (string, string) {
	objectType := policy.ObjectType
	objectID := commandObjectID(r, payload)
	if strings.TrimSpace(policy.OutcomeObjectType) != "" {
		objectType = strings.TrimSpace(policy.OutcomeObjectType)
	}
	if r != nil && strings.TrimSpace(policy.OutcomePathValue) != "" {
		objectID = strings.TrimSpace(r.PathValue(policy.OutcomePathValue))
	}
	return objectType, objectID
}

func commandObjectID(r *http.Request, payload map[string]any) string {
	if r != nil {
		if value := strings.TrimSpace(r.PathValue("id")); value != "" {
			return value
		}
		if value := strings.TrimSpace(r.PathValue("vendor_id")); value != "" {
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
