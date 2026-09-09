package httpapi

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type assessmentLabelResolver struct {
	resolution access.Resolution
	err        error
	calls      []string
}

func (s *assessmentLabelResolver) ResolveOIDC(context.Context, string, string, string, string) (access.Resolution, error) {
	return access.Resolution{}, access.ErrIdentityNotProvisioned
}
func (s *assessmentLabelResolver) ResolvePrincipal(_ context.Context, tenant, principal, entity string) (access.Resolution, error) {
	s.calls = append(s.calls, tenant+":"+entity+":"+principal)
	return s.resolution, s.err
}

func TestAssessmentReviewerLabelUsesOnlyExactHistoricalIdentity(t *testing.T) {
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "*", PrincipalID: "new-reviewer"}
	base := access.Resolution{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "old-reviewer", DisplayName: "Ada Okafor", Kind: "PERSON"}
	for _, tt := range []struct {
		name   string
		change func(*access.Resolution)
		err    error
		want   string
	}{
		{"historical reviewer", func(*access.Resolution) {}, nil, "Ada Okafor"},
		{"different principal", func(r *access.Resolution) { r.PrincipalID = "new-reviewer" }, nil, ""},
		{"different tenant", func(r *access.Resolution) { r.TenantID = "other" }, nil, ""},
		{"different entity", func(r *access.Resolution) { r.LegalEntityID = "other" }, nil, ""},
		{"unavailable", func(*access.Resolution) {}, access.ErrPrincipalUnavailable, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resolved := base
			tt.change(&resolved)
			resolver := &assessmentLabelResolver{resolution: resolved, err: tt.err}
			api := &API{deps: Dependencies{Access: resolver}}
			decision := &evidence.FieldAssessmentDecision{ReviewerID: "old-reviewer"}
			value := evidence.ResponseAssessment{Fields: []evidence.ResponseAssessmentField{{Decision: decision}, {Decision: decision}, {MayReview: true}}}
			result := api.responseAssessmentWithLabels(t.Context(), actor, "entity", value)
			if result.Fields[0].Decision.ReviewerDisplayName != tt.want || result.Fields[0].Decision.ReviewerID != "old-reviewer" || result.Fields[2].Decision != nil {
				t.Fatalf("historical label changed attribution: %+v", result.Fields)
			}
			if len(resolver.calls) != 1 || resolver.calls[0] != "tenant:entity:old-reviewer" {
				t.Fatalf("unbounded or unscoped lookup: %v", resolver.calls)
			}
			raw, _ := json.Marshal(result)
			var wire struct {
				Fields []struct {
					Decision map[string]any `json:"decision"`
				} `json:"fields"`
			}
			json.Unmarshal(raw, &wire)
			_, included := wire.Fields[0].Decision["reviewer_display_name"]
			if included != (tt.want != "") {
				t.Fatalf("unavailable name must be omitted: %s", raw)
			}
		})
	}
}

func TestVendorApplicationReceiptLabelKeepsRecordedActor(t *testing.T) {
	resolver := &assessmentLabelResolver{resolution: access.Resolution{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "original", DisplayName: "Ada Okafor"}}
	api := &API{deps: Dependencies{Access: resolver}}
	actor := thirdparty.Actor{TenantID: "tenant", LegalEntityID: "*", PrincipalID: "current"}
	view := thirdparty.AssessmentReviewView{Assessment: thirdparty.Assessment{LegalEntityID: "entity"}, ApplicationReceipt: &thirdparty.ResponseApplicationReceipt{ActorPrincipalID: "original"}}
	result := api.vendorAssessmentReviewWithLabels(t.Context(), actor, view)
	if result.ApplicationReceipt.ActorDisplayName != "Ada Okafor" || result.ApplicationReceipt.ActorPrincipalID != "original" {
		t.Fatalf("receipt attribution changed: %+v", result.ApplicationReceipt)
	}
	if len(resolver.calls) != 1 || resolver.calls[0] != "tenant:entity:original" {
		t.Fatalf("scope: %v", resolver.calls)
	}
}
