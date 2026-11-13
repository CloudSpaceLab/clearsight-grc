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

type requirementReviewTestRepository struct{ *continuity.MemoryRepository }

func (r *requirementReviewTestRepository) records(ctx context.Context, tenant, program string) ([]continuity.RequirementReview, error) {
	events, err := r.ProgramEvents(ctx, tenant, program, nil)
	if err != nil {
		return nil, err
	}
	records := map[string]continuity.RequirementReview{}
	for _, e := range events {
		switch e.Type {
		case continuity.EventRequirementReviewProposed:
			var value continuity.RequirementReview
			if err = json.Unmarshal(e.Payload, &value); err != nil {
				return nil, err
			}
			records[value.ID] = value
		case continuity.EventRequirementReviewApproved, continuity.EventRequirementReviewRejected:
			var value struct {
				Review continuity.RequirementReview `json:"review"`
			}
			if err = json.Unmarshal(e.Payload, &value); err != nil {
				return nil, err
			}
			records[value.Review.ID] = value.Review
		}
	}
	result := make([]continuity.RequirementReview, 0, len(records))
	for _, v := range records {
		result = append(result, v)
	}
	return result, nil
}
func (r *requirementReviewTestRepository) GetRequirementReview(ctx context.Context, t, p, id string) (continuity.RequirementReview, error) {
	v, e := r.records(ctx, t, p)
	if e != nil {
		return continuity.RequirementReview{}, e
	}
	for _, x := range v {
		if x.ID == id {
			return x, nil
		}
	}
	return continuity.RequirementReview{}, continuity.ErrNotFound
}
func (r *requirementReviewTestRepository) ListRequirementReviews(ctx context.Context, t, p, after string, limit int) ([]continuity.RequirementReview, error) {
	return r.records(ctx, t, p)
}
func TestRequirementReviewHTTPIndependentAuthorityAndScope(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(context.Background())
	repo := &requirementReviewTestRepository{continuity.NewMemoryRepository()}
	service := continuity.NewService(repo)
	program, err := service.CreateProgram(ctx, continuity.CreateProgramInput{TenantID: "bank", LegalEntityID: "bank-ng", Code: "REV-HTTP", Name: "Requirement review", Type: "REGULATORY", OwningFunction: "Risk", OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "checker-1", EffectiveFrom: time.Now().UTC(), ActorID: "owner-1"})
	if err != nil {
		t.Fatal(err)
	}
	program, err = service.AddRequirement(ctx, continuity.AddRequirementInput{TenantID: "bank", ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, Code: "DRAFT", Title: "Review access", Statement: "Review access quarterly", Status: continuity.RequirementDraft, ActorID: "owner-1"})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "owner-1"}}, authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "checker-1"}}}}
	handler := func(user, entity string) http.Handler {
		return New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", user, entity), Continuity: service, Authority: resolver})
	}
	post := func(h http.Handler, url, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(body)))
		return w
	}
	root := "/api/v1/programs/" + program.Program.ID
	// Reject direct approval before it can bypass the independent review flow.
	direct := `{"tenant_id":"bank","expected_version":` + strconv.FormatInt(program.Program.Version, 10) + `,"code":"BYPASS","title":"Bypass","statement":"Test direct approval","status":"APPROVED"}`
	if w := post(handler("owner-1", "bank-ng"), root+"/requirements", direct); w.Code != http.StatusBadRequest {
		t.Fatalf("direct approval accepted: %d %s", w.Code, w.Body.String())
	}
	// An omitted status must create a draft, never silently approve it.
	defaultStatus := `{"tenant_id":"bank","expected_version":` + strconv.FormatInt(program.Program.Version, 10) + `,"code":"NEW-DRAFT","title":"Review vendor access","statement":"Review vendor access quarterly"}`
	created := post(handler("owner-1", "bank-ng"), root+"/requirements", defaultStatus)
	if created.Code != http.StatusCreated {
		t.Fatalf("default draft creation: %d %s", created.Code, created.Body.String())
	}
	if err = json.Unmarshal(created.Body.Bytes(), &program); err != nil {
		t.Fatal(err)
	}
	if len(program.Requirements) != 2 || program.Requirements[1].Status != continuity.RequirementDraft {
		t.Fatalf("default requirement was not draft: %#v", program.Requirements)
	}
	proposal := `{"expected_version":` + strconv.FormatInt(program.Program.Version, 10) + `,"rationale":"Review this draft."}`
	route := root + "/requirements/" + program.Requirements[0].ID + "/review"
	if w := post(handler("other", "bank-ng"), route, proposal); w.Code == http.StatusCreated {
		t.Fatal("unauthorized proposer accepted")
	}
	if w := post(handler("owner-1", "foreign-entity"), route, proposal); w.Code == http.StatusCreated {
		t.Fatal("cross-entity proposal accepted")
	}
	w := post(handler("owner-1", "bank-ng"), route, proposal)
	if w.Code != http.StatusCreated {
		t.Fatalf("proposal failed %d: %s", w.Code, w.Body.String())
	}
	var proposed struct {
		Review  continuity.RequirementReview `json:"review"`
		Program continuity.Program           `json:"program"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &proposed); err != nil {
		t.Fatal(err)
	}
	get := func(h http.Handler, url string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		return w
	}
	list := root + "/requirement-reviews?tenant_id=bank"
	if w = get(handler("checker-1", "bank-ng"), list); w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(proposed.Review.ID)) {
		t.Fatalf("review list failed %d: %s", w.Code, w.Body.String())
	}
	if w = get(handler("checker-1", "foreign-entity"), list); w.Code == http.StatusOK {
		t.Fatal("cross-entity review list allowed")
	}
	if w = get(handler("checker-1", "*"), list); w.Code == http.StatusOK {
		t.Fatal("unscoped review list allowed")
	}
	if w = get(handler("checker-1", "bank-ng"), list+"&after_id=invalid"); w.Code != http.StatusBadRequest {
		t.Fatalf("invalid cursor status: %d", w.Code)
	}
	decision := `{"expected_version":` + strconv.FormatInt(proposed.Program.Version, 10) + `,"rationale":"Reviewed."}`
	approve := root + "/requirement-reviews/" + proposed.Review.ID + "/approve"
	if w = post(handler("owner-1", "bank-ng"), approve, decision); w.Code == http.StatusOK {
		t.Fatal("owner self approval accepted")
	}
	if w = post(handler("checker-1", "foreign-entity"), approve, decision); w.Code == http.StatusOK {
		t.Fatal("cross-entity approval accepted")
	}
	if w = post(handler("checker-1", "bank-ng"), approve, decision); w.Code != http.StatusOK {
		t.Fatalf("review failed %d: %s", w.Code, w.Body.String())
	}
	if w = post(handler("checker-1", "bank-ng"), approve, decision); w.Code == http.StatusOK {
		t.Fatal("repeated approval accepted")
	}
}
