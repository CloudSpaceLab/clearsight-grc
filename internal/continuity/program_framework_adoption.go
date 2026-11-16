package continuity

import (
	"context"
	"errors"
	"strings"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

// FrameworkEditionReference is published catalog identity only. It is not
// an applicability decision, scored obligation or control implementation.
type FrameworkEditionReference struct {
	ID            string `json:"id"`
	PublisherCode string `json:"publisher_code"`
	FrameworkCode string `json:"framework_code"`
	EditionCode   string `json:"edition_code"`
	ReleaseSHA256 string `json:"release_sha256"`
}

// FrameworkEditionLookup is implemented by the trusted reference catalog.
// Never infer edition identity from an untrusted command payload.
type FrameworkEditionLookup interface {
	GetFrameworkEdition(context.Context, string) (FrameworkEditionReference, error)
}

type FrameworkAdoptionStatus string

const (
	FrameworkAdoptionDraft    FrameworkAdoptionStatus = "DRAFT"
	FrameworkAdoptionPending  FrameworkAdoptionStatus = "PENDING"
	FrameworkAdoptionActive   FrameworkAdoptionStatus = "ACTIVE"
	FrameworkAdoptionRejected FrameworkAdoptionStatus = "REJECTED"
)

// SaveFrameworkAdoptionDraft persists an owner's working proposal as a
// versioned Program event. A draft is not an adoption or an approval.
func (s *Service) SaveFrameworkAdoptionDraft(ctx context.Context, input ProposeFrameworkAdoptionInput) (ProgramAggregate, error) {
	if input.TemplateCode != "" {
		return s.saveStarterAdoption(ctx, input, true)
	}
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	maker, rationale := strings.TrimSpace(input.ActorID), strings.TrimSpace(input.Rationale)
	if !programComplianceActorMatches(ctx, input.TenantID, maker) || maker != aggregate.Program.OwnerPrincipalID || !validFrameworkAdoptionRationale(rationale) || strings.TrimSpace(input.EditionID) == "" || aggregate.Program.LegalEntityID == "" {
		return ProgramAggregate{}, ErrInvalidState
	}
	if s.frameworkEditionLookup == nil {
		return ProgramAggregate{}, ErrFrameworkEditionUnavailable
	}
	edition, err := s.frameworkEditionLookup.GetFrameworkEdition(ctx, input.EditionID)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if edition.ID != input.EditionID || edition.PublisherCode == "" || edition.FrameworkCode == "" || edition.EditionCode == "" || edition.ReleaseSHA256 == "" {
		return ProgramAggregate{}, ErrInvalidState
	}
	var draft FrameworkAdoption
	for _, item := range aggregate.FrameworkAdoptions {
		if item.PublisherCode != edition.PublisherCode || item.FrameworkCode != edition.FrameworkCode {
			continue
		}
		if item.Status == FrameworkAdoptionPending || item.Status == FrameworkAdoptionActive {
			return ProgramAggregate{}, ErrInvalidState
		}
		if item.Status == FrameworkAdoptionDraft {
			if item.EditionID != edition.ID {
				return ProgramAggregate{}, ErrInvalidState
			}
			draft = item
		}
	}
	if draft.ID == "" {
		id, idErr := platformid.NewUUIDv7()
		if idErr != nil {
			return ProgramAggregate{}, idErr
		}
		draft = FrameworkAdoption{ID: id, TenantID: input.TenantID, LegalEntityID: aggregate.Program.LegalEntityID, ProgramID: input.ProgramID, EditionID: edition.ID, PublisherCode: edition.PublisherCode, FrameworkCode: edition.FrameworkCode, EditionCode: edition.EditionCode, Status: FrameworkAdoptionDraft, ProposedBy: maker, ProposedAt: s.now().UTC()}
	}
	draft.ProposedBy = maker
	draft.Rationale = rationale
	draft.Version++
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ProgramID, input.ExpectedVersion, EventProgramFrameworkAdoptionDraftSaved, draft, maker, draft.ID)
}

