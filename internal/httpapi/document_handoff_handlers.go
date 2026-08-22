package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) reviewDocumentProposalHandoff(w http.ResponseWriter, r *http.Request) {
	service, ok := a.documentImportService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	var input struct {
		Action                  documentimport.HandoffReviewAction `json:"action"`
		ExpectedDocumentVersion int64                              `json:"expected_document_version"`
		ExpectedHandoffVersion  int64                              `json:"expected_handoff_version"`
		Title                   string                             `json:"title,omitempty"`
		Statement               string                             `json:"statement,omitempty"`
		TargetType              documentimport.ConversionTarget    `json:"target_type,omitempty"`
		TargetProgramID         string                             `json:"target_program_id,omitempty"`
		TargetProgramVersion    int64                              `json:"target_program_version,omitempty"`
		Note                    string                             `json:"note,omitempty"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	documentID, proposalID := r.PathValue("id"), r.PathValue("proposal_id")
	document, loadErr := service.GetVisible(r.Context(), actor.TenantID, actor.LegalEntityID, documentID)
	if loadErr != nil {
		writeDocumentHandoffError(w, loadErr)
		return
	}
	proposal := documentProposal(document, proposalID)
	if proposal == nil {
		writeDocumentHandoffError(w, documentimport.ErrNotFound)
		return
	}
	if document.Version != input.ExpectedDocumentVersion || proposal.Handoff == nil || proposal.Handoff.Version != input.ExpectedHandoffVersion {
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "This proposal changed. Reload it before continuing.")
		return
	}
	if routeErr := a.requireDocumentHandoffRoute(r, actor, document, *proposal.Handoff); routeErr != nil {
		writeDocumentHandoffError(w, routeErr)
		return
	}
	if input.Action == documentimport.HandoffReviewSubmit {
		if a.deps.Continuity == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "continuity_unavailable", "Program continuity is unavailable.")
			return
		}
		program, programErr := a.deps.Continuity.GetProgram(r.Context(), actor.TenantID, strings.TrimSpace(input.TargetProgramID))
		if programErr != nil {
			writeDocumentHandoffError(w, programErr)
			return
		}
		if program.Program.Status == continuity.ProgramRetired {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "program_retired", "The target Program is retired. Choose a current Program before submitting this proposal for authorization.")
			return
		}
		if program.Program.Version != input.TargetProgramVersion {
			httpx.WriteError(w, http.StatusConflict, "program_version_conflict", "The target Program changed. Reload it before submitting this proposal for authorization.")
			return
		}
		if program.Program.LegalEntityID != "" && program.Program.LegalEntityID != document.LegalEntityID {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "program_scope_mismatch", "The target Program is outside this document's legal-entity scope.")
			return
		}
	}
	value, err := service.ReviewProposalHandoff(r.Context(), documentimport.HandoffReviewInput{
		TenantID: actor.TenantID, DocumentID: documentID, ProposalID: proposalID, ActorID: actor.PrincipalID,
		Action: input.Action, ExpectedDocumentVersion: input.ExpectedDocumentVersion, ExpectedHandoffVersion: input.ExpectedHandoffVersion,
		Title: input.Title, Statement: input.Statement, TargetType: input.TargetType,
		TargetProgramID: input.TargetProgramID, TargetProgramVersion: input.TargetProgramVersion, Note: input.Note,
	})
	if err != nil {
		writeDocumentHandoffError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.withDocumentHandoffRoutes(r, actor, value))
}

func (a *API) authorizeDocumentProposalHandoff(w http.ResponseWriter, r *http.Request) {
	service, ok := a.documentImportService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	var input struct {
		Action                  documentimport.HandoffAuthorizationAction `json:"action"`
		ExpectedDocumentVersion int64                                     `json:"expected_document_version"`
		ExpectedHandoffVersion  int64                                     `json:"expected_handoff_version"`
		Note                    string                                    `json:"note,omitempty"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	documentID, proposalID := r.PathValue("id"), r.PathValue("proposal_id")
	document, loadErr := service.GetVisible(r.Context(), actor.TenantID, actor.LegalEntityID, documentID)
	if loadErr != nil {
		writeDocumentHandoffError(w, loadErr)
		return
	}
	proposal := documentProposal(document, proposalID)
	if proposal == nil {
		writeDocumentHandoffError(w, documentimport.ErrNotFound)
		return
	}
	if document.Version != input.ExpectedDocumentVersion || proposal.Handoff == nil || proposal.Handoff.Version != input.ExpectedHandoffVersion {
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "This proposal changed. Reload it before continuing.")
		return
	}
	if routeErr := a.requireDocumentHandoffRoute(r, actor, document, *proposal.Handoff); routeErr != nil {
		writeDocumentHandoffError(w, routeErr)
		return
	}

	resultType, resultID := "", ""
	if input.Action == documentimport.HandoffAuthorizeApprove {
		if a.deps.Continuity == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "continuity_unavailable", "Program continuity is unavailable.")
			return
		}
		if err := validateDocumentHandoffApproval(proposal, actor.PrincipalID, input.Note); err != nil {
			writeDocumentHandoffError(w, err)
			return
		}
		objectID := documentimport.ConversionObjectID(proposal.Handoff.ID, proposal.Handoff.TargetType)
		code := documentimport.ConversionCode(proposal.Handoff.ID, proposal.Handoff.TargetType)
		sourceAnchor := proposalConversionAnchor(document.ID, *proposal)
		switch proposal.Handoff.TargetType {
		case documentimport.ConversionRequirement:
			modality, obligationActor, action, object := "MUST", "", "", ""
			if proposal.Obligation != nil {
				if value := strings.TrimSpace(proposal.Obligation.Modality); value != "" {
					modality = value
				}
				obligationActor, action, object = proposal.Obligation.Actor, proposal.Obligation.Action, proposal.Obligation.Object
			}
			_, loadErr = a.deps.Continuity.EnsureRequirement(r.Context(), objectID, continuity.AddRequirementInput{
				TenantID: actor.TenantID, ProgramID: proposal.Handoff.TargetProgramID, ExpectedVersion: proposal.Handoff.TargetProgramVersion,
				Code: code, Title: proposal.Handoff.DraftTitle, Statement: proposal.Handoff.DraftStatement, SourceAnchor: sourceAnchor,
				Modality: modality, Actor: obligationActor, Action: action, Object: object,
				Status: continuity.RequirementApproved, ActorID: actor.PrincipalID,
			})
			resultType = string(documentimport.ConversionRequirement)
		case documentimport.ConversionControlObjective:
			_, loadErr = a.deps.Continuity.EnsureControlObjective(r.Context(), objectID, continuity.AddControlObjectiveInput{
				TenantID: actor.TenantID, ProgramID: proposal.Handoff.TargetProgramID, ExpectedVersion: proposal.Handoff.TargetProgramVersion,
				Code: code, Name: proposal.Handoff.DraftTitle, Outcome: proposal.Handoff.DraftStatement,
				Status: continuity.ObjectiveActive, ActorID: actor.PrincipalID,
			})
			resultType = string(documentimport.ConversionControlObjective)
		default:
			loadErr = documentimport.ErrInvalidHandoff
		}
		if loadErr != nil {
			writeDocumentConversionError(w, loadErr)
			return
		}
		resultID = objectID
	}
	value, err := service.AuthorizeProposalHandoff(r.Context(), documentimport.HandoffAuthorizationInput{
		TenantID: actor.TenantID, DocumentID: documentID, ProposalID: proposalID, ActorID: actor.PrincipalID,
		Action: input.Action, ExpectedDocumentVersion: input.ExpectedDocumentVersion, ExpectedHandoffVersion: input.ExpectedHandoffVersion,
		Note: input.Note, ResultObjectType: resultType, ResultObjectID: resultID,
	})
	if err != nil {
		writeDocumentHandoffError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.withDocumentHandoffRoutes(r, actor, value))
}

