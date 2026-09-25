package reporting

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func TestExceptionPredicateMatchesRegisterClosureBlockers(t *testing.T) {
	// The register decides closure with ropa.ClosureBlockers. The report's
	// exception dataset must select exactly the activities that register would
	// refuse to close, or the same bank gets two different answers.
	cases := []struct {
		name     string
		activity ropa.ProcessingActivity
		excepted bool
	}{
		{"complete", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "CONFIRMED"}},
		}, false},
		{"missing lawful basis", ropa.ProcessingActivity{
			OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "CONFIRMED"}},
		}, true},
		{"blank lawful basis is missing", ropa.ProcessingActivity{
			LawfulBasis: "   ", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "CONFIRMED"}},
		}, true},
		{"missing owner", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "CONFIRMED"}},
		}, true},
		{"missing data subjects", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "CONFIRMED"}},
		}, true},
		{"no review at all", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
		}, true},
		{"review without completion", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{DueDate: time.Now()}},
		}, true},
		{"withdrawn review does not count", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "WITHDRAWN"}},
		}, true},
		{"blank outcome does not count", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "  "}},
		}, true},
		{"revised review counts", ropa.ProcessingActivity{
			LawfulBasis: "CONSENT", OwnerPrincipalID: "p-1", DataSubjectCategories: "CUSTOMERS",
			Reviews: []ropa.Review{{CompletedAt: ptr(time.Now()), Outcome: "REVISED"}},
		}, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			byRegister := len(ropa.ClosureBlockersForTest(testCase.activity)) > 0
			byPredicate := ExceptionPredicate(testCase.activity)
			if byRegister != testCase.excepted || byPredicate != byRegister {
				t.Fatalf("register says excepted=%v, predicate says %v, want %v",
					byRegister, byPredicate, testCase.excepted)
			}
		})
	}
}

func TestExceptionBlockersUseTheRegisterBlockerNames(t *testing.T) {
	activity := ropa.ProcessingActivity{}
	blockers := ExceptionBlockers(activity)
	if len(blockers) != 4 {
		t.Fatalf("expected four missing facts, got %#v", blockers)
	}
	want := []ExceptionBlocker{
		{Field: string(ReportFieldMissingBasis), Missing: "lawful basis"},
		{Field: string(ReportFieldMissingOwner), Missing: "named owner"},
		{Field: string(ReportFieldMissingSubjects), Missing: "data subject category"},
		{Field: string(ReportFieldReviewOverdue), Missing: "completed review"},
	}
	for index := range want {
		if blockers[index] != want[index] {
			t.Fatalf("blocker %d = %#v, want %#v", index, blockers[index], want[index])
		}
	}
}

func ptr(value time.Time) *time.Time { return &value }
