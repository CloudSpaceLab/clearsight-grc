package frameworkcatalog

import (
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func sampleAdoptionProgram() (continuity.ProgramAggregate, time.Time) {
	at := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	old := at.Add(-48 * time.Hour)
	return continuity.ProgramAggregate{
		Program: continuity.Program{
			ID: "program-A", TenantID: "tenant-A", LegalEntityID: "entity-A",
			Status: continuity.ProgramActive, EffectiveFrom: old,
		},
		Requirements: []continuity.Requirement{
			{ID: "req-1", TenantID: "tenant-A", ProgramID: "program-A", Code: "ACCESS",
				Title: "Access review", SourceAnchor: "Section 1.1", Status: continuity.RequirementApproved, EffectiveFrom: old},
			{ID: "req-2", TenantID: "tenant-A", ProgramID: "program-A", Code: "LOGGING",
				Title: "Logging controls", SourceAnchor: "Section 2", Status: continuity.RequirementApproved, EffectiveFrom: old},
		},
		Applicability: []continuity.Applicability{
			{ID: "a-1", RequirementID: "req-1", Status: continuity.ApplicabilityPartial, EffectiveFrom: old, CreatedAt: old},
			{ID: "a-2", RequirementID: "req-2", Status: continuity.ApplicabilityNotApplicable, EffectiveFrom: old, CreatedAt: old},
		},
	}, at
}

func sampleAdoptionEdition() Edition {
	return Edition{ID: "edition-1", PublisherCode: "EXAMPLE", FrameworkCode: "SEC", EditionCode: "2026", ReleaseSHA256: "sha"}
}

func TestPreviewClauseReuseTwoClausesOneCurrentRequirement(t *testing.T) {
	program, at := sampleAdoptionProgram()
	clauses := []Clause{
		{Code: "A.1", Title: "Access control", SourceAnchor: "Section 1.1"},
		{Code: "A.2", Title: "Access responsibility", SourceAnchor: "Section 1.1"},
		{Code: "LOGGING", Title: "Operating logging", SourceAnchor: "Another source"},
		{Code: "NEW", Title: "Unmapped", SourceAnchor: "Section 9"},
	}
	got, err := PreviewClauseReuse(program, sampleAdoptionEdition(), clauses, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[0].Status != ClauseReuseReviewRequired || got[1].Status != ClauseReuseReviewRequired ||
		got[0].MatchBasis != "EXACT_SOURCE_ANCHOR" || got[2].MatchBasis != "EXACT_CODE" ||
		got[3].Status != ClauseReuseUnmapped {
		t.Fatalf("incorrect matching statuses: %+v", got)
	}
	if len(got[0].Candidates) != 1 || len(got[1].Candidates) != 1 ||
		got[0].Candidates[0].RequirementID != got[1].Candidates[0].RequirementID ||
		got[0].Candidates[0].Applicability != continuity.ApplicabilityPartial ||
		got[2].Candidates[0].Applicability != continuity.ApplicabilityNotApplicable {
		t.Fatalf("lost canonical identity/applicability: %+v", got)
	}
	if program.Requirements[0].Code != "ACCESS" {
		t.Fatal("preview changed existing Program state")
	}
}

func TestPreviewClauseReuseExcludesFutureAndRetiredRequirements(t *testing.T) {
	program, at := sampleAdoptionProgram()
	retired := at.Add(-time.Minute)
	future := at.Add(time.Hour)
	program.Requirements[0].EffectiveUntil = &retired
	program.Requirements[1].EffectiveFrom = future
	candidates, err := PreviewClauseReuse(program, sampleAdoptionEdition(), []Clause{
		{Code: "ACCESS", SourceAnchor: "Section 1.1"},
		{Code: "LOGGING", SourceAnchor: "Section 2"},
	}, at)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range candidates {
		if item.Status != ClauseReuseUnmapped || item.TotalCandidates != 0 {
			t.Fatalf("inactive obligation was reused: %+v", item)
		}
	}
}

func TestPreviewClauseReuseAmbiguityNeedsReview(t *testing.T) {
	program, at := sampleAdoptionProgram()
	program.Requirements[1].SourceAnchor = program.Requirements[0].SourceAnchor
	got, err := PreviewClauseReuse(program, sampleAdoptionEdition(), []Clause{{Code: "C.1", SourceAnchor: "Section 1.1"}}, at)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Status != ClauseReuseAmbiguous || got[0].TotalCandidates != 2 ||
		got[0].Candidates[0].RequirementID != "req-1" || got[0].Candidates[1].RequirementID != "req-2" {
		t.Fatalf("ambiguous source was resolved automatically: %+v", got)
	}
}

func TestPreviewClauseReuseDoesNotCrossProgramOrTenant(t *testing.T) {
	program, at := sampleAdoptionProgram()
	program.Requirements = append(program.Requirements,
		continuity.Requirement{ID: "other-program", TenantID: "tenant-A", ProgramID: "program-B", Code: "OTHER", Title: "Other", SourceAnchor: "Section 8", Status: continuity.RequirementApproved, EffectiveFrom: at.Add(-time.Hour)},
		continuity.Requirement{ID: "other-tenant", TenantID: "tenant-B", ProgramID: "program-A", Code: "FOREIGN", Title: "Other", SourceAnchor: "Section 9", Status: continuity.RequirementApproved, EffectiveFrom: at.Add(-time.Hour)},
	)
	got, err := PreviewClauseReuse(program, sampleAdoptionEdition(), []Clause{
		{Code: "OTHER", SourceAnchor: "Section 8"}, {Code: "FOREIGN", SourceAnchor: "Section 9"},
	}, at)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Status != ClauseReuseUnmapped || got[1].Status != ClauseReuseUnmapped {
		t.Fatalf("out of scope requirement suggested: %+v", got)
	}
}

func TestPreviewClauseReuseRejectsMissingAndUnboundedInput(t *testing.T) {
	program, at := sampleAdoptionProgram()
	tests := []struct {
		name string
		edit func(*continuity.ProgramAggregate, *Edition, *[]Clause, *time.Time)
	}{
		{"missing tenant", func(p *continuity.ProgramAggregate, _ *Edition, _ *[]Clause, _ *time.Time) { p.Program.TenantID = "" }},
		{"missing edition", func(_ *continuity.ProgramAggregate, e *Edition, _ *[]Clause, _ *time.Time) { e.ID = "" }},
		{"zero as-of", func(_ *continuity.ProgramAggregate, _ *Edition, _ *[]Clause, at *time.Time) { *at = time.Time{} }},
		{"duplicate clause", func(_ *continuity.ProgramAggregate, _ *Edition, c *[]Clause, _ *time.Time) { *c = append(*c, (*c)[0]) }},
		{"unbounded clause", func(_ *continuity.ProgramAggregate, _ *Edition, c *[]Clause, _ *time.Time) {
			*c = make([]Clause, MaxAdoptionPreviewClauses+1)
		}},
		{"outside period", func(p *continuity.ProgramAggregate, _ *Edition, _ *[]Clause, at *time.Time) {
			*at = p.Program.EffectiveFrom.Add(-time.Hour)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, e, date := program, sampleAdoptionEdition(), at
			c := []Clause{{Code: "A", SourceAnchor: "Section 1"}}
			tc.edit(&p, &e, &c, &date)
			_, err := PreviewClauseReuse(p, e, c, date)
			if !errors.Is(err, ErrInvalidAdoptionPreview) {
				t.Fatalf("invalid scope or identity accepted: %v", err)
			}
		})
	}
}
