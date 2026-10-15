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

type frameworkAPIEditionFixture struct{}

func (frameworkAPIEditionFixture) GetFrameworkEdition(_ context.Context, id string) (continuity.FrameworkEditionReference, error) {
	if id != "edition-1" {
		return continuity.FrameworkEditionReference{}, continuity.ErrNotFound
	}
	return continuity.FrameworkEditionReference{
		ID: "edition-1", PublisherCode: "EXAMPLE", FrameworkCode: "SECURITY",
		EditionCode: "2026", ReleaseSHA256: "approved-release-checksum",
	}, nil
}

func TestProgramFrameworkRoutesEnforceOwnerAndIndependentReview(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	created, err := service.CreateProgram(continuity.WithTrustedSystemScope(context.Background()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Code: "FRAMEWORK",
		Name: "Information Security", Type: "REGULATORY", OwningFunction: "Risk",
		OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "checker-1",
		Scope:         json.RawMessage(`{}`),
		EffectiveFrom: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	service.ConfigureFrameworkEditionLookup(frameworkAPIEditionFixture{})
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:      {Principal: authority.Principal{ID: "owner-1", DisplayName: "Owner"}},
		authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "checker-1", DisplayName: "Checker"}},
	}}
	handler := func(user string) http.Handler {
		return New(Dependencies{
			Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
			Identity:   identity.NewDevelopmentAuthenticator("bank", user, "bank-ng"),
			Continuity: service, Authority: resolver,
		})
	}
	post := func(h http.Handler, route, body string) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, route, bytes.NewBufferString(body)))
		return rec
	}
	path := "/api/v1/programs/" + created.Program.ID + "/standards"
	proposal := `{"tenant_id":"bank","expected_version":1,"edition_id":"edition-1","rationale":"Adopt the published standard."}`
	if rec := post(handler("other"), path+"/propose", proposal); rec.Code != http.StatusForbidden && rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unauthorized proposal returned %d: %s", rec.Code, rec.Body.String())
	}
	untouched, err := service.GetProgram(continuity.WithTrustedSystemScope(context.Background()), "bank", created.Program.ID)
	if err != nil || untouched.Program.Version != created.Program.Version || len(untouched.FrameworkAdoptions) != 0 {
		t.Fatalf("unauthorized proposal modified the Program: %+v / %v", untouched.FrameworkAdoptions, err)
	}
	if rec := post(handler("owner-1"), path+"/propose",
		`{"tenant_id":"another-bank","expected_version":1,"edition_id":"edition-1","rationale":"Attempt cross tenant"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("tenant spoof returned %d: %s", rec.Code, rec.Body.String())
	}
	owner := handler("owner-1")
	rec := post(owner, path+"/propose", proposal)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner proposal %d: %s", rec.Code, rec.Body.String())
	}
	var result continuity.ProgramAggregate
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.FrameworkAdoptions) != 1 || result.FrameworkAdoptions[0].Status != continuity.FrameworkAdoptionPending {
		t.Fatalf("no pending Program edition: %+v", result.FrameworkAdoptions)
	}
	id := result.FrameworkAdoptions[0].ID
	decision := `{"tenant_id":"bank","expected_version":` + strconv.FormatInt(result.Program.Version, 10) + `,"rationale":"Source and scope reviewed."}`
	route := path + "/" + id + "/approve"
	if rec := post(owner, route, decision); rec.Code == http.StatusOK {
		t.Fatalf("maker approved own Program edition: %s", rec.Body.String())
	}
	rec = post(handler("checker-1"), route, decision)
	if rec.Code != http.StatusOK {
		t.Fatalf("independent checker %d: %s", rec.Code, rec.Body.String())
	}
	result = continuity.ProgramAggregate{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.FrameworkAdoptions) != 1 ||
		result.FrameworkAdoptions[0].Status != continuity.FrameworkAdoptionActive ||
		result.FrameworkAdoptions[0].ReviewedBy != "checker-1" ||
		len(result.Requirements) != 0 || result.Program.CalculationPolicy != nil {
		t.Fatalf("approval changed unrelated GRC facts: %+v", result)
	}
}

func TestProgramFrameworkRoutesFailClosedWithoutCatalog(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	program, err := service.CreateProgram(continuity.WithTrustedSystemScope(context.Background()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "bank-ng", Code: "NO-CATALOG", Name: "Reference", Type: "REGULATORY",
		OwningFunction: "Risk", OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "checker-1",
		EffectiveFrom: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "owner-1"}},
	}}
	h := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:   identity.NewDevelopmentAuthenticator("bank", "owner-1", "bank-ng"),
		Continuity: service, Authority: resolver})
	rec := httptest.NewRecorder()
	path := "/api/v1/programs/" + program.Program.ID + "/standards/propose"
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(
		`{"expected_version":1,"edition_id":"edition-1","rationale":"Request unavailable standard"}`)))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured catalog did not fail closed: %d %s", rec.Code, rec.Body.String())
	}
}
