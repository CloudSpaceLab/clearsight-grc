package documentcoverage

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
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

type continuityCommander interface {
	ContinuityReader
	AddRequirement(context.Context, continuity.AddRequirementInput) (continuity.ProgramAggregate, error)
	CreateMatter(context.Context, continuity.CreateMatterInput) (continuity.MatterAggregate, error)
	CreateProgram(context.Context, continuity.CreateProgramInput) (continuity.ProgramAggregate, error)
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
	page, next, pageErr := paginateAssessment(assessment, input.Cursor, input.Limit)
	if pageErr != nil {
		return View{}, pageErr
	}
	visibleCandidates := make(map[string]struct{}, len(page.Candidates))
	for _, candidate := range page.Candidates {
		visibleCandidates[candidate.ID] = struct{}{}
	}
	filteredMatters := make([]MatterContext, 0, len(matters))
	for _, matter := range matters {
		if _, ok := visibleCandidates[matter.CandidateID]; ok {
			filteredMatters = append(filteredMatters, matter)
		}
	}
	return View{Assessment: page, Status: viewStatus, Matters: filteredMatters, NextCursor: next}, nil
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

func (s *Service) PrepareSuggestion(ctx context.Context, input ApplySuggestionInput) (PreparedSuggestion, error) {
	current, programs, err := s.currentSuggestionContext(ctx, input)
	if err != nil {
		return PreparedSuggestion{}, err
	}
	var suggestion *Suggestion
	for index := range current.Suggestions {
		if current.Suggestions[index].ID == input.SuggestionID && current.Suggestions[index].Status == SuggestionProposed {
			suggestion = &current.Suggestions[index]
			break
		}
	}
	if suggestion == nil {
		return PreparedSuggestion{}, ErrNotFound
	}
	var candidate *Candidate
	for index := range current.Candidates {
		if current.Candidates[index].ID == suggestion.CandidateID {
			candidate = &current.Candidates[index]
			break
		}
	}
	if candidate == nil {
		return PreparedSuggestion{}, ErrNotFound
	}
	prepared := PreparedSuggestion{AssessmentVersion: current.Version, Suggestion: *suggestion, Candidate: cloneCandidate(*candidate)}
	for _, match := range candidate.Matches {
		if match.RequirementID == suggestion.RequirementID || (suggestion.RequirementID == "" && match.ProgramID == suggestion.ProgramID) {
			matched := match
			prepared.Match = &matched
			prepared.ProgramVersion = match.ProgramVersion
			break
		}
	}
	if prepared.ProgramVersion == 0 && suggestion.ProgramID != "" {
		for _, program := range programs {
			if program.ProgramID == suggestion.ProgramID {
				prepared.ProgramVersion = program.Version
				break
			}
		}
	}
	return prepared, nil
}

func (s *Service) ApplySuggestion(ctx context.Context, input ApplySuggestionInput) (ApplySuggestionResult, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.DocumentID = strings.TrimSpace(input.DocumentID)
	input.SuggestionID = strings.TrimSpace(input.SuggestionID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.TenantID == "" || input.DocumentID == "" || input.SuggestionID == "" || input.ActorID == "" || input.ExpectedVersion < 1 {
		return ApplySuggestionResult{}, ErrInvalidReview
	}
	prepared, err := s.PrepareSuggestion(ctx, input)
	if err != nil {
		return ApplySuggestionResult{}, err
	}
	commander, ok := s.continuity.(continuityCommander)
	if !ok {
		return ApplySuggestionResult{}, fmt.Errorf("governed suggestion application is unavailable")
	}
	now := s.now().UTC()
	objectType, objectID := "", ""
	switch prepared.Suggestion.Type {
	case SuggestionLinkRequirement:
		if prepared.Match == nil {
			return ApplySuggestionResult{}, ErrInvalidReview
		}
		assessment, reviewErr := s.Review(ctx, ReviewInput{
			TenantID: input.TenantID, DocumentID: input.DocumentID, ExpectedVersion: input.ExpectedVersion, ReviewerID: input.ActorID,
			Decisions: []DecisionInput{{CandidateID: prepared.Candidate.ID, Decision: DecisionAccept, MatchID: prepared.Match.ID}},
		})
		if reviewErr != nil {
			return ApplySuggestionResult{}, reviewErr
		}
		return ApplySuggestionResult{Assessment: assessment}, nil
	case SuggestionAddRequirement:
		if prepared.Suggestion.ProgramID == "" || prepared.ProgramVersion < 1 {
			return ApplySuggestionResult{}, ErrInvalidReview
		}
		program, commandErr := commander.AddRequirement(ctx, continuity.AddRequirementInput{
			TenantID: input.TenantID, ProgramID: prepared.Suggestion.ProgramID, ExpectedVersion: prepared.ProgramVersion,
			SourceID: input.DocumentID, Code: suggestionCode("DOC", prepared.Candidate.Fingerprint),
			Title: boundedTitle(prepared.Candidate.Statement), Statement: prepared.Candidate.Statement,
			SourceAnchor: sourceAnchor(input.DocumentID, prepared.Candidate.Anchor), Modality: continuityModality(prepared.Candidate.Modality),
			Actor: prepared.Candidate.Actor, Action: prepared.Candidate.Action, Object: prepared.Candidate.Object,
			Status: continuity.RequirementDraft, EffectiveFrom: now, ActorID: input.ActorID,
		})
		if commandErr != nil {
			return ApplySuggestionResult{}, commandErr
		}
		objectType, objectID = "REQUIREMENT", latestRequirementID(program)
	case SuggestionCreateMatter:
		matterType := continuity.MatterRegulatoryChange
		if prepared.Match != nil && (!prepared.Match.Coverage.ControlImplemented || !prepared.Match.Coverage.EvidenceSupported) {
			matterType = continuity.MatterControlGap
		}
		known, _ := json.Marshal(map[string]any{"document_id": input.DocumentID, "candidate_id": prepared.Candidate.ID, "quote": prepared.Candidate.Anchor.Quote, "page": prepared.Candidate.Anchor.Page})
		matter, commandErr := commander.CreateMatter(ctx, continuity.CreateMatterInput{
			TenantID: input.TenantID, Type: matterType, Priority: 3,
			Title: boundedTitle(prepared.Candidate.Statement), Summary: "Review and address the source-backed regulatory obligation.",
			Scope: json.RawMessage(`{"access":"INTERNAL"}`), SourceType: "DOCUMENT_IMPORT", SourceID: input.DocumentID,
			TriggerType: "DOCUMENT_COVERAGE", TriggerID: input.DocumentID, TriggerKey: prepared.Suggestion.ID,
			KnownFacts: known, MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`),
			ProgramID: prepared.Suggestion.ProgramID, RequirementID: prepared.Suggestion.RequirementID, ActorID: input.ActorID,
		})
		if commandErr != nil {
			return ApplySuggestionResult{}, commandErr
		}
		objectType, objectID = "MATTER", matter.Matter.ID
	case SuggestionCreateProgram:
		programType := prepared.Candidate.ProgramType
		if programType == "" {
			programType = "COMPLIANCE"
		}
		program, commandErr := commander.CreateProgram(ctx, continuity.CreateProgramInput{
			TenantID: input.TenantID, LegalEntityID: input.LegalEntityID,
			Code: suggestionCode("DOC", prepared.Candidate.Fingerprint), Name: boundedTitle(prepared.Candidate.Statement),
			Type: programType, OwningFunction: "Compliance", Jurisdiction: prepared.Candidate.Jurisdiction,
			Scope: json.RawMessage(`{"source":"DOCUMENT_COVERAGE"}`), EffectiveFrom: now, ActorID: input.ActorID,
		})
		if commandErr != nil {
			return ApplySuggestionResult{}, commandErr
		}
		objectType, objectID = "PROGRAM", program.Program.ID
	default:
		return ApplySuggestionResult{}, ErrInvalidReview
	}
	current, err := s.repo.Current(ctx, input.TenantID, input.DocumentID)
	if err != nil {
		return ApplySuggestionResult{}, err
	}
	if current.Version != input.ExpectedVersion {
		return ApplySuggestionResult{}, ErrVersionConflict
	}
	for index := range current.Suggestions {
		if current.Suggestions[index].ID == input.SuggestionID {
			current.Suggestions[index].Status = SuggestionApplied
			current.Suggestions[index].AppliedType = objectType
			current.Suggestions[index].AppliedID = objectID
		}
	}
	current.UpdatedAt = now
	updated, err := s.repo.Review(ctx, current, current.Version)
	if err != nil {
		return ApplySuggestionResult{}, err
	}
	return ApplySuggestionResult{Assessment: updated, ObjectType: objectType, ObjectID: objectID}, nil
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

func (s *Service) currentSuggestionContext(ctx context.Context, input ApplySuggestionInput) (Assessment, []ProgramSnapshot, error) {
	current, err := s.repo.Current(ctx, strings.TrimSpace(input.TenantID), strings.TrimSpace(input.DocumentID))
	if err != nil {
		return Assessment{}, nil, err
	}
	if current.Version != input.ExpectedVersion {
		return Assessment{}, nil, ErrVersionConflict
	}
	programs, hash, err := s.programSnapshots(ctx, input.TenantID)
	if err != nil {
		return Assessment{}, nil, err
	}
	if hash != current.ProgramSnapshotHash {
		return Assessment{}, nil, ErrStaleAssessment
	}
	return current, programs, nil
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
			TenantID: tenant, LegalEntityID: aggregate.Program.LegalEntityID,
			ProgramID: aggregate.Program.ID, Code: aggregate.Program.Code, Name: aggregate.Program.Name,
			Type: aggregate.Program.Type, Status: aggregate.Program.Status, Jurisdiction: aggregate.Program.Jurisdiction,
			Regulator: regulatorFromScope(aggregate.Program.Scope), Version: aggregate.Program.Version, Requirements: []RequirementTarget{},
		}
		for _, requirement := range aggregate.Requirements {
			if requirement.Status != continuity.RequirementApproved {
				continue
			}
			parsed := documentimport.ParseObligation(strings.TrimSpace(requirement.Statement+" "+requirement.SourceAnchor), "REQUIREMENT_CANDIDATE")
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

func paginateAssessment(value Assessment, cursor string, limit int) (Assessment, string, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	start := 0
	if strings.TrimSpace(cursor) != "" {
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(cursor))
		if err != nil {
			return Assessment{}, "", ErrInvalidCursor
		}
		parts := strings.SplitN(string(raw), "\x00", 2)
		if len(parts) != 2 {
			return Assessment{}, "", ErrInvalidCursor
		}
		ordinal, err := strconv.Atoi(parts[0])
		if err != nil || ordinal < 0 || ordinal >= len(value.Candidates) || value.Candidates[ordinal].ID != parts[1] {
			return Assessment{}, "", ErrInvalidCursor
		}
		start = ordinal + 1
	}
	end := start + limit
	if end > len(value.Candidates) {
		end = len(value.Candidates)
	}
	page := cloneAssessment(value)
	page.Candidates = append([]Candidate(nil), value.Candidates[start:end]...)
	visible := make(map[string]struct{}, len(page.Candidates))
	for _, candidate := range page.Candidates {
		visible[candidate.ID] = struct{}{}
	}
	page.Reviews = page.Reviews[:0]
	for _, review := range value.Reviews {
		if _, ok := visible[review.CandidateID]; ok {
			page.Reviews = append(page.Reviews, review)
		}
	}
	page.Suggestions = page.Suggestions[:0]
	for _, suggestion := range value.Suggestions {
		if _, ok := visible[suggestion.CandidateID]; ok {
			page.Suggestions = append(page.Suggestions, suggestion)
		}
	}
	next := ""
	if end < len(value.Candidates) && end > 0 {
		next = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(end-1) + "\x00" + value.Candidates[end-1].ID))
	}
	return page, next, nil
}

func suggestionCode(prefix, fingerprint string) string {
	fingerprint = strings.ToUpper(strings.TrimSpace(fingerprint))
	if len(fingerprint) > 8 {
		fingerprint = fingerprint[:8]
	}
	if fingerprint == "" {
		fingerprint = "REVIEW"
	}
	return strings.ToUpper(strings.TrimSpace(prefix)) + "-" + fingerprint
}

func boundedTitle(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= 120 {
		return value
	}
	return strings.TrimSpace(string(runes[:117])) + "..."
}

func sourceAnchor(documentID string, anchor documentimport.Anchor) string {
	if anchor.Page > 0 {
		return fmt.Sprintf("document:%s page:%d", documentID, anchor.Page)
	}
	return "document:" + documentID + " section:" + anchor.SectionID
}

func continuityModality(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MUST_NOT":
		return "MUST_NOT"
	case "MAY":
		return "MAY"
	case "SHOULD":
		return "SHOULD"
	case "EXPECTED":
		return "EXPECTED"
	default:
		return "MUST"
	}
}

func latestRequirementID(program continuity.ProgramAggregate) string {
	if len(program.Requirements) == 0 {
		return ""
	}
	latest := program.Requirements[0]
	for _, requirement := range program.Requirements[1:] {
		if requirement.CreatedAt.After(latest.CreatedAt) || (requirement.CreatedAt.Equal(latest.CreatedAt) && requirement.ID > latest.ID) {
			latest = requirement
		}
	}
	return latest.ID
}

var _ workflowruntime.Publisher = (*Service)(nil)
