package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/frameworkcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestProgramFrameworkCandidatesReuseOnlyCurrentScopedRequirements(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(context.Background())
	service := continuity.NewService(continuity.NewMemoryRepository())
	now := time.Now().UTC().Add(-time.Hour)
	program, err := service.CreateProgram(ctx, continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a",
		Code: "FRAMEWORK-READ", Name: "Standards review",
		Type: "REGULATORY", OwningFunction: "Risk",
		OwnerPrincipalID: "owner", AuthorityPrincipalID: "checker",
		EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, continuity.AddRequirementInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		ExpectedVersion: program.Program.Version, Code: "A",
		Title: "Access", Statement: "Review access", SourceAnchor: "Section 1",
		Modality: "MUST", Status: continuity.RequirementApproved, EffectiveFrom: now,
		ActorID: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	reader := testFrameworkEditionReader{items: []frameworkcatalog.Edition{{
		ID: "edition-1", PublisherCode: "EXAMPLE", FrameworkCode: "SECURITY",
		EditionCode: "2026", ReleaseSHA256: "approved-release-checksum",
		Jurisdiction: "GLOBAL", SourceURL: "https://example.org/security",
	}}}
	service.ConfigureFrameworkEditionLookup(reader)
	program, err = service.ProposeFrameworkAdoption(ctx, continuity.ProposeFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		ExpectedVersion: program.Program.Version, EditionID: "edition-1",
		ActorID: "owner", Rationale: "Applicable standard",
	})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.ApproveFrameworkAdoption(ctx, continuity.DecideFrameworkAdoptionInput{
		TenantID: "bank", ProgramID: program.Program.ID,
		AdoptionID: program.FrameworkAdoptions[0].ID, ExpectedVersion: program.Program.Version,
		ActorID: "checker", Rationale: "Reviewed scope and edition",
	})
	if err != nil {
		t.Fatal(err)
	}
	adoptionID := program.FrameworkAdoptions[0].ID
	handlerFor := func(entity string) http.Handler {
		return New(Dependencies{
			Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
			Identity:   identity.NewDevelopmentAuthenticator("bank", "owner", entity),
			Continuity: service, FrameworkCatalog: reader,
		})
	}
	url := "/api/v1/programs/" + program.Program.ID + "/standards/" + adoptionID + "/candidates"
	rec := httptest.NewRecorder()
	handlerFor("entity-a").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("candidate read %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		ProgramVersion int64                                  `json:"program_version"`
		Candidates     []frameworkcatalog.ClauseReuseProposal `json:"candidates"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProgramVersion != program.Program.Version || len(payload.Candidates) != 1 ||
		payload.Candidates[0].Status != frameworkcatalog.ClauseReuseReviewRequired ||
		len(payload.Candidates[0].Candidates) != 1 ||
		payload.Candidates[0].Candidates[0].RequirementID != program.Requirements[0].ID {
		t.Fatalf("did not reuse canonical scoped requirement: %+v", payload)
	}
	for _, target := range []struct{ entity, path string }{
		{"entity-b", url}, {"entity-a", "/api/v1/programs/" + program.Program.ID + "/standards/not-adopted/candidates"},
	} {
		rec = httptest.NewRecorder()
		handlerFor(target.entity).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target.path, nil))
		if rec.Code == http.StatusOK {
			t.Fatalf("cross-scope or non-adopted clauses exposed: %s", rec.Body.String())
		}
	}
}
