package documentcoverage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
)

type serviceContinuity struct {
	programs []continuity.ProgramAggregate
	matters  []continuity.MatterAggregate
}

func (f *serviceContinuity) ListPrograms(context.Context, string, int) ([]continuity.ProgramAggregate, error) {
	return f.programs, nil
}

func (f *serviceContinuity) ListMatters(context.Context, string, string, int) ([]continuity.MatterAggregate, error) {
	return f.matters, nil
}

func TestServiceProcessesAndReviewsCompleteCoverage(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	documents := documentimport.NewMemoryRepository()
	document, err := documents.Create(context.Background(), extractedCoverageDocument(now))
	if err != nil {
		t.Fatal(err)
	}
	programs := &serviceContinuity{programs: []continuity.ProgramAggregate{completeCoverageProgram(now)}}
	repository := NewMemoryRepository()
	service := NewService(repository, documents, programs)
	service.now = func() time.Time { return now }

	assessment, err := service.Process(context.Background(), document.TenantID, document.ID)
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Status != AssessmentReady || assessment.Version != 1 || assessment.Metrics.EstimatedVerified.Numerator != 1 {
		t.Fatalf("unexpected initial assessment: %#v", assessment)
	}
	if assessment.Metrics.Verified.Numerator != 0 || len(assessment.Candidates) != 1 || len(assessment.Candidates[0].Matches) != 1 {
		t.Fatalf("unreviewed assessment overstated coverage: %#v", assessment)
	}

	updated, err := service.Review(context.Background(), ReviewInput{
		TenantID: document.TenantID, DocumentID: document.ID, ExpectedVersion: assessment.Version, ReviewerID: "reviewer-1",
		Decisions: []DecisionInput{{CandidateID: assessment.Candidates[0].ID, Decision: DecisionAccept, MatchID: assessment.Candidates[0].Matches[0].ID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 || updated.Metrics.Verified.Numerator != 1 || updated.Candidates[0].Classification != ClassificationVerified {
		t.Fatalf("accepted complete chain not verified: %#v", updated)
	}
	if len(updated.Reviews) != 1 || updated.Reviews[0].CandidateID != assessment.Candidates[0].ID {
		t.Fatalf("review history was not retained: %#v", updated.Reviews)
	}
	if _, err := service.Review(context.Background(), ReviewInput{
		TenantID: document.TenantID, DocumentID: document.ID, ExpectedVersion: assessment.Version, ReviewerID: "reviewer-1",
		Decisions: []DecisionInput{{CandidateID: assessment.Candidates[0].ID, Decision: DecisionReject}},
	}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
}

func TestServiceUsesRequestedTenantIdentifierForProgramSnapshots(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	documents := documentimport.NewMemoryRepository()
	document, err := documents.Create(context.Background(), extractedCoverageDocument(now))
	if err != nil {
		t.Fatal(err)
	}
	program := completeCoverageProgram(now)
	program.Program.TenantID = "00000000-0000-4000-8000-000000000001"
	service := NewService(NewMemoryRepository(), documents, &serviceContinuity{programs: []continuity.ProgramAggregate{program}})
	service.now = func() time.Time { return now }

	assessment, err := service.Process(context.Background(), document.TenantID, document.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(assessment.Candidates) != 1 || len(assessment.Candidates[0].Matches) != 1 {
		t.Fatalf("tenant slug and canonical UUID must describe the same scoped snapshot: %#v", assessment.Candidates)
	}
	_, slugHash, err := service.programSnapshots(context.Background(), document.TenantID)
	if err != nil {
		t.Fatal(err)
	}
	_, canonicalHash, err := service.programSnapshots(context.Background(), program.Program.TenantID)
	if err != nil {
		t.Fatal(err)
	}
	if slugHash != canonicalHash {
		t.Fatalf("Program snapshot hash must not depend on tenant alias: slug=%s canonical=%s", slugHash, canonicalHash)
	}
}

func TestServiceMarksPriorAssessmentStale(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	documents := documentimport.NewMemoryRepository()
	document, _ := documents.Create(context.Background(), extractedCoverageDocument(now))
	reader := &serviceContinuity{programs: []continuity.ProgramAggregate{completeCoverageProgram(now)}}
	service := NewService(NewMemoryRepository(), documents, reader)
	service.now = func() time.Time { return now }
	if _, err := service.Process(context.Background(), document.TenantID, document.ID); err != nil {
		t.Fatal(err)
	}
	reader.programs[0].Program.Version++
	view, err := service.Get(context.Background(), ReadInput{TenantID: document.TenantID, DocumentID: document.ID, PrincipalID: "reviewer-1"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != ViewStale || view.Metrics.EstimatedVerified.Numerator != 1 {
		t.Fatalf("stale view must preserve prior metrics: %#v", view)
	}
}

func TestServiceFiltersMatterContextPerActorWithoutChangingMetrics(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	documents := documentimport.NewMemoryRepository()
	document, _ := documents.Create(context.Background(), extractedCoverageDocument(now))
	reader := &serviceContinuity{
		programs: []continuity.ProgramAggregate{completeCoverageProgram(now)},
		matters: []continuity.MatterAggregate{{Matter: continuity.Matter{
			ID: "matter-secret", TenantID: document.TenantID, Type: continuity.MatterRegulatoryChange,
			Status: continuity.MatterAssessment, Title: "Retain processing records remediation",
			Summary: "Address the annual processing record obligation.",
			Scope:   json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["reviewer-allowed"]}`),
		}}},
	}
	service := NewService(NewMemoryRepository(), documents, reader)
	service.now = func() time.Time { return now }
	if _, err := service.Process(context.Background(), document.TenantID, document.ID); err != nil {
		t.Fatal(err)
	}
	allowed, err := service.Get(context.Background(), ReadInput{TenantID: document.TenantID, DocumentID: document.ID, PrincipalID: "reviewer-allowed"})
	if err != nil {
		t.Fatal(err)
	}
	denied, err := service.Get(context.Background(), ReadInput{TenantID: document.TenantID, DocumentID: document.ID, PrincipalID: "reviewer-denied"})
	if err != nil {
		t.Fatal(err)
	}
	if len(allowed.Matters) != 1 || len(denied.Matters) != 0 {
		t.Fatalf("unexpected actor projections: allowed=%#v denied=%#v", allowed.Matters, denied.Matters)
	}
	if allowed.Metrics != denied.Metrics {
		t.Fatalf("Matter visibility must not change Program metrics: allowed=%#v denied=%#v", allowed.Metrics, denied.Metrics)
	}
	stored, err := service.repo.Current(context.Background(), document.TenantID, document.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(stored)
	if string(raw) == "" || containsString(string(raw), "matter-secret") {
		t.Fatalf("restricted Matter leaked into stored assessment: %s", raw)
	}
}

func TestServiceRejectsUnsupportedExtraction(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	documents := documentimport.NewMemoryRepository()
	document := extractedCoverageDocument(now)
	document.ExtractionStatus = documentimport.ExtractionUnsupported
	document.AnalysisStatus = documentimport.AnalysisUnavailable
	document.Proposals = nil
	document, _ = documents.Create(context.Background(), document)
	service := NewService(NewMemoryRepository(), documents, &serviceContinuity{})
	if _, err := service.Process(context.Background(), document.TenantID, document.ID); !errors.Is(err, ErrDocumentNotReady) {
		t.Fatalf("unsupported extraction must not invent coverage: %v", err)
	}
}

func TestServicePaginatesCandidatesWithOpaqueCursor(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	documents := documentimport.NewMemoryRepository()
	document := extractedCoverageDocument(now)
	document.Proposals = nil
	for index := 1; index <= 30; index++ {
		statement := fmt.Sprintf("The regulated bank must retain compliance record %d for five years under rule %d.", index, index)
		obligation := documentimport.ParseObligation(statement, "REQUIREMENT_CANDIDATE")
		document.Proposals = append(document.Proposals, documentimport.Proposal{
			ID: fmt.Sprintf("candidate-%02d", index), Kind: "REQUIREMENT_CANDIDATE", Statement: statement,
			Anchor: documentimport.Anchor{SectionID: fmt.Sprintf("page-%d", index), Page: index, Quote: statement}, Obligation: &obligation,
		})
	}
	document, _ = documents.Create(context.Background(), document)
	service := NewService(NewMemoryRepository(), documents, &serviceContinuity{})
	service.now = func() time.Time { return now }
	if _, err := service.Process(context.Background(), document.TenantID, document.ID); err != nil {
		t.Fatal(err)
	}
	first, err := service.Get(context.Background(), ReadInput{TenantID: document.TenantID, DocumentID: document.ID, PrincipalID: "reviewer-1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Candidates) != 10 || first.NextCursor == "" || first.Metrics.RequirementMapped.Denominator != 30 {
		t.Fatalf("unexpected first page: candidates=%d cursor=%q metrics=%#v", len(first.Candidates), first.NextCursor, first.Metrics)
	}
	second, err := service.Get(context.Background(), ReadInput{TenantID: document.TenantID, DocumentID: document.ID, PrincipalID: "reviewer-1", Limit: 10, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Candidates) != 10 || second.Candidates[0].ID == first.Candidates[0].ID {
		t.Fatalf("unexpected second page: %#v", second.Candidates)
	}
	if _, err := service.Get(context.Background(), ReadInput{TenantID: document.TenantID, DocumentID: document.ID, Cursor: "not-a-valid-cursor"}); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected invalid cursor, got %v", err)
	}
}

func TestServiceAppliesCreateProgramSuggestionAsDraft(t *testing.T) {
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	documents := documentimport.NewMemoryRepository()
	document, _ := documents.Create(context.Background(), extractedCoverageDocument(now))
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	service := NewService(NewMemoryRepository(), documents, continuityService)
	service.now = func() time.Time { return now }
	assessment, err := service.Process(context.Background(), document.TenantID, document.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(assessment.Suggestions) != 1 || assessment.Suggestions[0].Type != SuggestionCreateProgram {
		t.Fatalf("expected a create Program suggestion, got %#v", assessment.Suggestions)
	}
	result, err := service.ApplySuggestion(context.Background(), ApplySuggestionInput{
		TenantID: document.TenantID, LegalEntityID: document.LegalEntityID, DocumentID: document.ID,
		SuggestionID: assessment.Suggestions[0].ID, ExpectedVersion: assessment.Version, ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ObjectType != "PROGRAM" || result.ObjectID == "" || result.Assessment.Version != 2 || result.Assessment.Suggestions[0].Status != SuggestionApplied {
		t.Fatalf("unexpected applied suggestion result: %#v", result)
	}
	program, err := continuityService.GetProgram(context.Background(), document.TenantID, result.ObjectID)
	if err != nil {
		t.Fatal(err)
	}
	if program.Program.Status != continuity.ProgramDraft || program.Program.LegalEntityID != document.LegalEntityID {
		t.Fatalf("suggestion must create a scoped draft Program: %#v", program.Program)
	}
}

func extractedCoverageDocument(now time.Time) documentimport.Document {
	statement := "A data controller in Nigeria must retain processing records annually under section 41."
	obligation := documentimport.ParseObligation(statement, "REQUIREMENT_CANDIDATE")
	return documentimport.Document{
		ID: "document-1", TenantID: "bank-demo", LegalEntityID: "entity-1", FileName: "ndpc-guidance.pdf",
		SHA256:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ExtractionStatus: documentimport.ExtractionExtracted, AnalysisStatus: documentimport.AnalysisReviewRequired,
		Proposals: []documentimport.Proposal{{
			ID: "candidate-1", Kind: "REQUIREMENT_CANDIDATE", Statement: statement,
			Anchor: documentimport.Anchor{SectionID: "page-7", Quote: statement, Page: 7}, Obligation: &obligation,
		}}, CreatedAt: now, UpdatedAt: now, Version: 2,
	}
}

func completeCoverageProgram(now time.Time) continuity.ProgramAggregate {
	return continuity.ProgramAggregate{
		Program: continuity.Program{
			ID: "program-1", TenantID: "bank-demo", LegalEntityID: "entity-1", Code: "NDPA-2023",
			Name: "Nigeria data protection", Type: "PRIVACY", Status: continuity.ProgramActive,
			Jurisdiction: "Nigeria", EffectiveFrom: now.Add(-24 * time.Hour), Version: 3,
		},
		Requirements: []continuity.Requirement{{
			ID: "requirement-1", TenantID: "bank-demo", ProgramID: "program-1", Code: "NDPA-41",
			Title: "Retain processing records", Statement: "A data controller in Nigeria must retain processing records annually under section 41.",
			SourceAnchor: "section 41", Modality: "MUST", Actor: "data controller in Nigeria", Action: "retain",
			Object: "processing records annually under section 41", Status: continuity.RequirementApproved,
			EffectiveFrom: now.Add(-24 * time.Hour), Version: 2,
		}},
		Applicability: []continuity.Applicability{{
			RequirementID: "requirement-1", Status: continuity.ApplicabilityApplicable,
			EffectiveFrom: now.Add(-24 * time.Hour), CreatedAt: now.Add(-24 * time.Hour),
		}},
		ControlImplementations: []continuity.ControlImplementation{{
			ID: "control-1", Status: continuity.ImplementationImplemented, EffectiveFrom: now.Add(-24 * time.Hour),
		}},
		RequirementControlLinks: []continuity.RequirementControlLink{{RequirementID: "requirement-1", ImplementationID: "control-1"}},
		EvidenceContracts: []continuity.EvidenceContract{{
			ID: "contract-1", RequirementID: "requirement-1", ControlImplementationID: "control-1",
			Status: continuity.EvidenceContractActive, FreshnessMinutes: 1440, MinimumCoverage: .9,
		}},
		EvidenceAssessments: []continuity.EvidenceAssessment{{
			ContractID: "contract-1", Conclusion: continuity.EvidenceSupported, Coverage: 1,
			AssessedAt: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour),
		}},
	}
}

func containsString(value, target string) bool {
	for index := 0; index+len(target) <= len(value); index++ {
		if value[index:index+len(target)] == target {
			return true
		}
	}
	return false
}
