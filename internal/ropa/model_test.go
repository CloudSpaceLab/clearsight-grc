package ropa_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func TestProcessingActivityRoundTripsThroughJSON(t *testing.T) {
	fixed := time.Date(2026, 1, 15, 12, 30, 0, 0, time.UTC)
	end := fixed.AddDate(0, 6, 0)
	nextReview := fixed.AddDate(0, 3, 0)
	completed := fixed.AddDate(0, 1, 0)
	start := fixed.AddDate(-1, 0, 0)

	original := ropa.ProcessingActivity{
		ID:                           "activity-1",
		TenantID:                     "tenant-1",
		LegalEntityID:                "entity-1",
		Code:                         "PA-001",
		Name:                         "Customer onboarding",
		Description:                  "Collects identity data at onboarding.",
		Status:                       ropa.StatusOpen,
		Purpose:                      "Onboard customers",
		LawfulBasis:                  "Contract",
		Controller:                   "Fidelity Bank",
		Processor:                    "Internal operations",
		AutomatedDecisionMaking:      true,
		DataSubjectCategories:        "Customers",
		PersonalDataCategories:       "Name; Date of birth",
		SecurityMeasures:             "Encryption at rest",
		RetentionPeriod:              "7 years",
		StartDate:                    &start,
		EndDate:                      &end,
		NextReviewDate:               &nextReview,
		OwnerPrincipalID:             "owner-1",
		RequiredAuthorityPrincipalID: "authority-1",
		ProgramID:                    "program-1",
		Version:                      3,
		CreatedAt:                    fixed,
		UpdatedAt:                    fixed.Add(48 * time.Hour),
		DataCategories: []ropa.DataCategory{
			{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"},
		},
		Recipients: []ropa.Recipient{
			{Recipient: "Card processor", RecipientKind: "EXTERNAL", TransferBasis: "Contract"},
		},
		Systems: []ropa.System{
			{SystemName: "Onboarding portal", SystemKind: "APPLICATION"},
		},
		Reviews: []ropa.Review{
			{
				ID:                  "review-1",
				CreatedAt:           fixed,
				DueDate:             nextReview,
				CompletedAt:         &completed,
				Outcome:             "CONFIRMED",
				ReviewerPrincipalID: "reviewer-1",
			},
		},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal activity: %v", err)
	}
	var decoded ropa.ProcessingActivity
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal activity: %v", err)
	}
	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("round trip changed activity:\nwant: %#v\ngot:  %#v", original, decoded)
	}
	if decoded.EndDate == nil || !decoded.EndDate.Equal(end) {
		t.Fatalf("round trip lost optional end date: %#v", decoded.EndDate)
	}
	if !decoded.AutomatedDecisionMaking {
		t.Fatal("round trip lost automated decision making")
	}
	if decoded.Version != original.Version {
		t.Fatalf("round trip lost version: got %d, want %d", decoded.Version, original.Version)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("decode activity fields: %v", err)
	}
	if _, ok := fields["end_date"]; !ok {
		t.Fatal("round-trip JSON must retain end_date when it is set")
	}
	if _, ok := fields["automated_decision_making"]; !ok {
		t.Fatal("round-trip JSON must retain automated_decision_making")
	}
}

func TestReviewCreatedAtSurvivesJSONRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 1, 15, 12, 30, 0, 0, time.UTC)
	input := []byte(`{"id":"review-1","due_date":"2026-04-15T00:00:00Z","created_at":"2026-01-15T12:30:00Z"}`)

	var original ropa.Review
	if err := json.Unmarshal(input, &original); err != nil {
		t.Fatalf("unmarshal original review: %v", err)
	}
	if !original.CreatedAt.Equal(createdAt) {
		t.Fatalf("unmarshalled review created_at = %s, want %s", original.CreatedAt, createdAt)
	}
	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal review: %v", err)
	}
	var decoded ropa.Review
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal review round trip: %v", err)
	}
	if !decoded.CreatedAt.Equal(createdAt) {
		t.Fatalf("round-trip review created_at = %s, want %s", decoded.CreatedAt, createdAt)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("decode review fields: %v", err)
	}
	rawCreatedAt, ok := fields["created_at"]
	if !ok {
		t.Fatal("review JSON must retain created_at")
	}
	var gotCreatedAt time.Time
	if err := json.Unmarshal(rawCreatedAt, &gotCreatedAt); err != nil {
		t.Fatalf("decode review created_at: %v", err)
	}
	if !gotCreatedAt.Equal(createdAt) {
		t.Fatalf("review created_at = %s, want %s", gotCreatedAt, createdAt)
	}
}

