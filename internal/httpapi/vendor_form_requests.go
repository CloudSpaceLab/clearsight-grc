package httpapi

import (
	"net/http"
	"regexp"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type vendorFormRequestTarget struct {
	RelationshipID string                              `json:"relationship_id"`
	Recipient      evidence.DistributionRecipientInput `json:"recipient"`
}
type vendorFormBatchRequest struct {
	createFormDistributionRequest
	BatchID string                    `json:"batch_id"`
	Targets []vendorFormRequestTarget `json:"targets"`
}
type vendorFormRequestOutcome struct {
	RelationshipID    string                      `json:"relationship_id"`
	Status            string                      `json:"status"`
	DistributionID    string                      `json:"distribution_id,omitempty"`
	DistributionState evidence.DistributionStatus `json:"distribution_state,omitempty"`
	Error             string                      `json:"error,omitempty"`
}

var vendorBatchIDPattern = regexp.MustCompile(`^[a-zA-Z0-9-]{16,80}$`)

func validVendorFormBatch(input vendorFormBatchRequest) bool {
	if !vendorBatchIDPattern.MatchString(input.BatchID) || len(input.Targets) < 1 || len(input.Targets) > 50 {
		return false
	}
	seen := map[string]bool{}
	for _, target := range input.Targets {
		if target.RelationshipID == "" || len(target.RelationshipID) > 100 || seen[target.RelationshipID] || target.Recipient.Role != evidence.RecipientTo {
			return false
		}
		seen[target.RelationshipID] = true
	}
	return true
}

func (a *API) requestVendorForms(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	var input vendorFormBatchRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil || !validVendorFormBatch(input) {
		httpx.WriteError(w, http.StatusBadRequest, "vendor_form_request_invalid", "Choose between 1 and 50 vendor relationships and one To recipient for each request.")
		return
	}
	entity, ok := distributionLegalEntity(w, r, actor, input.LegalEntityID)
	if !ok {
		return
	}
	vendors, ok := a.thirdPartyService(w)
	if !ok {
		return
	}
	vendorActor, err := thirdPartyActor(r)
	if err != nil {
		writeThirdPartyError(w, err)
		return
	}
	// A tenant-wide identity must still read the relationship in the one
	// legal entity already selected and validated for this batch.
	vendorActor.LegalEntityID = entity
	if a.deps.Authority == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_form_authority_unavailable", "Vendor request responsibilities could not be checked. Try again when the authority service is available.")
		return
	}
	outcomes := make([]vendorFormRequestOutcome, 0, len(input.Targets))
	for _, target := range input.Targets {
		if target.Recipient.Type == evidence.RecipientExternalAudience {
			target.Recipient.AudienceHint = "Vendor contact"
		}
		outcome := vendorFormRequestOutcome{RelationshipID: target.RelationshipID, Status: "FAILED"}
		if _, err := vendors.GetRelationship(r.Context(), vendorActor, target.RelationshipID); err != nil {
			outcome.Error = "This vendor relationship is unavailable in your current access scope."
			outcomes = append(outcomes, outcome)
			continue
		}
		route, err := a.deps.Authority.Resolve(r.Context(), authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: entity, ObjectType: "VENDOR_RELATIONSHIP", ObjectID: target.RelationshipID, Responsibility: authority.ResponsibilityOwner, DecisionType: "forms.distribution.create", Materiality: 3})
		if err != nil || !route.AllowsPrincipal(actor.PrincipalID) {
			outcome.Error = "You do not hold the current responsibility to request this vendor's form."
			outcomes = append(outcomes, outcome)
			continue
		}
		prepared, err := service.Prepare(r.Context(), evidence.CreateDistributionInput{IdempotencyKey: "vendor-form:" + input.BatchID + ":" + target.RelationshipID, TenantID: actor.TenantID, LegalEntityID: entity, FormTemplateID: input.FormTemplateID, FormTemplateVersion: input.FormTemplateVersion, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: target.RelationshipID, Title: input.Title, Purpose: input.Purpose, AccessPolicy: input.AccessPolicy, EstimatedMinutes: input.EstimatedMinutes, Deadline: input.Deadline, RouteExpiresAt: input.RouteExpiresAt, CreatedBy: actor.PrincipalID, Recipients: []evidence.DistributionRecipientInput{target.Recipient}})
		if err != nil {
			outcome.Error = "The form request could not be prepared. Check its form revision, recipient and dates; use a new batch if you changed a previously saved request."
			outcomes = append(outcomes, outcome)
			continue
		}
		outcome.DistributionID = prepared.Distribution.ID
		outcome.DistributionState = prepared.Distribution.Status
		outcome.Status = "PREPARED"
		if prepared.Distribution.Status != evidence.DistributionDraft {
			outcome.Status = "CREATED"
			outcomes = append(outcomes, outcome)
			continue
		}
		if hasExternalTO(prepared.Recipients) {
			if a.deps.FormDistributionAccess == nil {
				outcome.Error = "The request is saved. Recipient access is unavailable; retry this batch to continue delivery."
				outcomes = append(outcomes, outcome)
				continue
			}
			if _, err := a.deps.FormDistributionAccess.EnsureDistributionAccessRoutes(r.Context(), prepared.Distribution.TenantID, entity, prepared.Distribution.ID, actor.PrincipalID); err != nil {
				outcome.Error = "The request is saved. Recipient access could not be prepared; retry this batch to continue delivery."
				outcomes = append(outcomes, outcome)
				continue
			}
		}
		if opened, err := service.Open(r.Context(), prepared.Distribution.TenantID, entity, prepared.Distribution.ID, prepared.Distribution.Version, actor.PrincipalID); err != nil {
			outcome.Error = "The request is saved. Delivery could not be started; retry this batch to check its current status."
		} else {
			outcome.Status = "CREATED"
			outcome.DistributionState = opened.Distribution.Status
		}
		outcomes = append(outcomes, outcome)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"batch_id": input.BatchID, "items": outcomes})
}
