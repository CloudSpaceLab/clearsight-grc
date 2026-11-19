package continuity

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"strings"
)

// StarterRequirement is a Program-owned snapshot of a bank starter obligation.
// It never changes the framework library and never asserts standards coverage.
type StarterRequirement struct {
	Code              string               `json:"code"`
	Title             string               `json:"title"`
	OriginalTitle     string               `json:"original_title"`
	Statement         string               `json:"statement"`
	OriginalStatement string               `json:"original_statement"`
	Reference         string               `json:"reference"`
	Version           int64                `json:"version"`
	FormID            string               `json:"form_template_id"`
	FormVersion       int64                `json:"form_template_version"`
	FormName          string               `json:"form_name"`
	FormStatus        string               `json:"form_status"`
	FormApproved      bool                 `json:"form_approved"`
	GroupID           string               `json:"group_id"`
	GroupTitle        string               `json:"group_title"`
	Fields            []formcontract.Field `json:"fields"`
	Decision          string               `json:"decision"`
	Reason            string               `json:"reason"`
}
type StarterConfiguration struct {
	Requirements       []StarterRequirement `json:"requirements"`
	OwnerID            string               `json:"owner_id"`
	RespondentScopeIDs []string             `json:"respondent_scope_ids"`
}
type StarterTemplateLookup interface {
	StarterRequirements(context.Context, string, string, string) ([]StarterRequirement, error)
	ValidateStarterRespondents(context.Context, string, string, []string) error
}

func (s *Service) saveStarterAdoption(ctx context.Context, input ProposeFrameworkAdoptionInput, draft bool) (ProgramAggregate, error) {
	a, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	actor := strings.TrimSpace(input.ActorID)
	if !programComplianceActorMatches(ctx, input.TenantID, actor) || actor != a.Program.OwnerPrincipalID || !validFrameworkAdoptionRationale(input.Rationale) || input.EditionID != "" || input.Configuration == nil {
		return ProgramAggregate{}, ErrInvalidState
	}
	catalog, ok := s.frameworkEditionLookup.(StarterTemplateLookup)
	if !ok {
		return ProgramAggregate{}, ErrFrameworkEditionUnavailable
	}
	source, err := catalog.StarterRequirements(ctx, input.TenantID, a.Program.LegalEntityID, input.TemplateCode)
	if err != nil {
		return ProgramAggregate{}, err
	}
	// A saved draft keeps its reviewed source revision even when the bank library changes.
	for _, saved := range a.FrameworkAdoptions {
		if saved.TemplateCode == input.TemplateCode && saved.Status == FrameworkAdoptionDraft && saved.Configuration != nil {
			latest := source
			source = append([]StarterRequirement{}, saved.Configuration.Requirements...)
			// An owner may explicitly select a currently configured bank link.
			// Merely publishing a newer Forms revision never updates the draft.
			for i, old := range source {
				for _, edit := range input.Configuration.Requirements {
					if edit.Code != old.Code {
						continue
					}
					if edit.Version != old.Version || edit.FormID != old.FormID || edit.FormVersion != old.FormVersion || edit.GroupID != old.GroupID {
						for _, candidate := range latest {
							if candidate.Code == old.Code && candidate.Version == edit.Version && candidate.FormID == edit.FormID && candidate.FormVersion == edit.FormVersion && candidate.GroupID == edit.GroupID {
								source[i] = candidate
							}
						}
					}
				}
			}
			for i := range source {
				source[i].Statement = source[i].OriginalStatement
				if source[i].OriginalTitle != "" {
					source[i].Title = source[i].OriginalTitle
				}
			}
		}
	}
	config, err := prepareStarterConfiguration(source, *input.Configuration, a.Program.OwnerPrincipalID, draft)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if !draft {
		validator, ok := catalog.(interface {
			ValidateStarterSnapshot(context.Context, string, string, StarterConfiguration) error
		})
		if !ok {
			return ProgramAggregate{}, ErrFrameworkEditionUnavailable
		}
		if err = validator.ValidateStarterSnapshot(ctx, input.TenantID, a.Program.LegalEntityID, config); err != nil {
			return ProgramAggregate{}, err
		}
	}
	if err = catalog.ValidateStarterRespondents(ctx, input.TenantID, a.Program.LegalEntityID, config.RespondentScopeIDs); err != nil {
		return ProgramAggregate{}, err
	}
	var value FrameworkAdoption
	for _, v := range a.FrameworkAdoptions {
		if v.TemplateCode == input.TemplateCode {
			if v.Status == FrameworkAdoptionPending || v.Status == FrameworkAdoptionActive {
				return ProgramAggregate{}, ErrInvalidState
			}
			if v.Status == FrameworkAdoptionDraft {
				value = v
			}
		}
	}
	if value.ID == "" {
		value.ID, err = platformid.NewUUIDv7()
		if err != nil {
			return ProgramAggregate{}, err
		}
	}
	value.TenantID = input.TenantID
	value.LegalEntityID = a.Program.LegalEntityID
	value.ProgramID = input.ProgramID
	value.TemplateCode = input.TemplateCode
	value.PublisherCode = "STARTER"
	value.FrameworkCode = input.TemplateCode
	value.EditionCode = "starter-1"
	value.Configuration = &config
	value.ProposedBy = actor
	value.ProposedAt = s.now().UTC()
	value.Rationale = input.Rationale
	value.Version++
	value.Status = FrameworkAdoptionDraft
	event := EventProgramFrameworkAdoptionDraftSaved
	if !draft {
		value.Status = FrameworkAdoptionPending
		event = EventProgramFrameworkAdoptionProposed
	}
	return s.applyProgramValueAndResult(ctx, a, input.TenantID, input.ProgramID, input.ExpectedVersion, event, value, actor, value.ID)
}
func prepareStarterConfiguration(source []StarterRequirement, input StarterConfiguration, owner string, draft bool) (StarterConfiguration, error) {
	if len(source) == 0 || len(source) > 100 || len(input.Requirements) != len(source) || input.OwnerID != owner || len(input.RespondentScopeIDs) > 50 || (!draft && len(input.RespondentScopeIDs) == 0) {
		return StarterConfiguration{}, ErrInvalidState
	}
	byCode := map[string]StarterRequirement{}
	for _, r := range source {
		byCode[r.Code] = r
	}
	result := StarterConfiguration{OwnerID: owner, RespondentScopeIDs: append([]string{}, input.RespondentScopeIDs...), Requirements: make([]StarterRequirement, 0, len(source))}
	seen := map[string]bool{}
	for _, edit := range input.Requirements {
		r, ok := byCode[edit.Code]
		if !ok || seen[edit.Code] || edit.Version != r.Version || edit.FormID != r.FormID || edit.FormVersion != r.FormVersion || edit.GroupID != r.GroupID {
			return StarterConfiguration{}, ErrVersionConflict
		}
		seen[edit.Code] = true
		if strings.TrimSpace(edit.Title) == "" || len(edit.Title) > 200 || strings.TrimSpace(edit.Statement) == "" || len(edit.Statement) > 2000 || len(edit.Reason) > 2000 {
			return StarterConfiguration{}, ErrInvalidState
		}
		if edit.Decision != "KEEP" && edit.Decision != "TAILOR" && edit.Decision != "REVIEW" && !(draft && edit.Decision == "") {
			return StarterConfiguration{}, ErrInvalidState
		}
		if (edit.Decision == "TAILOR" || edit.Decision == "REVIEW") && strings.TrimSpace(edit.Reason) == "" {
			return StarterConfiguration{}, ErrInvalidState
		}
		if edit.Decision == "KEEP" && (edit.Title != r.Title || edit.Statement != r.Statement) {
			return StarterConfiguration{}, ErrInvalidState
		}
		if !draft && !r.FormApproved {
			return StarterConfiguration{}, ErrInvalidState
		}
		r.OriginalStatement = r.Statement
		r.OriginalTitle = r.Title
		r.Title = edit.Title
		r.Statement = edit.Statement
		r.Decision = edit.Decision
		r.Reason = edit.Reason
		result.Requirements = append(result.Requirements, r)
	}
	return result, nil
}

