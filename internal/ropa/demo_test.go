package ropa_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func newDemoServiceForTest(now time.Time) (*ropa.Service, *ropa.MemoryRepository, *ropa.MemorySummaryRepository) {
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)
	service.Now = func() time.Time { return now }
	return service, repository, summaries
}

func demoList(t *testing.T, service *ropa.Service, includeRetired bool) ropa.ActivityPage {
	t.Helper()
	page, err := service.ListActivities(context.Background(), ropa.ActivityScope{
		TenantID:      ropa.DemoTenant,
		LegalEntityID: ropa.DemoLegalEntity,
	}, ropa.ListActivitiesFilter{
		IncludeRetired: includeRetired,
		Limit:          50,
	})
	if err != nil {
		t.Fatalf("list demo activities (include retired=%t): %v", includeRetired, err)
	}
	return page
}

func TestInstallDemoSeedsHonestSignalsAndIsIdempotent(t *testing.T) {
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	service, repository, summaries := newDemoServiceForTest(now)

	if err := ropa.InstallDemo(context.Background(), service); err != nil {
		t.Fatalf("install demo: %v", err)
	}

	all := demoList(t, service, true)
	if len(all.Rows) != 8 {
		t.Fatalf("demo population = %d, want exactly 8", len(all.Rows))
	}

	byCode := make(map[string]ropa.ProcessingActivity, len(all.Rows))
	for _, activity := range all.Rows {
		if activity.TenantID != ropa.DemoTenant || activity.LegalEntityID != ropa.DemoLegalEntity {
			t.Fatalf("activity is outside demo scope: %#v", activity)
		}
		if strings.Contains(strings.ToLower(activity.Code), "sample") || strings.Contains(strings.ToLower(activity.Code), "test") ||
			strings.Contains(strings.ToLower(activity.Name), "sample") || strings.Contains(strings.ToLower(activity.Name), "test") {
			t.Fatalf("sample/test label must be shown by the surface, not the business record: %#v", activity)
		}
		got, err := service.GetActivity(context.Background(), activityScopeForDemo(activity), activity.ID)
		if err != nil {
			t.Fatalf("retrieve seeded activity %s: %v", activity.Code, err)
		}
		if got.Code != activity.Code || got.Version < 1 {
			t.Fatalf("retrieved activity is not the current validated record: %#v", got)
		}
		byCode[activity.Code] = got
	}

	healthy, ok := byCode["PA-CUSTOMER-ACCOUNT-OPENING"]
	if !ok {
		t.Fatalf("missing healthy account-opening activity: codes=%v", demoCodes(byCode))
	}
	if healthy.Status != ropa.StatusOpen || healthy.LawfulBasis != "Contract" || healthy.OwnerPrincipalID == "" || healthy.DataSubjectCategories == "" {
		t.Fatalf("healthy activity does not have the required complete open state: %#v", healthy)
	}
	if healthy.NextReviewDate == nil || !healthy.NextReviewDate.After(now) {
		t.Fatalf("healthy activity next review is not in the future: %#v", healthy.NextReviewDate)
	}
	if !hasConfirmedReview(healthy.Reviews) {
		t.Fatalf("healthy activity has no completed CONFIRMED review: %#v", healthy.Reviews)
	}
	if blockers, err := service.ClosureBlockers(context.Background(), activityScopeForDemo(healthy), healthy.ID); err != nil {
		t.Fatalf("healthy activity closure check: %v", err)
	} else if len(blockers) != 0 {
		t.Fatalf("healthy activity has unexpected closure blockers: %#v", blockers)
	}

	missingBasis, ok := byCode["PA-LOAN-APPLICATION"]
	if !ok {
		t.Fatalf("missing loan activity: codes=%v", demoCodes(byCode))
	}
	if missingBasis.LawfulBasis != "" || missingBasis.Status != ropa.StatusOpen {
		t.Fatalf("loan activity does not demonstrate a missing lawful basis: %#v", missingBasis)
	}
	if blockers, err := service.ClosureBlockers(context.Background(), activityScopeForDemo(missingBasis), missingBasis.ID); err != nil {
		t.Fatalf("missing-basis activity closure check: %v", err)
	} else if !reflect.DeepEqual(blockers, []string{"lawful basis"}) {
		t.Fatalf("missing-basis closure blockers = %#v, want lawful basis", blockers)
	}

	overdue, ok := byCode["PA-CUSTOMER-SERVICE-CHANNEL"]
	if !ok {
		t.Fatalf("missing overdue-review activity: codes=%v", demoCodes(byCode))
	}
	if overdue.Status != ropa.StatusOpen || overdue.NextReviewDate == nil || !overdue.NextReviewDate.Before(now) {
		t.Fatalf("customer-service activity does not demonstrate an overdue review: %#v", overdue)
	}
	if blockers, err := service.ClosureBlockers(context.Background(), activityScopeForDemo(overdue), overdue.ID); err != nil {
		t.Fatalf("overdue activity closure check: %v", err)
	} else if len(blockers) != 0 {
		t.Fatalf("overdue activity has unexpected closure blockers: %#v", blockers)
	}

	retired, ok := byCode["PA-ARCHIVED-CUSTOMER-RECORDS"]
	if !ok {
		t.Fatalf("missing retired activity: codes=%v", demoCodes(byCode))
	}
	if retired.Status != ropa.StatusClosed || retired.EndDate == nil || !retired.EndDate.Before(now) || !hasConfirmedReview(retired.Reviews) {
		t.Fatalf("archived activity is not a completed, retired record: %#v", retired)
	}

	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	if err := maintainer.Maintain(context.Background(), ropa.ActivityScope{TenantID: ropa.DemoTenant, LegalEntityID: ropa.DemoLegalEntity}); err != nil {
		t.Fatalf("maintain demo summary: %v", err)
	}
	summary, err := summaries.LatestSummary(context.Background(), ropa.DemoTenant, ropa.DemoLegalEntity)
	if err != nil {
		t.Fatalf("read demo summary: %v", err)
	}
	wantCounts := ropa.RegisterCounts{
		Total:              8,
		Open:               7,
		Closed:             1,
		ReviewOverdue:      4,
		MissingLawfulBasis: 2,
		Retired:            1,
	}
	if !reflect.DeepEqual(summary.Counts, wantCounts) {
		t.Fatalf("demo dashboard counts = %#v, want %#v", summary.Counts, wantCounts)
	}

	live := demoList(t, service, false)
	if len(live.Rows) != 7 {
		t.Fatalf("live demo population = %d, want 7; retired record must be excluded", len(live.Rows))
	}
	for _, activity := range live.Rows {
		if activity.Code == retired.Code {
			t.Fatalf("retired activity appeared in the live register: %#v", activity)
		}
	}
	history := demoList(t, service, true)
	foundRetired := false
	for _, activity := range history.Rows {
		if activity.ID == retired.ID {
			foundRetired = true
		}
	}
	if !foundRetired {
		t.Fatalf("retired activity was not present in IncludeRetired history: %#v", history.Rows)
	}

	if err := ropa.InstallDemo(context.Background(), service); err != nil {
		t.Fatalf("second install must be successful: %v", err)
	}
	second := demoList(t, service, true)
	if len(second.Rows) != 8 {
		t.Fatalf("second install changed demo population to %d, want 8", len(second.Rows))
	}
	secondCodes := make(map[string]bool, len(second.Rows))
	for _, activity := range second.Rows {
		secondCodes[activity.Code] = true
	}
	for code := range byCode {
		if !secondCodes[code] {
			t.Fatalf("second install lost seeded code %q", code)
		}
	}
}

