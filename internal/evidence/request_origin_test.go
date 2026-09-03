package evidence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCreateRequestReusesExactOrigin(t *testing.T) {
	service := newOriginService()
	input := validOriginRequestInput("bank-a")
	input.Origin = &RequestOrigin{Type: OriginMonitoringCollection, ID: "11111111-1111-7111-8111-111111111111", Sequence: 1}
	first, err := service.CreateRequest(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateRequest(context.Background(), input)
	if err != nil || second.ID != first.ID {
		t.Fatalf("second = %#v, err = %v", second, err)
	}
}

func TestCreateRequestRejectsChangedImmutableOriginInput(t *testing.T) {
	service := newOriginService()
	input := validOriginRequestInput("bank-a")
	input.Origin = &RequestOrigin{Type: OriginMonitoringCollection, ID: "11111111-1111-7111-8111-111111111111", Sequence: 1}
	if _, err := service.CreateRequest(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	input.Title = "Changed collection title"
	if _, err := service.CreateRequest(context.Background(), input); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("changed replay error = %v, want version conflict", err)
	}
}

func TestCreateRequestKeepsOriginsTenantScoped(t *testing.T) {
	service := newOriginService()
	inputA := validOriginRequestInput("bank-a")
	inputA.Origin = &RequestOrigin{Type: OriginMonitoringCollection, ID: "11111111-1111-7111-8111-111111111111", Sequence: 1}
	inputB := validOriginRequestInput("bank-b")
	inputB.Origin = inputA.Origin
	first, err := service.CreateRequest(context.Background(), inputA)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateRequest(context.Background(), inputB)
	if err != nil || first.ID == second.ID || second.TenantID != "bank-b" {
		t.Fatalf("tenant-scoped requests = %#v %#v, err = %v", first, second, err)
	}
}

func TestCreateRequestValidatesPredecessorOriginAndSubject(t *testing.T) {
	service := newOriginService()
	firstInput := validOriginRequestInput("bank-a")
	firstInput.Origin = &RequestOrigin{Type: OriginMonitoringCollection, ID: "11111111-1111-7111-8111-111111111111", Sequence: 1}
	first, err := service.CreateRequest(context.Background(), firstInput)
	if err != nil {
		t.Fatal(err)
	}
	secondInput := validOriginRequestInput("bank-a")
	secondInput.Origin = &RequestOrigin{Type: OriginMonitoringCollection, ID: firstInput.Origin.ID, Sequence: 2}
	secondInput.PredecessorRequestID = first.ID
	second, err := service.CreateRequest(context.Background(), secondInput)
	if err != nil || second.PredecessorRequestID != first.ID {
		t.Fatalf("successor = %#v, err = %v", second, err)
	}

	wrongSubject := validOriginRequestInput("bank-a")
	wrongSubject.SubjectID = "program-2"
	wrongSubject.Origin = &RequestOrigin{Type: OriginMonitoringCollection, ID: firstInput.Origin.ID, Sequence: 3}
	wrongSubject.PredecessorRequestID = second.ID
	if _, err := service.CreateRequest(context.Background(), wrongSubject); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("wrong-subject predecessor error = %v", err)
	}

	missingPredecessor := validOriginRequestInput("bank-a")
	missingPredecessor.Origin = &RequestOrigin{Type: OriginMonitoringCollection, ID: "22222222-2222-7222-8222-222222222222", Sequence: 2}
	if _, err := service.CreateRequest(context.Background(), missingPredecessor); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("missing predecessor error = %v", err)
	}
}

func newOriginService() *Service {
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	service.now = func() time.Time { return time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC) }
	return service
}

func validOriginRequestInput(tenant string) CreateRequestInput {
	return CreateRequestInput{
		TenantID: tenant, SubjectType: "PROGRAM", SubjectID: "program-1", Title: "Vendor review", Purpose: "Collect a current vendor response.",
		WhyYou: "You are responsible for this vendor response.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: "respondent"}, EstimatedMinutes: 5,
		Deadline: time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC), KnownFacts: map[string]string{"reviewer": "reviewer"},
		Fields: []Field{{ID: "answer", Label: "Answer", Type: "text", Required: true}}, CreatedBy: "owner",
	}
}
