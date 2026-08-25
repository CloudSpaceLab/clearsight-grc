package continuity

import "testing"

func TestVendorReviewIsACanonicalMatterType(t *testing.T) {
	if !validMatterType(MatterVendorReview) {
		t.Fatal("vendor review must be accepted as a canonical Matter type")
	}
	if label := matterTypeLabel(MatterVendorReview); label != "Vendor review" {
		t.Fatalf("label = %q, want Vendor review", label)
	}
}

func TestVendorReviewCannotCloseWithoutAnOutcomeCheck(t *testing.T) {
	assessment := assessClosure(MatterAggregate{Matter: Matter{Type: MatterVendorReview}})
	if assessment.Ready {
		t.Fatal("vendor review must remain open until an outcome check is defined")
	}
}