func TestInstallDemoSeedsClientReferenceSystemsAndOpenExceptions(t *testing.T) {
	now := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	service, _, _ := newDemoServiceForTest(now)
	if err := ropa.InstallDemo(context.Background(), service); err != nil {
		t.Fatalf("install demo: %v", err)
	}

	byCode := make(map[string]ropa.ProcessingActivity)
	for _, activity := range demoList(t, service, true).Rows {
		byCode[activity.Code] = activity
	}

	wantSystems := map[string][]string{
		"PA-CUSTOMER-ACCOUNT-OPENING":     {"BVN Link Portal/Matching System", "Soft Token"},
		"PA-LOAN-APPLICATION":             {"Fincore/Coligo", "BVN Link Portal/Matching System", "Soft Token"},
		"PA-CUSTOMER-SERVICE-CHANNEL":     {"Ringo sms", "Card Management Portal/Instant card"},
		"PA-ARCHIVED-CUSTOMER-RECORDS":    {"Finacle Treasury"},
		"PA-PAYMENTS-TREASURY-OPERATIONS": {"Finacle Treasury", "Fincore/Coligo", "Card Management Portal/Instant card", "Cloudspace OEM — POS Support/PTSP", "Cloudspace OEM — Montgomery Vault Services"},
		"PA-SECURITY-MONITORING-CONTROL-REVIEW": {
			"Log management review (Qradar)", "Virus Scan and Update", "File Integrity Monitoring Review", "Patching Process", "User Access Control Review", "Internal & External Penetration Testing", "Checkmarx (Software testing/code scanning)", "FortiProxy, Analyzer, Manager, Gate (Firewall Management)", "Falcon (AD Security)", "Entrust/Entrust Middleware (Credential Security)",
		},
		"PA-AZURE-USER-ACCESS-MANAGEMENT": {"Azure portal", "Falcon (AD Security)", "Entrust/Entrust Middleware (Credential Security)"},
		"PA-AZURE-DEVICE-COMPLIANCE":      {"Azure portal", "FortiProxy, Analyzer, Manager, Gate (Firewall Management)", "Falcon (AD Security)", "Entrust/Entrust Middleware (Credential Security)"},
	}
	for code, wanted := range wantSystems {
		activity, ok := byCode[code]
		if !ok {
			t.Errorf("missing seeded activity %q", code)
			continue
		}
		got := make([]string, 0, len(activity.Systems))
		for _, system := range activity.Systems {
			got = append(got, system.SystemName)
		}
		if !reflect.DeepEqual(got, wanted) {
			t.Errorf("systems for %s = %#v, want %#v", code, got, wanted)
		}
	}

	payment, ok := byCode["PA-PAYMENTS-TREASURY-OPERATIONS"]
	if !ok {
		t.Fatal("missing payments and treasury activity")
	}
	if len(payment.Recipients) != 1 {
		t.Fatalf("payments recipients = %#v, want one Cloudspace OEM recipient", payment.Recipients)
	}
	cloudspace := payment.Recipients[0]
	if cloudspace.Recipient != "Cloudspace OEM" || cloudspace.RecipientKind != "EXTERNAL" || cloudspace.CountryCode != "" || cloudspace.IsCrossBorder || cloudspace.TransferBasis != ropa.TransferBasisNotApplicable {
		t.Fatalf("Cloudspace OEM recipient = %#v; it must be a domestic external processor and the current model leaves country code empty", cloudspace)
	}
	if !strings.Contains(payment.Description, "Nigerian") {
		t.Fatalf("Cloudspace OEM activity does not record its Nigerian operating context: %q", payment.Description)
	}
	if !strings.Contains(payment.Processor, "Cloudspace OEM") || payment.Status != ropa.StatusOpen {
		t.Fatalf("Cloudspace OEM processing activity is not an open service record: %#v", payment)
	}

	staleUsers := byCode["PA-AZURE-USER-ACCESS-MANAGEMENT"]
	if !strings.Contains(staleUsers.Description, "Risk ID 72") || !strings.Contains(staleUsers.Description, "160 stale") || !strings.Contains(staleUsers.Description, "31 January 2026") {
		t.Fatalf("stale-user exception is not grounded in source Risk ID 72: %#v", staleUsers)
	}
	if blockers, err := service.ClosureBlockers(context.Background(), activityScopeForDemo(staleUsers), staleUsers.ID); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(blockers, []string{"lawful basis", "completed review"}) {
		t.Fatalf("stale-user exception blockers = %#v", blockers)
	}

	deviceCompliance := byCode["PA-AZURE-DEVICE-COMPLIANCE"]
	if !strings.Contains(deviceCompliance.Description, "Risk ID 82") || !strings.Contains(deviceCompliance.Description, "1,315 stale") || !strings.Contains(deviceCompliance.Description, "8,693 unmanaged") {
		t.Fatalf("device exception is not grounded in source Risk ID 82: %#v", deviceCompliance)
	}
	if blockers, err := service.ClosureBlockers(context.Background(), activityScopeForDemo(deviceCompliance), deviceCompliance.ID); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(blockers, []string{"completed review"}) {
		t.Fatalf("device exception blockers = %#v", blockers)
	}

	for _, activity := range byCode {
		if !strings.Contains(strings.ToLower(activity.Description), "sample data") {
			t.Errorf("seeded activity %q is not labelled as sample data: %q", activity.Code, activity.Description)
		}
		if strings.Contains(activity.Description, "Risk ID") && !strings.Contains(strings.ToLower(activity.Description), "not legal advice") {
			t.Errorf("source exception activity %q does not state the reference-data limitation: %q", activity.Code, activity.Description)
		}
	}
	for _, forbidden := range []string{"Customer onboarding portal", "Credit decisioning platform", "Contact centre recording archive", "Legacy customer records archive"} {
		for _, activity := range byCode {
			for _, system := range activity.Systems {
				if strings.Contains(system.SystemName, forbidden) {
					t.Errorf("generic system %q remains in %q", forbidden, activity.Code)
				}
			}
		}
	}
}

