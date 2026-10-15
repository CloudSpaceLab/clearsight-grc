package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestProgramCompliancePolicyCommandsEnforceOwnerAndIndependentChecker(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	created, err := service.CreateProgram(continuity.WithTrustedSystemScope(context.Background()), continuity.CreateProgramInput{
		TenantID:             "bank",
		LegalEntityID:        "bank-ng",
		Code:                 "ASSURANCE",
		Name:                 "Enterprise assurance",
		Type:                 "ASSURANCE",
		OwningFunction:       "Risk",
		OwnerPrincipalID:     "owner-1",
		AuthorityPrincipalID: "checker-1",
		Scope:                json.RawMessage(`{}`),
		EffectiveFrom:        time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner: {
			Principal: authority.Principal{ID: "owner-1", DisplayName: "Program owner"},
		},
		authority.ResponsibilityAuthorizer: {
			Principal: authority.Principal{ID: "checker-1", DisplayName: "Independent checker"},
		},
	}}
	handlerFor := func(id string) http.Handler {
		return New(Dependencies{
			Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
			Identity:   identity.NewDevelopmentAuthenticator("bank", id, "bank-ng"),
			Continuity: service,
			Authority:  resolver,
		})
	}
	post := func(handler http.Handler, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body)))
		return rec
	}
	path := "/api/v1/programs/" + created.Program.ID + "/calculation-policy"
	proposal := `{"tenant_id":"bank","expected_version":1,"definition":{"revision":"untrusted","minimum_coverage_bps":10000,"categories":[{"code":"REQUIREMENTS","weight_bps":10000}]},"rationale":"Set policy weights"}`
	if rec := post(handlerFor("other-1"), path+"/propose", proposal); rec.Code != http.StatusForbidden {
		t.Fatalf("unassigned proposer returned %d: %s", rec.Code, rec.Body.String())
	}
	owner := handlerFor("owner-1")
	proposed := post(owner, path+"/propose", proposal)
	if proposed.Code != http.StatusOK {
		t.Fatalf("assigned proposer returned %d: %s", proposed.Code, proposed.Body.String())
	}
	var aggregate continuity.ProgramAggregate
	if err := json.Unmarshal(proposed.Body.Bytes(), &aggregate); err != nil {
		t.Fatal(err)
	}
	if aggregate.Program.CalculationPolicy == nil || aggregate.Program.CalculationPolicy.Pending == nil ||
		aggregate.Program.CalculationPolicy.Pending.MakerID != "owner-1" ||
		aggregate.Program.CalculationPolicy.Pending.Definition.Revision != "program-v1" {
		t.Fatalf("proposal trusted a client revision or did not retain governed facts: %+v", aggregate.Program.CalculationPolicy)
	}
	decision := `{"tenant_id":"bank","expected_version":` + strconv.FormatInt(aggregate.Program.Version, 10) + `,"rationale":"Independent review"}`
	if rec := post(owner, path+"/approve", decision); rec.Code == http.StatusOK {
		t.Fatalf("maker self-approved policy: %s", rec.Body.String())
	}
	checker := handlerFor("checker-1")
	approved := post(checker, path+"/approve", decision)
	if approved.Code != http.StatusOK {
		t.Fatalf("independent checker returned %d: %s", approved.Code, approved.Body.String())
	}
	if err := json.Unmarshal(approved.Body.Bytes(), &aggregate); err != nil {
		t.Fatal(err)
	}
	if aggregate.Program.CalculationPolicy == nil || aggregate.Program.CalculationPolicy.Active == nil ||
		aggregate.Program.CalculationPolicy.Pending != nil ||
		aggregate.Program.CalculationPolicy.Active.ApproverID != "checker-1" {
		t.Fatalf("independent approval not retained: %+v", aggregate.Program.CalculationPolicy)
	}
}

