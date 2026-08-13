package documentcoverage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const EventCoverageComparisonRequested = "DocumentCoverageComparisonRequested"

type DocumentReader interface {
	Get(context.Context, string, string) (documentimport.Document, error)
}

type ContinuityReader interface {
	ListPrograms(context.Context, string, int) ([]continuity.ProgramAggregate, error)
	ListMatters(context.Context, string, string, int) ([]continuity.MatterAggregate, error)
}

type Service struct {
	repo       Repository
	documents  DocumentReader
	continuity ContinuityReader
	now        func() time.Time
}

func NewService(repo Repository, documents DocumentReader, continuityReader ContinuityReader) *Service {
	return &Service{repo: repo, documents: documents, continuity: continuityReader, now: time.Now}
}

func (s *Service) Process(ctx context.Context, tenant, documentID string) (Assessment, error) {
	if s == nil || s.repo == nil || s.documents == nil || s.continuity == nil {
		return Assessment{}, fmt.Errorf("document coverage is unavailable")
	}
	tenant = strings.TrimSpace(tenant)
	documentID = strings.TrimSpace(documentID)
	document, err := s.documents.Get(ctx, tenant, documentID)
	if err != nil {
		if errors.Is(err, documentimport.ErrNotFound) {
			return Assessment{}, ErrNotFound
		}
		return Assessment{}, err
	}
	if document.ExtractionStatus != documentimport.ExtractionExtracted {
		return Assessment{}, ErrDocumentNotReady
	}
	programs, hash, err := s.programSnapshots(ctx, tenant)
	if err != nil {
		return Assessment{}, err
	}
	now := s.now().UTC()
	proposed := Assessment{
		TenantID: tenant, LegalEntityID: document.LegalEntityID, DocumentID: document.ID, DocumentSHA256: document.SHA256,
		Status: AssessmentComparing, AnalyzerVersion: AnalyzerVersion, MatcherVersion: MatcherVersion,
		ScoringPolicyVersion: ScoringPolicyVersion, ProgramSnapshotHash: hash,
		Candidates: []Candidate{}, Reviews: []ReviewRecord{}, Suggestions: []Suggestion{}, Limitations: []string{},
		AssessedAt: now, CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	if current, currentErr := s.repo.Current(ctx, tenant, documentID); currentErr == nil && sameAssessmentTuple(current, proposed) && current.Status == AssessmentReady {
		return current, nil
	}
	assessmentID, err := id.NewUUIDv7()
	if err != nil {
		return Assessment{}, err
	}
	proposed.ID = assessmentID
	started, err := s.repo.BeginVersion(ctx, proposed)
	if err != nil {
		return Assessment{}, err
	}
	if started.Status == AssessmentReady {
		return started, nil
	}
	evaluation := Evaluate(candidatesFromDocument(document), programs)
	started.Status = AssessmentReady
	started.Candidates = evaluation.Candidates
	started.Suggestions = evaluation.Suggestions
	started.Metrics = evaluation.Metrics
	started.AssessedAt = now
	started.UpdatedAt = now
	completed, err := s.repo.CompleteVersion(ctx, started, started.Version)
	if err != nil {
		return Assessment{}, err
	}
	return completed, nil
}

func (s *Service) Get(ctx context.Context, input ReadInput) (View, error) {
	document, err := s.documents.Get(ctx, strings.TrimSpace(input.TenantID), strings.TrimSpace(input.DocumentID))
	if err != nil {
		if errors.Is(err, documentimport.ErrNotFound) {
			return View{}, ErrNotFound
		}
		return View{}, err
	}
	assessment, err := s.repo.Current(ctx, input.TenantID, input.DocumentID)
	if errors.Is(err, ErrNotFound) {
		status := ViewPending
		if document.ExtractionStatus == documentimport.ExtractionExtracted {
			status = ViewComparing
		}
		return View{Assessment: Assessment{TenantID: input.TenantID, LegalEntityID: document.LegalEntityID, DocumentID: document.ID, DocumentSHA256: document.SHA256, Status: AssessmentPending, Version: 0}, Status: status, Matters: []MatterContext{}}, nil
	}
	if err != nil {
		return View{}, err
	}
	programs, currentHash, err := s.programSnapshots(ctx, input.TenantID)
	if err != nil {
		return View{}, err
	}
	_ = programs
	viewStatus := assessmentViewStatus(assessment.Status)
	if assessment.ProgramSnapshotHash != currentHash {
		viewStatus = ViewStale
	}
	matters, err := s.visibleMatterContext(ctx, input, assessment.Candidates)
	if err != nil {
		return View{}, err
	}
	return View{Assessment: assessment, Status: viewStatus, Matters: matters}, nil
}

func (s *Service) Review(ctx context.Context, input ReviewInput) (Assessment, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.DocumentID = strings.TrimSpace(input.DocumentID)
	input.ReviewerID = strings.TrimSpace(input.ReviewerID)
	if input.TenantID == "" || input.DocumentID == "" || input.ReviewerID == "" || input.ExpectedVersion < 1 || len(input.Decisions) == 0 || len(input.Decisions) > 50 {
		return Assessment{}, ErrInvalidReview
	}
	current, err := s.repo.Current(ctx, input.TenantID, input.DocumentID)
	if err != nil {
		return Assessment{}, err
	}
	if current.Version != input.ExpectedVersion {
		return Assessment{}, ErrVersionConflict
	}
	programs, hash, err := s.programSnapshots(ctx, input.TenantID)
	if err != nil {
		return Assessment{}, err
	}
	if hash != current.ProgramSnapshotHash {
		return Assessment{}, ErrStaleAssessment
	}
	byID := make(map[string]int, len(current.Candidates))
	for index := range current.Candidates {
		byID[current.Candidates[index].ID] = index
	}
	now := s.now().UTC()
	seen := map[string]struct{}{}
	for _, decision := range input.Decisions {
		decision.CandidateID = strings.TrimSpace(decision.CandidateID)
		decision.MatchID = strings.TrimSpace(decision.MatchID)
		decision.Reason = strings.TrimSpace(decision.Reason)
		index, ok := byID[decision.CandidateID]
		if !ok {
			return Assessment{}, ErrInvalidReview
		}
		if _, duplicate := seen[decision.CandidateID]; duplicate {
			return Assessment{}, ErrInvalidReview
		}
		seen[decision.CandidateID] = struct{}{}
		candidate := &current.Candidates[index]
		switch decision.Decision {
		case DecisionAccept:
			if decision.MatchID == "" || !hasMatch(*candidate, decision.MatchID) {
				return Assessment{}, ErrInvalidReview
			}
		case DecisionReject:
		case DecisionNotApplicable:
			if decision.Reason == "" {
				return Assessment{}, ErrInvalidReview
			}
		default:
			return Assessment{}, ErrInvalidReview
		}
		candidate.Review = &ReviewDecision{Decision: decision.Decision, MatchID: decision.MatchID, Reason: decision.Reason, ReviewerID: input.ReviewerID, ReviewedAt: now}
		current.Reviews = append(current.Reviews, ReviewRecord{
			CandidateID: decision.CandidateID, Decision: decision.Decision, MatchID: decision.MatchID,
			Reason: decision.Reason, ReviewerID: input.ReviewerID, ReviewedAt: now,
		})
	}
	evaluated := Evaluate(current.Candidates, programs)
	current.Candidates = evaluated.Candidates
	current.Suggestions = evaluated.Suggestions
	current.Metrics = evaluated.Metrics
	current.UpdatedAt = now
	return s.repo.Review(ctx, current, input.ExpectedVersion)
}

func (s *Service) Recompare(ctx context.Context, tenant, documentID string) error {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(documentID) == "" {
		return ErrNotFound
	}
	if _, err := s.documents.Get(ctx, tenant, documentID); err != nil {
		if errors.Is(err, documentimport.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return s.repo.QueueRecompare(ctx, tenant, documentID)
}

func (s *Service) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if s == nil || event.AggregateType != "DOCUMENT_IMPORT" {
		return nil
	}
	if event.EventType != documentimport.EventDocumentProcessingRequested && event.EventType != EventCoverageComparisonRequested {
		return nil
	}
	_, err := s.Process(ctx, event.TenantID, event.AggregateID)
	if errors.Is(err, ErrDocumentNotReady) {
		return nil
	}
	return err
}

func (s *Service) programSnapshots(ctx context.Context, tenant string) ([]ProgramSnapshot, string, error) {
	aggregates, err := s.continuity.ListPrograms(ctx, tenant, 100)
	if err != nil {
		return nil, "", err
	}
	now := s.now().UTC()
	programs := make([]ProgramSnapshot, 0, len(aggregates))
	for _, aggregate := range aggregates {
		if aggregate.Program.Status == continuity.ProgramRetired {
			continue
		}
		coverage := continuity.CurrentRequirementCoverage(aggregate, now)
		program := ProgramSnapshot{
			TenantID: aggregate.Program.TenantID, LegalEntityID: aggregate.Program.LegalEntityID,
			ProgramID: aggregate.Program.ID, Code: aggregate.Program.Code, Name: aggregate.Program.Name,
			Type: aggregate.Program.Type, Status: aggregate.Program.Status, Jurisdiction: aggregate.Program.Jurisdiction,
			Regulator: regulatorFromScope(aggregate.Program.Scope), Version: aggregate.Program.Version, Requirements: []RequirementTarget{},
		}
		for _, requirement := range aggregate.Requirements {
			if requirement.Status != continuity.RequirementApproved {
				continue
			}
			parsed := documentimport.ParseObligation(requirement.Statement, "REQUIREMENT_CANDIDATE")
			program.Requirements = append(program.Requirements, RequirementTarget{
				ID: requirement.ID, Code: requirement.Code, Title: requirement.Title, Statement: requirement.Statement,
				SourceAnchor: requirement.SourceAnchor, Status: requirement.Status, Version: requirement.Version,
				Modality: requirement.Modality, Actor: requirement.Actor, Action: requirement.Action, Object: requirement.Object,
				Citations: parsed.Citations, Dates: parsed.Dates, Topics: parsed.Topics,
				Applicability: coverage[requirement.ID].Applicability, Coverage: coverage[requirement.ID],
			})
		}
		sort.Slice(program.Requirements, func(i, j int) bool { return program.Requirements[i].ID < program.Requirements[j].ID })
		programs = append(programs, program)
	}
	sort.Slice(programs, func(i, j int) bool { return programs[i].ProgramID < programs[j].ProgramID })
	raw, err := json.Marshal(programs)
	if err != nil {
		return nil, "", err
	}
	digest := sha256.Sum256(raw)
	return programs, hex.EncodeToString(digest[:]), nil
}

func candidatesFromDocument(document documentimport.Document) []Candidate {
	jurisdiction := inferJurisdiction(document)
	regulator := inferRegulator(document)
	programType := inferProgramType(document)
	values := make([]Candidate, 0, len(document.Proposals))
	for _, proposal := range document.Proposals {
		if proposal.Obligation == nil {
			continue
		}
		obligation := proposal.Obligation
		values = append(values, Candidate{
			ID: proposal.ID, Fingerprint: obligation.Fingerprint, Eligible: obligation.Eligible,
			Statement: proposal.Statement, Anchor: proposal.Anchor, Modality: obligation.Modality,
			Actor: obligation.Actor, Action: obligation.Action, Object: obligation.Object,
			Citations: append([]string(nil), obligation.Citations...), Dates: append([]string(nil), obligation.Dates...),
			Topics: append([]string(nil), obligation.Topics...), Uncertainty: append([]string(nil), obligation.Uncertainty...),
			TenantID: document.TenantID, LegalEntityID: document.LegalEntityID, Jurisdiction: jurisdiction,
			Regulator: regulator, ProgramType: programType,
		})
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	return values
}

func (s *Service) visibleMatterContext(ctx context.Context, input ReadInput, candidates []Candidate) ([]MatterContext, error) {
	matters, err := s.continuity.ListMatters(ctx, input.TenantID, "", 100)
	if err != nil {
		return nil, err
	}
	values := make([]MatterContext, 0)
	for _, aggregate := range matters {
		if !continuity.MatterVisibleTo(aggregate.Matter, input.PrincipalID) {
			continue
		}
		for _, candidate := range candidates {
			score := similarity(candidate.Topics, tokenize(aggregate.Matter.Title+" "+aggregate.Matter.Summary))
			if score < .2 {
				continue
			}
			values = append(values, MatterContext{
				CandidateID: candidate.ID, MatterID: aggregate.Matter.ID, Reference: aggregate.Matter.Reference,
				Type: aggregate.Matter.Type, Status: aggregate.Matter.Status, Title: aggregate.Matter.Title,
				Summary: aggregate.Matter.Summary, Score: score,
			})
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Score != values[j].Score {
			return values[i].Score > values[j].Score
		}
		if values[i].CandidateID != values[j].CandidateID {
			return values[i].CandidateID < values[j].CandidateID
		}
		return values[i].MatterID < values[j].MatterID
	})
	return values, nil
}

func assessmentViewStatus(status AssessmentStatus) ViewStatus {
	switch status {
	case AssessmentComparing:
		return ViewComparing
	case AssessmentReady:
		return ViewReady
	case AssessmentPartial:
		return ViewPartial
	case AssessmentFailed:
		return ViewFailed
	default:
		return ViewPending
	}
}

func hasMatch(candidate Candidate, matchID string) bool {
	for _, match := range candidate.Matches {
		if match.ID == matchID {
			return true
		}
	}
	return false
}

func regulatorFromScope(scope json.RawMessage) string {
	var value map[string]any
	if json.Unmarshal(scope, &value) != nil {
		return ""
	}
	for _, key := range []string{"regulator", "authority"} {
		if text, ok := value[key].(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func documentText(document documentimport.Document) string {
	parts := []string{document.FileName, document.Purpose}
	for _, proposal := range document.Proposals {
		parts = append(parts, proposal.Statement)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func inferJurisdiction(document documentimport.Document) string {
	text := documentText(document)
	switch {
	case strings.Contains(text, "nigeria"), strings.Contains(text, "ndpc"), strings.Contains(text, "ndpa"):
		return "Nigeria"
	case strings.Contains(text, "united states"), strings.Contains(text, "federal reserve"):
		return "United States"
	case strings.Contains(text, "united kingdom"), strings.Contains(text, "bank of england"):
		return "United Kingdom"
	default:
		return ""
	}
}

func inferRegulator(document documentimport.Document) string {
	text := documentText(document)
	switch {
	case strings.Contains(text, "ndpc"), strings.Contains(text, "data protection commission"):
		return "Nigeria Data Protection Commission"
	case strings.Contains(text, "federal reserve"):
		return "Federal Reserve"
	case strings.Contains(text, "bank of england"):
		return "Bank of England"
	default:
		return ""
	}
}

func inferProgramType(document documentimport.Document) string {
	text := documentText(document)
	switch {
	case strings.Contains(text, "data protection"), strings.Contains(text, "personal data"), strings.Contains(text, "controller"):
		return "PRIVACY"
	case strings.Contains(text, "cybersecurity"):
		return "CYBERSECURITY"
	case strings.Contains(text, "operational resilience"):
		return "OPERATIONAL_RESILIENCE"
	default:
		return ""
	}
}

var _ workflowruntime.Publisher = (*Service)(nil)