type FrameworkAdoption struct {
	TemplateCode  string                  `json:"template_code,omitempty"`
	Configuration *StarterConfiguration   `json:"configuration,omitempty"`
	ID            string                  `json:"id"`
	TenantID      string                  `json:"tenant_id"`
	LegalEntityID string                  `json:"legal_entity_id"`
	ProgramID     string                  `json:"program_id"`
	EditionID     string                  `json:"edition_id"`
	PublisherCode string                  `json:"publisher_code"`
	FrameworkCode string                  `json:"framework_code"`
	EditionCode   string                  `json:"edition_code"`
	Status        FrameworkAdoptionStatus `json:"status"`
	ProposedBy    string                  `json:"proposed_by"`
	ReviewedBy    string                  `json:"reviewed_by,omitempty"`
	Rationale     string                  `json:"rationale"`
	ProposedAt    time.Time               `json:"proposed_at"`
	ReviewedAt    *time.Time              `json:"reviewed_at,omitempty"`
	EffectiveFrom *time.Time              `json:"effective_from,omitempty"`
	Version       int64                   `json:"version"`
}

type ProposeFrameworkAdoptionInput struct {
	TemplateCode    string                `json:"template_code,omitempty"`
	Configuration   *StarterConfiguration `json:"configuration,omitempty"`
	TenantID        string                `json:"tenant_id"`
	ProgramID       string                `json:"program_id"`
	EditionID       string                `json:"edition_id"`
	ExpectedVersion int64                 `json:"expected_version"`
	ActorID         string                `json:"actor_id"`
	Rationale       string                `json:"rationale"`
}

type DecideFrameworkAdoptionInput struct {
	TenantID        string `json:"tenant_id"`
	ProgramID       string `json:"program_id"`
	AdoptionID      string `json:"adoption_id"`
	ExpectedVersion int64  `json:"expected_version"`
	ActorID         string `json:"actor_id"`
	Rationale       string `json:"rationale"`
}

var ErrFrameworkEditionUnavailable = errors.New("approved framework edition catalog is unavailable")

// ConfigureFrameworkEditionLookup is internal service wiring. Without a
// trusted catalog resolver, all adoption proposals fail closed.
func (s *Service) ConfigureFrameworkEditionLookup(lookup FrameworkEditionLookup) {
	s.frameworkEditionLookup = lookup
}

func (s *Service) ProposeFrameworkAdoption(ctx context.Context, input ProposeFrameworkAdoptionInput) (ProgramAggregate, error) {
	if input.TemplateCode != "" {
		return s.saveStarterAdoption(ctx, input, false)
	}
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	maker, rationale := strings.TrimSpace(input.ActorID), strings.TrimSpace(input.Rationale)
	if !programComplianceActorMatches(ctx, input.TenantID, maker) ||
		maker != aggregate.Program.OwnerPrincipalID || !validFrameworkAdoptionRationale(rationale) ||
		strings.TrimSpace(input.EditionID) == "" || aggregate.Program.LegalEntityID == "" {
		return ProgramAggregate{}, ErrInvalidState
	}
	if s.frameworkEditionLookup == nil {
		return ProgramAggregate{}, ErrFrameworkEditionUnavailable
	}
	edition, err := s.frameworkEditionLookup.GetFrameworkEdition(ctx, input.EditionID)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if edition.ID != input.EditionID || edition.PublisherCode == "" || edition.FrameworkCode == "" ||
		edition.EditionCode == "" || edition.ReleaseSHA256 == "" {
		return ProgramAggregate{}, ErrInvalidState
	}
	var draft FrameworkAdoption
	// Edition upgrades are F4: initial adoption cannot displace an already
	// active or pending edition of this standard.
	for _, adoption := range aggregate.FrameworkAdoptions {
		if adoption.PublisherCode == edition.PublisherCode && adoption.FrameworkCode == edition.FrameworkCode && adoption.Status == FrameworkAdoptionDraft {
			if adoption.EditionID != edition.ID {
				return ProgramAggregate{}, ErrInvalidState
			}
			draft = adoption
		}
		if adoption.PublisherCode == edition.PublisherCode && adoption.FrameworkCode == edition.FrameworkCode &&
			(adoption.Status == FrameworkAdoptionPending || adoption.Status == FrameworkAdoptionActive) {
			return ProgramAggregate{}, ErrInvalidState
		}
	}
	id := draft.ID
	if id == "" {
		id, err = platformid.NewUUIDv7()
		if err != nil {
			return ProgramAggregate{}, err
		}
	}
	proposal := FrameworkAdoption{
		ID: id, TenantID: input.TenantID, LegalEntityID: aggregate.Program.LegalEntityID,
		ProgramID: input.ProgramID, EditionID: edition.ID,
		PublisherCode: edition.PublisherCode, FrameworkCode: edition.FrameworkCode, EditionCode: edition.EditionCode,
		Status: FrameworkAdoptionPending, ProposedBy: maker, Rationale: rationale,
		ProposedAt: s.now().UTC(), Version: 1,
	}
	if draft.ID != "" {
		proposal.Version = draft.Version + 1
	}
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ProgramID,
		input.ExpectedVersion, EventProgramFrameworkAdoptionProposed, proposal, maker, proposal.ID)
}

