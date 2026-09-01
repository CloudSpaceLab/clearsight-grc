package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func TestFormScorePreviewRouteIsAuthenticatedWrite(t *testing.T) {
	want := "POST /api/v1/config/form-templates/{id}/score-preview"
	for _, route := range (&API{}).formPolicyRoutes() {
		if route.Method+" "+route.Path != want {
			continue
		}
		if route.Class != routeAuthenticatedWrite || route.Permission != identity.PermissionConfigRead || route.Command != nil {
			t.Fatalf("score preview route = %#v", route)
		}
		return
	}
	t.Fatalf("missing %s", want)
}

func TestFormScorePreviewUsesExactStoredRevisionWithoutPersistence(t *testing.T) {
	handler, repo := scorePreviewHandler(t)
	body := []byte(`{"form_template_version":2,"answers":{"certified":{"text":"No"},"incidents":{"text":"3"}}}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/config/form-templates/form-a/score-preview", bytes.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("preview: %d %s", response.Code, response.Body.String())
	}
	var result formcontract.AdvancedScoreResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if !result.Final || result.RawScore == nil || *result.RawScore != 100 || result.Band != formcontract.ConcernCritical {
		t.Fatalf("preview result = %#v", result)
	}
	forms, err := repo.ListReusableFormRevisions(t.Context(), "bank-a", "entity-a", 10)
	if err != nil || len(forms) != 1 || forms[0].Version != 2 {
		t.Fatalf("preview persisted or changed form state: %#v err=%v", forms, err)
	}
}

func TestFormScorePreviewRejectsMissingVersionAndOversizedAnswerSet(t *testing.T) {
	handler, _ := scorePreviewHandler(t)
	for name, body := range map[string][]byte{
		"missing version":  []byte(`{"answers":{}}`),
		"too many answers": previewAnswersBody(501),
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/config/form-templates/form-a/score-preview", bytes.NewReader(body)))
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestFormScorePreviewRejectsOversizedNestedDocumentValues(t *testing.T) {
	handler, _ := scorePreviewHandler(t)
	oversized := strings.Repeat("x", 513)
	body, _ := json.Marshal(map[string]any{
		"form_template_version": 2,
		"answers":               map[string]formcontract.AnswerValue{"certified": {Document: &formcontract.DocumentAnswer{ArtifactID: oversized, DocumentType: "certificate"}}},
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/config/form-templates/form-a/score-preview", bytes.NewReader(body)))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("nested document bound = %d %s", response.Code, response.Body.String())
	}
}

func scorePreviewHandler(t *testing.T) (http.Handler, *monitoring.MemoryRepository) {
	t.Helper()
	repo := monitoring.NewMemoryRepository()
	profile := &formcontract.ScoreProfile{
		Version: "risk-v2", Mode: formcontract.ScoringRisk, Direction: formcontract.DirectionHighIsPoor, Bands: formcontract.DefaultConcernBands(),
		Contributions: []formcontract.ScoreContribution{
			{ID: "missing-cert", Label: "Certification", Predicate: formcontract.Predicate{FieldID: "certified", Operator: formcontract.PredicateEquals, Values: []string{"No"}}, Weight: 1, MatchPoints: 100, NonMatchPoints: 0, Missing: formcontract.MissingZero},
			{ID: "incidents", Label: "Incidents", Predicate: formcontract.Predicate{FieldID: "incidents", Operator: formcontract.PredicateGreaterThan, Values: []string{"2"}}, Weight: 1, MatchPoints: 100, NonMatchPoints: 0, Missing: formcontract.MissingZero},
		},
	}
	_, err := repo.CreateFormRevision(t.Context(), monitoring.FormTemplate{
		ID: "form-a", TenantID: "bank-a", LegalEntityID: "entity-a", Code: "VENDOR", Name: "Vendor review", Purpose: "Review vendor evidence.", Sensitivity: "INTERNAL",
		ScoringMode: formcontract.ScoringRisk, ScoreProfile: profile, Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic},
		Sections: []formcontract.Section{{ID: "risk", Title: "Risk"}}, Fields: []formcontract.Field{
			{ID: "certified", SectionID: "risk", Label: "Certified", Type: formcontract.TypeYesNo, Required: true},
			{ID: "incidents", SectionID: "risk", Label: "Incidents", Type: formcontract.TypeInteger, Required: true},
		},
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 2, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	})
	if err != nil {
		t.Fatal(err)
	}
	service := monitoring.NewService(repo, nil)
	return New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank-a", "maker-a", "entity-a", "GRC_ADMIN"), Monitoring: service,
	}), repo
}

func previewAnswersBody(count int) []byte {
	answers := make(map[string]formcontract.AnswerValue, count)
	for index := 0; index < count; index++ {
		answers[string(rune(index+1000))] = formcontract.TextAnswer("value")
	}
	payload, _ := json.Marshal(map[string]any{"form_template_version": 2, "answers": answers})
	return payload
}
