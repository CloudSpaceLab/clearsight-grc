package autonomy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestAutomationPolicyPageBoundsLatestCodesAndReportsTruncation(t *testing.T) {
	for _, count := range []int{0, 199, 200, 201, 250} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			policies := []AutomationPolicy{{ID: "foreign", TenantID: "other-bank", Code: "000", Version: 99}}
			for i := count - 1; i >= 0; i-- {
				code := fmt.Sprintf("POLICY-%03d", i)
				policies = append(policies,
					AutomationPolicy{ID: code + "-active", TenantID: "bank", Code: code, Name: "A", Status: AutomationPolicyActive, Version: 1},
					AutomationPolicy{ID: code + "-draft", TenantID: "bank", Code: code, Name: fmt.Sprint(count - i), Status: AutomationPolicyDraft, Version: 2},
				)
			}
			repo := NewMemoryRepository(policies...)
			service := NewService(repo)
			page, err := service.ListAutomationPolicyPage(context.Background(), " bank ")
			if err != nil {
				t.Fatal(err)
			}
			wantCount := min(count, AutomationPolicyListLimit)
			if page.TenantID != "bank" || page.Scope != "TENANT" || page.RevisionBasis != "LATEST_PER_CODE" || page.Limit != 200 {
				t.Fatalf("unexpected page identity: %#v", page)
			}
			if len(page.Items) != wantCount || page.ReturnedCount != wantCount || page.Truncated != (count > 200) {
				t.Fatalf("count=%d returned=%d items=%d truncated=%t", count, page.ReturnedCount, len(page.Items), page.Truncated)
			}
			if page.Items == nil {
				t.Fatal("empty page must encode items as an array")
			}
			for i, policy := range page.Items {
				if policy.Code != fmt.Sprintf("POLICY-%03d", i) || policy.Version != 2 || policy.Status != AutomationPolicyDraft || policy.TenantID != "bank" {
					t.Fatalf("policy %d lost code order, latest revision, lifecycle or tenant: %#v", i, policy)
				}
			}
			legacy, err := service.ListAutomationPolicies(context.Background(), "bank")
			if err != nil || len(legacy) != wantCount {
				t.Fatalf("legacy list must retain 200-row boundary: count=%d err=%v", len(legacy), err)
			}
			stored, err := repo.ListAutomationPolicies(context.Background(), "bank")
			if err != nil || len(stored) != min(count, AutomationPolicyListLimit+1) {
				t.Fatalf("repository sentinel boundary: count=%d err=%v", len(stored), err)
			}
		})
	}
}

type failedAutomationPolicyRepository struct{ Repository }

func (failedAutomationPolicyRepository) ListAutomationPolicies(context.Context, string) ([]AutomationPolicy, error) {
	return []AutomationPolicy{{ID: "partial", TenantID: "bank"}}, errors.New("read interrupted")
}

func TestAutomationPolicyPageRejectsUnavailableAndMissingScope(t *testing.T) {
	if _, err := NewService(NewMemoryRepository()).ListAutomationPolicyPage(context.Background(), " "); err == nil {
		t.Fatal("empty tenant must fail")
	}
	page, err := NewService(failedAutomationPolicyRepository{}).ListAutomationPolicyPage(context.Background(), "bank")
	if err == nil || len(page.Items) != 0 || page.Scope != "" {
		t.Fatalf("failed read exposed a partial success: %#v err=%v", page, err)
	}
	page, err = NewService(NewMemoryRepository()).ListAutomationPolicyPage(context.Background(), "bank")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["items"].([]any); !ok {
		t.Fatalf("empty items are not an array: %s", encoded)
	}
	if _, ok := body["total_count"]; ok {
		t.Fatal("bounded latest revisions cannot claim a total count")
	}
}