func (s *Service) ApproveFrameworkAdoption(ctx context.Context, input DecideFrameworkAdoptionInput) (ProgramAggregate, error) {
	return s.decideFrameworkAdoption(ctx, input, true)
}

func (s *Service) RejectFrameworkAdoption(ctx context.Context, input DecideFrameworkAdoptionInput) (ProgramAggregate, error) {
	return s.decideFrameworkAdoption(ctx, input, false)
}

func (s *Service) decideFrameworkAdoption(ctx context.Context, input DecideFrameworkAdoptionInput, approve bool) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	checker, rationale := strings.TrimSpace(input.ActorID), strings.TrimSpace(input.Rationale)
	if !programComplianceActorMatches(ctx, input.TenantID, checker) ||
		checker != aggregate.Program.AuthorityPrincipalID || !validFrameworkAdoptionRationale(rationale) {
		return ProgramAggregate{}, ErrInvalidState
	}
	var current FrameworkAdoption
	found := false
	for _, adoption := range aggregate.FrameworkAdoptions {
		if adoption.ID == input.AdoptionID && adoption.Status == FrameworkAdoptionPending {
			current, found = adoption, true
			break
		}
	}
	if !found {
		return ProgramAggregate{}, ErrNotFound
	}
	if checker == current.ProposedBy {
		return ProgramAggregate{}, ErrMakerChecker
	}
	now := s.now().UTC()
	if now.Before(current.ProposedAt) {
		return ProgramAggregate{}, ErrInvalidState
	}
	current.Version++
	current.ReviewedBy = checker
	current.ReviewedAt = &now
	eventType := EventProgramFrameworkAdoptionRejected
	current.Status = FrameworkAdoptionRejected
	if approve && current.TemplateCode == "" {
		if s.frameworkEditionLookup == nil {
			return ProgramAggregate{}, ErrFrameworkEditionUnavailable
		}
		edition, err := s.frameworkEditionLookup.GetFrameworkEdition(ctx, current.EditionID)
		if err != nil {
			return ProgramAggregate{}, err
		}
		if edition.PublisherCode != current.PublisherCode || edition.FrameworkCode != current.FrameworkCode ||
			edition.EditionCode != current.EditionCode || edition.ReleaseSHA256 == "" {
			return ProgramAggregate{}, ErrInvalidState
		}
		current.Status = FrameworkAdoptionActive
		current.EffectiveFrom = &now
		eventType = EventProgramFrameworkAdoptionApproved
	}

	if approve && current.TemplateCode != "" {
		catalog, ok := s.frameworkEditionLookup.(StarterTemplateLookup)
		if !ok || current.Configuration == nil {
			return ProgramAggregate{}, ErrFrameworkEditionUnavailable
		}
		// Native Forms approval is checked on the selected historical revisions, not a newer form.
		if validator, ok := catalog.(interface {
			ValidateStarterSnapshot(context.Context, string, string, StarterConfiguration) error
		}); ok {
			if err := validator.ValidateStarterSnapshot(ctx, current.TenantID, current.LegalEntityID, *current.Configuration); err != nil {
				return ProgramAggregate{}, err
			}
		} else {
			return ProgramAggregate{}, ErrFrameworkEditionUnavailable
		}
		current.Status = FrameworkAdoptionActive
		current.EffectiveFrom = &now
		eventType = EventProgramFrameworkAdoptionApproved
	}
	// Preserve the original proposal rationale; the decision rationale is
	// recorded in its immutable event envelope, not retroactively overwritten.
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ProgramID,
		input.ExpectedVersion, eventType,
		frameworkAdoptionDecision{Adoption: current, Rationale: rationale}, checker, current.ID)
}

type frameworkAdoptionDecision struct {
	Adoption  FrameworkAdoption `json:"adoption"`
	Rationale string            `json:"rationale"`
}

func validFrameworkAdoptionRationale(value string) bool {
	return len(value) > 0 && len(value) <= 2000 && strings.TrimSpace(value) == value
}

func upsertFrameworkAdoption(values []FrameworkAdoption, updated FrameworkAdoption) []FrameworkAdoption {
	for i := range values {
		if values[i].ID == updated.ID {
			values[i] = updated
			return values
		}
	}
	return append(values, updated)
}
