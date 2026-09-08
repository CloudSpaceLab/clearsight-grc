package evidence

import (
	"context"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"testing"
	"time"
)

func TestVendorFormsRetirementRequiresExactFieldReplacement(t *testing.T) {
	for _, tc := range []struct{ name, change, currency string }{
		{"complete", "", "HISTORICAL"}, {"partial", "partial", "PARTIALLY_REPLACED"}, {"supplemental", "supplemental", "CURRENT"}, {"other form", "form", "CURRENT"}, {"other revision", "revision", "CURRENT"}, {"other subject", "subject", "CURRENT"}, {"unlinked", "unlinked", "CURRENT"}, {"pending", "pending", "CURRENT"}, {"separate partial replacements", "union", "HISTORICAL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now().UTC()
			repo := NewMemoryRepository(nil, nil)
			store := NewMemoryDistributionStore(repo, nil, nil)
			old := Request{ID: "old", TenantID: "bank", LegalEntityID: "entity", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "rel", FormTemplateID: "form", FormTemplateVersion: 1, Status: RequestSubmitted, Fields: []Field{{ID: "a"}, {ID: "b"}}, UpdatedAt: now}
			old.Origin.Type = "THIRD_PARTY_ASSESSMENT"
			old.Origin.ID = "assessment"
			old.Origin.Version = 1
			newer := old
			newer.ID = "newer"
			newer.Origin.Version = 2
			newer.UpdatedAt = now.Add(time.Minute)
			switch tc.change {
			case "partial", "union":
				newer.Fields = []Field{{ID: "a"}}
			case "supplemental":
				newer.Fields = []Field{{ID: "c"}}
			case "form":
				newer.FormTemplateID = "other"
			case "revision":
				newer.FormTemplateVersion = 2
			case "subject":
				newer.SubjectID = "other"
			case "pending":
				newer.Status = RequestInProgress
			}
			repo.requests[old.ID] = old
			repo.requests[newer.ID] = newer
			repo.submissions["old-sub"] = Submission{ID: "old-sub", TenantID: "bank", RequestID: "old", SubmittedAt: now}
			if tc.change != "pending" {
				repo.submissions["new-sub"] = Submission{ID: "new-sub", TenantID: "bank", RequestID: "newer", SubmittedAt: now.Add(time.Minute)}
			}
			if tc.change == "union" {
				third := newer
				third.ID = "third"
				third.Origin.Version = 3
				third.Fields = []Field{{ID: "b"}}
				repo.requests[third.ID] = third
				repo.submissions["third-sub"] = Submission{ID: "third-sub", TenantID: "bank", RequestID: third.ID, SubmittedAt: now.Add(2 * time.Minute)}
			}
			store.documentContexts = documentContextFunc(func(_ context.Context, _ DocumentQuery, r Request, _ Submission) (DocumentContext, error) {
				if r.ID == "newer" && tc.change == "unlinked" {
					return DocumentContext{}, ErrNotFound
				}
				return DocumentContext{AssessmentID: "assessment", RelationshipID: r.SubjectID}, nil
			})
			page, err := store.ListVendorForms(t.Context(), VendorFormsQuery{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner", RelationshipIDs: []string{"rel"}, Limit: 25})
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range page.Items {
				if row.RequestID == "old" {
					raw, _ := json.Marshal(row)
					var detail map[string]any
					_ = json.Unmarshal(raw, &detail)
					if detail["response_currency"] != tc.currency {
						t.Fatalf("currency=%v want %s", detail["response_currency"], tc.currency)
					}
					if row.Current != (tc.currency != "HISTORICAL") {
						t.Fatalf("current=%v", row.Current)
					}
					return
				}
			}
			t.Fatal("old response disappeared")
		})
	}
}
func TestPartlyReplacedConcernRemainsVisible(t *testing.T) {
	row := VendorFormRow{RelationshipID: "rel", Current: true, ResponseState: "SUBMITTED", Score: &ResponseScoreResult{State: ResponseScoreFinal, Band: formcontract.ConcernHigh}}
	raw, _ := json.Marshal(row)
	var detail map[string]any
	_ = json.Unmarshal(raw, &detail)
	detail["response_currency"] = "PARTIALLY_REPLACED"
	raw, _ = json.Marshal(detail)
	_ = json.Unmarshal(raw, &row)
	summary := summarizeVendorForms("rel", []VendorFormRow{row}, time.Now())
	raw, _ = json.Marshal(summary)
	_ = json.Unmarshal(raw, &detail)
	if summary.HighestConcern != formcontract.ConcernHigh || detail["partially_replaced_forms"] != float64(1) {
		t.Fatalf("summary=%s", raw)
	}
}