func TestIsDemoScopeMatchesOnlyTheSeededPopulation(t *testing.T) {
	if !ropa.IsDemoScope(ropa.DemoTenant, ropa.DemoLegalEntity) {
		t.Fatal("the exact demo scope must be recognised")
	}
	if ropa.IsDemoScope(ropa.DemoTenant+"-other", ropa.DemoLegalEntity) || ropa.IsDemoScope(ropa.DemoTenant, ropa.DemoLegalEntity+"-other") {
		t.Fatal("a different tenant or legal entity must not be labelled as demo data")
	}
	if !ropa.IsDemoScope("  "+ropa.DemoTenant+"  ", "  "+ropa.DemoLegalEntity+"  ") {
		t.Fatal("demo scope detection should tolerate surrounding whitespace")
	}
}

func activityScopeForDemo(activity ropa.ProcessingActivity) ropa.ActivityScope {
	return ropa.ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}
}

func hasConfirmedReview(reviews []ropa.Review) bool {
	for _, review := range reviews {
		if review.CompletedAt != nil && review.Outcome == "CONFIRMED" {
			return true
		}
	}
	return false
}

func demoCodes(activities map[string]ropa.ProcessingActivity) []string {
	codes := make([]string, 0, len(activities))
	for code := range activities {
		codes = append(codes, code)
	}
	return codes
}