func validateDocumentHandoffApproval(proposal *documentimport.Proposal, actorID, note string) error {
	actorID = strings.TrimSpace(actorID)
	if proposal == nil || proposal.Status != documentimport.ProposalAccepted || proposal.Handoff == nil || proposal.Handoff.Status != documentimport.HandoffAwaitingAuthorization || actorID == "" || strings.TrimSpace(note) == "" {
		return documentimport.ErrInvalidHandoff
	}
	handoff := proposal.Handoff
	if handoff.IntakePrincipalID == actorID || handoff.ReviewerPrincipalID == actorID {
		return documentimport.ErrHandoffSegregation
	}
	if strings.TrimSpace(handoff.DraftTitle) == "" || strings.TrimSpace(handoff.DraftStatement) == "" || strings.TrimSpace(handoff.TargetProgramID) == "" || handoff.TargetProgramVersion < 1 {
		return documentimport.ErrInvalidHandoff
	}
	if handoff.TargetType != documentimport.ConversionRequirement && handoff.TargetType != documentimport.ConversionControlObjective {
		return documentimport.ErrInvalidHandoff
	}
	return nil
}

func documentProposal(document documentimport.Document, proposalID string) *documentimport.Proposal {
	proposalID = strings.TrimSpace(proposalID)
	for index := range document.Proposals {
		if document.Proposals[index].ID == proposalID {
			return &document.Proposals[index]
		}
	}
	return nil
}

