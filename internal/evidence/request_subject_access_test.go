package evidence

import (
	"context"
	"errors"
	"testing"
	"time"
)

type denyingSubjectRepository struct{ *MemoryRepository }

func (r *denyingSubjectRepository) CanReadSubject(_ context.Context, tenant, principalID, subjectType, subjectID string) (bool, error) {
	return false, nil
}

func TestCreateExternalRequestValidatesRequesterSubjectAccess(t *testing.T) {
	repo := &denyingSubjectRepository{MemoryRepository: NewMemoryRepository(nil, nil)}
	service := NewService(repo, NewMemoryObjectStore())
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	_, err := service.CreateRequest(context.Background(), CreateRequestInput{
		TenantID: "bank", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-other", Title: "Provide current records",
		Purpose: "Review the vendor service.", WhyYou: "You maintain the requested records.", Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR",
		Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "contact@vendor.example"}, EstimatedMinutes: 5,
		Deadline: now.Add(time.Hour), Fields: []Field{{ID: "confirm", Label: "Confirm", Type: "yes_no", Required: true}}, CreatedBy: "requester-1",
	})
	if !errors.Is(err, ErrRecipientInvalid) {
		t.Fatalf("expected inaccessible subject rejection, got %v", err)
	}
}

func TestReservedRequestOriginRequiresOwningWorkflow(t *testing.T) {
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, NewMemoryObjectStore())
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	input := CreateRequestInput{
		TenantID: "bank", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-1", Title: "Provide current records",
		Purpose: "Review the vendor service.", WhyYou: "You maintain the requested records.", Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR",
		Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "contact@vendor.example"}, EstimatedMinutes: 5,
		Deadline: now.Add(time.Hour), Fields: []Field{{ID: "confirm", Label: "Confirm", Type: "yes_no", Required: true}},
		Origin: RequestOrigin{Type: "THIRD_PARTY_ASSESSMENT", ID: "assessment-1", Version: 1},
	}
	if _, err := service.CreateRequest(context.Background(), input); err == nil {
		t.Fatal("expected reserved origin to be rejected")
	}
	if _, err := service.CreateRequest(WithRequestOriginAuthority(context.Background(), "THIRD_PARTY_ASSESSMENT"), input); err != nil {
		t.Fatalf("owning workflow should create reserved origin: %v", err)
	}
}
