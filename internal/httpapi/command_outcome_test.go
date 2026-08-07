package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestMaterialHandlerReturnsReceiptWhenWriteCommittedBeforeResponseFailure(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	matter, err := service.CreateMatter(t.Context(), continuity.CreateMatterInput{
		TenantID: "bank-demo", Type: continuity.MatterControlGap, Priority: 3,
		Title: "Commit truth", Summary: "Verify post-commit response handling", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	api := &API{deps: Dependencies{Continuity: service}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/actions", strings.NewReader(`{}`))
	request.SetPathValue("id", matter.Matter.ID)
	response := httptest.NewRecorder()
	payload := map[string]any{"tenant_id": "bank-demo", "expected_version": float64(matter.Matter.Version)}
	policy := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner}

	api.executeMaterialHandler(response, request, policy, payload, func(w http.ResponseWriter, _ *http.Request) {
		if _, err := service.AddAction(t.Context(), continuity.AddActionInput{
			TenantID: "bank-demo", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
			Title: "Committed action", Description: "The authoritative write succeeds first.",
		}); err != nil {
			t.Fatal(err)
		}
		http.Error(w, "simulated response reconstruction failure", http.StatusInternalServerError)
	})

	if response.Code != http.StatusOK {
		t.Fatalf("expected committed receipt, got %d: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-ClearSight-Command-Outcome") != "committed-response-degraded" {
		t.Fatalf("missing committed outcome header: %#v", response.Header())
	}
	var receipt committedCommandReceipt
	if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "COMMITTED" || receipt.AggregateID != matter.Matter.ID || receipt.Version != matter.Matter.Version+1 || !receipt.ResponseDegraded {
		t.Fatalf("unexpected receipt: %#v", receipt)
	}
}

func TestMaterialHandlerPreservesFailureWhenNoWriteCommitted(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	matter, err := service.CreateMatter(t.Context(), continuity.CreateMatterInput{
		TenantID: "bank-demo", Type: continuity.MatterControlGap, Priority: 3,
		Title: "No commit", Summary: "Verify genuine failure handling", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	api := &API{deps: Dependencies{Continuity: service}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/actions", strings.NewReader(`{}`))
	request.SetPathValue("id", matter.Matter.ID)
	response := httptest.NewRecorder()
	payload := map[string]any{"tenant_id": "bank-demo", "expected_version": float64(matter.Matter.Version)}
	policy := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner}

	api.executeMaterialHandler(response, request, policy, payload, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "genuine failure", http.StatusInternalServerError)
	})

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected genuine failure, got %d: %s", response.Code, response.Body.String())
	}
}
