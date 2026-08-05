package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) continuityService(w http.ResponseWriter) (*continuity.Service, bool) {
	if a.deps.Continuity == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "continuity_unavailable", "Programs and issue tracking are unavailable.")
		return nil, false
	}
	return a.deps.Continuity, true
}

func (a *API) listPrograms(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListPrograms(r.Context(), tenant, limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "programs_failed", "Programs could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values, "generated_at": time.Now().UTC()})
}

func (a *API) createProgram(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.CreateProgramInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateProgram(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) getProgram(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	value, err := service.GetProgram(r.Context(), tenant, r.PathValue("id"))
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) getProgramHistory(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	at, ok := requiredTimeQuery(w, r, "at")
	if !ok {
		return
	}
	value, err := service.ProgramAt(r.Context(), tenant, r.PathValue("id"), at)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) transitionProgram(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.ProgramTransitionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ID = r.PathValue("id")
	value, err := service.TransitionProgram(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) addProgramRequirement(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddRequirementInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.AddRequirement(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) determineProgramApplicability(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.DetermineApplicabilityInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.DetermineApplicability(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) addProgramControlObjective(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddControlObjectiveInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.AddControlObjective(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) addProgramControlImplementation(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddControlImplementationInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.AddControlImplementation(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) linkProgramRequirementControl(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.LinkRequirementControlInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.LinkRequirementControl(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) addProgramEvidenceContract(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddEvidenceContractInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.AddEvidenceContract(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) recordProgramEvidenceAssessment(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.RecordEvidenceAssessmentInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	value, err := service.RecordEvidenceAssessment(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) applyProgramTrigger(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.Trigger
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ProgramID = r.PathValue("id")
	program, matter, inserted, err := service.ApplyTrigger(r.Context(), input)
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"program": program, "matter": matter, "inserted": inserted})
}

func (a *API) listMatters(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	values, err := service.ListMatters(r.Context(), tenant, r.URL.Query().Get("status"), limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "matters_failed", "Issues and changes could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values, "generated_at": time.Now().UTC()})
}

func (a *API) createMatter(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.CreateMatterInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateMatter(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) getMatter(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	value, err := service.GetMatter(r.Context(), tenant, r.PathValue("id"))
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) getMatterHistory(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	at, ok := requiredTimeQuery(w, r, "at")
	if !ok {
		return
	}
	value, err := service.MatterAt(r.Context(), tenant, r.PathValue("id"), at)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) addMatterLink(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddMatterLinkInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.AddMatterLink(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) transitionMatter(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.TransitionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.ID = r.PathValue("id")
	value, err := service.TransitionMatter(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) addMatterDecision(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddDecisionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.AddDecision(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) addMatterAction(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddActionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.AddAction(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) transitionMatterAction(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.TransitionActionInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	input.ActionID = r.PathValue("action_id")
	value, err := service.TransitionAction(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func (a *API) addMatterVerificationContract(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddVerificationContractInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.AddVerificationContract(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) recordMatterVerificationResult(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.RecordVerificationResultInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.RecordVerificationResult(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) addMatterResponse(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.AddResponsePackageInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	value, err := service.AddResponsePackage(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusCreated)
}

func (a *API) transitionMatterResponse(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	var input continuity.TransitionResponseInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input.MatterID = r.PathValue("id")
	input.ResponseID = r.PathValue("response_id")
	value, err := service.TransitionResponsePackage(r.Context(), input)
	writeContinuityResult(w, value, err, http.StatusOK)
}

func requiredTimeQuery(w http.ResponseWriter, r *http.Request, name string) (time.Time, bool) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_query", name+" is required")
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_time", name+" must be an RFC3339 timestamp")
		return time.Time{}, false
	}
	return parsed, true
}

func writeContinuityResult[T any](w http.ResponseWriter, value T, err error, success int) {
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	httpx.WriteJSON(w, success, value)
}

func writeContinuityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, continuity.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "The requested program or issue was not found in this bank scope.")
	case errors.Is(err, continuity.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "This record changed. Reload it before saving your update.")
	case errors.Is(err, continuity.ErrDuplicate), errors.Is(err, continuity.ErrTriggerDuplicate):
		httpx.WriteError(w, http.StatusConflict, "duplicate", "An active record already exists for this code or trigger.")
	case errors.Is(err, continuity.ErrInvalidState):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "state_not_allowed", "That status change is not allowed from the current state.")
	case errors.Is(err, continuity.ErrClosureBlocked):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "closure_blocked", err.Error())
	case err != nil:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "continuity_invalid", err.Error())
	}
}
