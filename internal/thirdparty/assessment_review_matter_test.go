package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

type assessmentMatterLinkReaderStub struct {
	values []AssessmentMatterLink
	limit  int
}

func (s *assessmentMatterLinkReaderStub) ListAssessmentMatterLinks(_ context.Context, _ Scope, _ string, limit int) ([]AssessmentMatterLink, error) {
	s.limit = limit
	return append([]AssessmentMatterLink(nil), s.values...), nil
}

type assessmentCanonicalMatterStub struct {
	values map[string]continuity.MatterAggregate
}

func (s assessmentCanonicalMatterStub) GetMatter(_ context.Context, tenantID, matterID string) (continuity.MatterAggregate, error) {
	value, ok := s.values[matterID]
	if !ok || value.Matter.TenantID != tenantID {
		return continuity.MatterAggregate{}, continuity.ErrNotFound
	}
	return value, nil
}

func TestCanonicalAssessmentReviewMatterReaderReturnsOnlyVisibleLinkedDeficiencies(t *testing.T) {
	scope := Scope{TenantID: "bank-a", LegalEntityID: "entity-a"}
	links := &assessmentMatterLinkReaderStub{values: []AssessmentMatterLink{
		{Scope: scope, AssessmentID: "assessment-1", MatterID: "review-matter", Kind: AssessmentMatterReview},
		{Scope: scope, AssessmentID: "assessment-1", MatterID: "visible-deficiency", Kind: AssessmentMatterDeficiency},
		{Scope: scope, AssessmentID: "assessment-1", MatterID: "restricted-deficiency", Kind: AssessmentMatterDeficiency},
	}}
	matters := assessmentCanonicalMatterStub{values: map[string]continuity.MatterAggregate{
		"review-matter":         {Matter: reviewMatter("bank-a", "review-matter", continuity.MatterVendorReview, `{"access":"INTERNAL"}`)},
		"visible-deficiency":    {Matter: reviewMatter("bank-a", "visible-deficiency", continuity.MatterVendorDeficiency, `{"access":"RESTRICTED","allowed_principal_ids":["reviewer-a"]}`)},
		"restricted-deficiency": {Matter: reviewMatter("bank-a", "restricted-deficiency", continuity.MatterVendorDeficiency, `{"access":"RESTRICTED","allowed_principal_ids":["other-reviewer"]}`)},
	}}
	reader := NewCanonicalAssessmentReviewMatterReader(links, matters)
	values, err := reader.ListAssessmentReviewMatters(context.Background(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "reviewer-a"}, scope, "assessment-1", assessmentReviewMaxMatters+1)
	if err != nil {
		t.Fatal(err)
	}
	if links.limit != assessmentReviewMaxMatters+1 || len(values) != 1 || values[0].MatterID != "visible-deficiency" || values[0].Type != string(continuity.MatterVendorDeficiency) {
		t.Fatalf("unexpected bounded visible deficiency projection %#v (limit %d)", values, links.limit)
	}
}

func TestCanonicalAssessmentReviewMatterReaderFailsClosedOnLinkScopeMismatch(t *testing.T) {
	scope := Scope{TenantID: "bank-a", LegalEntityID: "entity-a"}
	links := &assessmentMatterLinkReaderStub{values: []AssessmentMatterLink{{
		Scope: Scope{TenantID: "bank-a", LegalEntityID: "entity-b"}, AssessmentID: "assessment-1", MatterID: "deficiency-1", Kind: AssessmentMatterDeficiency,
	}}}
	reader := NewCanonicalAssessmentReviewMatterReader(links, assessmentCanonicalMatterStub{values: map[string]continuity.MatterAggregate{}})
	if _, err := reader.ListAssessmentReviewMatters(context.Background(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "reviewer-a"}, scope, "assessment-1", 10); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected mismatched link scope to fail closed, got %v", err)
	}
}

func reviewMatter(tenantID, matterID string, matterType continuity.MatterType, scope string) continuity.Matter {
	return continuity.Matter{ID: matterID, TenantID: tenantID, Type: matterType, Status: continuity.MatterAssessment, Title: "Vendor evidence gap", Scope: json.RawMessage(scope)}
}