// starterProgramRequirements materializes only independently approved adoptions.
// Applicability flags remain draft requirements for a further explicit decision.
func starterProgramRequirements(a FrameworkAdoption) []Requirement {
	if a.TemplateCode == "" || a.Status != FrameworkAdoptionActive || a.Configuration == nil || a.EffectiveFrom == nil {
		return nil
	}
	values := make([]Requirement, 0, len(a.Configuration.Requirements))
	for _, r := range a.Configuration.Requirements {
		digest := sha256.Sum256([]byte(a.ID + ":" + r.Code))
		d := digest[:16]
		id := fmt.Sprintf("%x-%x-%x-%x-%x", d[0:4], d[4:6], d[6:8], d[8:10], d[10:16])
		status := RequirementApproved
		if r.Decision == "REVIEW" {
			status = RequirementDraft
		}
		values = append(values, Requirement{ID: id, TenantID: a.TenantID, ProgramID: a.ProgramID, Code: a.TemplateCode + "-" + r.Code, Title: r.Title, Statement: r.Statement, SourceAnchor: r.Reference, Modality: "MUST", Status: status, EffectiveFrom: *a.EffectiveFrom, CreatedAt: *a.EffectiveFrom, Version: 1})
	}
	return values
}
func applyStarterRequirements(a *ProgramAggregate, adoption FrameworkAdoption) {
	for _, r := range starterProgramRequirements(adoption) {
		exists := false
		for _, saved := range a.Requirements {
			if saved.ID == r.ID {
				exists = true
				break
			}
		}
		if !exists {
			a.Requirements = append(a.Requirements, r)
		}
	}
}
