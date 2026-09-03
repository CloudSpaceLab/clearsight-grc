package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/activity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type auditExportRequest struct {
	Format        string `json:"format"`
	From          string `json:"from,omitempty"`
	To            string `json:"to,omitempty"`
	Category      string `json:"category,omitempty"`
	EventType     string `json:"event_type,omitempty"`
	ObjectType    string `json:"object_type,omitempty"`
	ObjectID      string `json:"object_id,omitempty"`
	ActorID       string `json:"actor_id,omitempty"`
	Actor         string `json:"actor,omitempty"`
	ActorKind     string `json:"actor_kind,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
}

func (a *API) createAuditExport(w http.ResponseWriter, r *http.Request) {
	if a.deps.AuditExports == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "audit_export_unavailable", "Audit export is unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	var input auditExportRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_audit_export", "Check the export format and filters and try again.")
		return
	}
	from, err := parseActivityTime(input.From, false)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_audit_export", "Export start time must be an RFC3339 timestamp or YYYY-MM-DD date.")
		return
	}
	to, err := parseActivityTime(input.To, true)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_audit_export", "Export end time must be an RFC3339 timestamp or YYYY-MM-DD date.")
		return
	}
	query := activity.Query{
		From:          from,
		To:            to,
		Category:      input.Category,
		EventType:     input.EventType,
		ObjectType:    input.ObjectType,
		ObjectID:      input.ObjectID,
		ActorID:       input.ActorID,
		ActorQuery:    input.Actor,
		ActorKind:     input.ActorKind,
		LegalEntityID: input.LegalEntityID,
	}
	receipt, err := a.deps.AuditExports.Create(r.Context(), actor.TenantID, input.LegalEntityID, actor.PrincipalID, input.Format, query)
	switch {
	case errors.Is(err, activity.ErrExportTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "audit_export_too_large", "The matching audit population is too large for a direct export. Narrow the date range or filters; no partial file was created.")
		return
	case errors.Is(err, activity.ErrExportInvalid), errors.Is(err, activity.ErrInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_audit_export", "The audit export filter is invalid.")
		return
	case err != nil:
		httpx.WriteError(w, http.StatusInternalServerError, "audit_export_failed", "The audit export could not be generated.")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, receipt)
}

func (a *API) getAuditExport(w http.ResponseWriter, r *http.Request) {
	if a.deps.AuditExports == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "audit_export_unavailable", "Audit export is unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	receipt, err := a.deps.AuditExports.Get(r.Context(), actor.TenantID, r.PathValue("id"))
	if errors.Is(err, activity.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "audit_export_not_found", "The audit export was not found.")
		return
	}
	if errors.Is(err, activity.ErrExportInvalid) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_audit_export", "The audit export identifier is invalid.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "audit_export_failed", "The audit export status could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, receipt)
}

func (a *API) downloadAuditExport(w http.ResponseWriter, r *http.Request) {
	if a.deps.AuditExports == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "audit_export_unavailable", "Audit export is unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	receipt, reader, err := a.deps.AuditExports.Open(r.Context(), actor.TenantID, r.PathValue("id"), actor.PrincipalID)
	if errors.Is(err, activity.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "audit_export_not_found", "The audit export was not found or has expired.")
		return
	}
	if errors.Is(err, activity.ErrExportInvalid) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_audit_export", "The audit export identifier is invalid.")
		return
	}
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "audit_export_failed", "The audit export could not be opened.")
		return
	}
	defer reader.Close()

	extension := "csv"
	contentType := "text/csv; charset=utf-8"
	if receipt.Format == activity.ExportFormatNDJSON {
		extension = "ndjson"
		contentType = "application/x-ndjson; charset=utf-8"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="clearsight-audit-%s.%s"`, receipt.ID, extension))
	w.Header().Set("Cache-Control", "private, no-store")
	if checksum := strings.TrimSpace(receipt.DataSHA256); checksum != "" {
		w.Header().Set("ETag", `"`+checksum+`"`)
	}
	if _, err := io.Copy(w, reader); err != nil && a.deps.Logger != nil {
		a.deps.Logger.Warn("audit export stream interrupted", "export_id", receipt.ID, "error", err)
	}
}