func proposalConversionAnchor(documentID string, proposal documentimport.Proposal) string {
	parts := []string{"document-import:" + strings.TrimSpace(documentID), "proposal:" + strings.TrimSpace(proposal.ID)}
	if proposal.Anchor.SectionID != "" {
		parts = append(parts, "section:"+proposal.Anchor.SectionID)
	}
	if proposal.Anchor.Page > 0 {
		parts = append(parts, "page:"+strconv.Itoa(proposal.Anchor.Page))
	}
	if proposal.Anchor.Sheet != "" {
		parts = append(parts, "sheet:"+proposal.Anchor.Sheet)
	}
	if proposal.Anchor.RowStart > 0 {
		row := strconv.Itoa(proposal.Anchor.RowStart)
		if proposal.Anchor.RowEnd > proposal.Anchor.RowStart {
			row += "-" + strconv.Itoa(proposal.Anchor.RowEnd)
		}
		parts = append(parts, "row:"+row)
	}
	return strings.Join(parts, "#")
}

func writeDocumentConversionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, continuity.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "program_version_conflict", "The target Program changed. Return this proposal for review before authorizing conversion.")
	case errors.Is(err, continuity.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "program_not_found", "The target Program no longer exists.")
	case errors.Is(err, continuity.ErrDuplicate):
		httpx.WriteError(w, http.StatusConflict, "conversion_identity_conflict", "The deterministic conversion identity is already bound to different canonical content.")
	case errors.Is(err, continuity.ErrInvalidState):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "program_not_mutable", "The target Program cannot accept new governance objects in its current state.")
	case errors.Is(err, documentimport.ErrInvalidHandoff):
		writeDocumentHandoffError(w, err)
	default:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "conversion_failed", "The canonical Program object could not be created. No approval was recorded.")
	}
}

func writeDocumentHandoffError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, documentimport.ErrNotFound), errors.Is(err, continuity.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "The document proposal or target Program was not found.")
	case errors.Is(err, documentimport.ErrVersionConflict), errors.Is(err, continuity.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "This proposal changed. Reload it before continuing.")
	case errors.Is(err, errDocumentHandoffRoutingUnresolved):
		httpx.WriteError(w, http.StatusConflict, "routing_unresolved", "This handoff does not have one independent directly assigned actor. Resolve routing before continuing.")
	case errors.Is(err, errDocumentHandoffNotAssigned):
		httpx.WriteError(w, http.StatusForbidden, "handoff_not_assigned", "This handoff is assigned to another authorized person.")
	case errors.Is(err, documentimport.ErrHandoffSegregation):
		httpx.WriteError(w, http.StatusForbidden, "independent_reviewer_required", "A different authorized person must perform this step.")
	case errors.Is(err, documentimport.ErrInvalidHandoff):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "handoff_invalid", "This proposal is not in a valid state for that action.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "handoff_failed", "The governed proposal handoff could not be updated.")
	}
}