func TestReviewWithOnlyCreatedAtSerialisesTheRequiredKey(t *testing.T) {
	createdAt := time.Date(2026, 1, 15, 12, 30, 0, 0, time.UTC)
	review := ropa.Review{CreatedAt: createdAt}

	encoded, err := json.Marshal(review)
	if err != nil {
		t.Fatalf("marshal review: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("decode review fields: %v", err)
	}
	if _, ok := fields["created_at"]; !ok {
		t.Fatal("a review with only created_at must serialise created_at")
	}
}

func TestProcessingActivityChildrenPreserveValuesInsideParentScope(t *testing.T) {
	fixed := time.Date(2026, 1, 15, 12, 30, 0, 0, time.UTC)
	completed := fixed.AddDate(0, 1, 0)
	original := ropa.ProcessingActivity{
		ID:            "activity-1",
		TenantID:      "tenant-1",
		LegalEntityID: "entity-1",
		DataCategories: []ropa.DataCategory{
			{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"},
			{Category: "Contact", Sensitivity: "INDIRECT_PERSONAL"},
		},
		Recipients: []ropa.Recipient{
			{Recipient: "Card processor", RecipientKind: "EXTERNAL", TransferBasis: "Contract"},
			{Recipient: "Fraud team", RecipientKind: "INTERNAL", TransferBasis: "Legitimate interests"},
		},
		Systems: []ropa.System{
			{SystemName: "Onboarding portal", SystemKind: "APPLICATION"},
			{SystemName: "Fraud warehouse", SystemKind: "DATABASE"},
		},
		Reviews: []ropa.Review{
			{
				ID:                  "review-1",
				CreatedAt:           fixed,
				DueDate:             fixed.AddDate(0, 3, 0),
				CompletedAt:         &completed,
				Outcome:             "CONFIRMED",
				ReviewerPrincipalID: "reviewer-1",
			},
			{
				ID:                  "review-2",
				CreatedAt:           fixed.Add(48 * time.Hour),
				DueDate:             fixed.AddDate(0, 6, 0),
				Outcome:             "REVISED",
				ReviewerPrincipalID: "reviewer-2",
			},
		},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal activity: %v", err)
	}
	var decoded ropa.ProcessingActivity
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal activity: %v", err)
	}
	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("nested child values changed during round trip:\nwant: %#v\ngot:  %#v", original, decoded)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("decode activity fields: %v", err)
	}
	for _, childKey := range []string{"data_categories", "recipients", "systems", "reviews"} {
		if _, ok := fields[childKey]; !ok {
			t.Fatalf("nested activity JSON must retain %s", childKey)
		}
	}
}

func TestStatusStringUsesBankOperatorLanguage(t *testing.T) {
	tests := []struct {
		name   string
		status ropa.Status
		want   string
	}{
		{name: "new", status: ropa.StatusNew, want: "Not started"},
		{name: "open", status: ropa.StatusOpen, want: "In progress"},
		{name: "closed", status: ropa.StatusClosed, want: "Complete"},
		{name: "unknown", status: ropa.Status("UNRECOGNISED"), want: "Unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.status.String(); got != test.want {
				t.Fatalf("Status(%q).String() = %q, want %q", test.status, got, test.want)
			}
		})
	}
}

func TestIsRetiredUsesOnlyAConfiguredEndDate(t *testing.T) {
	fixed := time.Date(2026, 1, 15, 12, 30, 0, 0, time.UTC)
	zero := time.Time{}
	past := fixed.Add(-24 * time.Hour)

	tests := []struct {
		name string
		end  *time.Time
		want bool
	}{
		{name: "nil", end: nil, want: false},
		{name: "zero", end: &zero, want: false},
		{name: "past", end: &past, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			activity := ropa.Aggregate{ProcessingActivity: ropa.ProcessingActivity{EndDate: test.end}}
			if got := activity.IsRetired(); got != test.want {
				t.Fatalf("IsRetired() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestReviewOverdueUsesStrictlyPastBoundary(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 30, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Second)
	zero := time.Time{}

	tests := []struct {
		name string
		date *time.Time
		want bool
	}{
		{name: "nil", date: nil, want: false},
		{name: "zero", date: &zero, want: false},
		{name: "future", date: &future, want: false},
		{name: "equal boundary", date: &now, want: false},
		{name: "strictly past", date: &past, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			activity := ropa.Aggregate{ProcessingActivity: ropa.ProcessingActivity{NextReviewDate: test.date}}
			if got := activity.ReviewOverdue(now); got != test.want {
				t.Fatalf("ReviewOverdue(%s) = %t, want %t", now, got, test.want)
			}
		})
	}
}

func TestRegisterSummaryOmitsScopeFieldsFromJSON(t *testing.T) {
	fixed := time.Date(2026, 1, 15, 12, 30, 0, 0, time.UTC)
	summary := ropa.RegisterSummary{
		TenantID:          "tenant-1",
		LegalEntityID:     "entity-1",
		GeneratedAt:       fixed,
		ProjectionVersion: ropa.ProjectionVersion,
		Freshness:         ropa.FreshnessCurrent,
		SourceHighWater:   fixed,
		Coverage: ropa.Coverage{
			Population: 4,
		},
		Counts: ropa.RegisterCounts{
			Total:              4,
			New:                1,
			Open:               2,
			Closed:             1,
			ReviewOverdue:      1,
			MissingLawfulBasis: 1,
			MissingOwner:       1,
			NoDataSubjects:     1,
			Retired:            0,
		},
	}

	encoded, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal summary: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("decode summary fields: %v", err)
	}
	if _, ok := fields["tenant_id"]; ok {
		t.Fatal("RegisterSummary JSON must omit tenant_id")
	}
	if _, ok := fields["legal_entity_id"]; ok {
		t.Fatal("RegisterSummary JSON must omit legal_entity_id")
	}
}

func TestAggregateStringRendersNameAndCode(t *testing.T) {
	activity := ropa.Aggregate{ProcessingActivity: ropa.ProcessingActivity{
		Name: "Customer onboarding",
		Code: "PA-001",
	}}
	if got, want := activity.String(), "Customer onboarding (PA-001)"; got != want {
		t.Fatalf("Aggregate.String() = %q, want %q", got, want)
	}
}
