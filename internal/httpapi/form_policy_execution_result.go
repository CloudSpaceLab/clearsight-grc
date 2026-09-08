package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"net/http"
)

type formPolicyResultTarget struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (a *API) getFormPolicyExecutionResult(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	service, actor, ok := a.formPolicyContext(w, r)
	if !ok {
		return
	}
	receipt, err := service.ExecutionResult(r.Context(), actor, r.PathValue("id"), r.PathValue("execution_id"))
	if err != nil {
		writeFormPolicyError(w, err)
		return
	}
	targets := []formPolicyResultTarget{}
	if receipt.MatterID != "" && a.deps.Continuity != nil {
		matter, readErr := a.deps.Continuity.GetMatter(r.Context(), actor.TenantID, receipt.MatterID)
		if readErr == nil && matter.Matter.LegalEntityID == actor.LegalEntityID && canReadMatterAggregate(r.Context(), matter) {
			targets = append(targets, formPolicyResultTarget{Type: "MATTER", ID: matter.Matter.ID, Title: matter.Matter.Title})
		}
	}
	if receipt.ResponseRevisionID != "" && a.deps.FormDistributions != nil {
		response, _, readErr := a.deps.FormDistributions.GetCompletedResponse(r.Context(), actor.TenantID, actor.LegalEntityID, actor.PrincipalID, receipt.ResponseRevisionID)
		if readErr == nil && response.TenantID == actor.TenantID && response.LegalEntityID == actor.LegalEntityID {
			targets = append(targets, formPolicyResultTarget{Type: "FORM_RESPONSE", ID: response.ID, Title: response.Title})
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"execution": receipt.ExecutionHistoryItem, "targets": targets})
}
