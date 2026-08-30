package httpapi

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestVendorWorkDocumentAvailableUsesEachDocumentRequest(t *testing.T) {
	view := thirdparty.VendorWorkReviewView{Documents: []thirdparty.AssessmentReviewDocument{
		{RequestID: "request-original", ArtifactID: "pci-current", ArtifactStatus: evidence.ArtifactAvailable},
		{RequestID: "request-replacement", ArtifactID: "iso-replacement", ArtifactStatus: evidence.ArtifactAvailable},
	}}
	if !vendorWorkDocumentAvailable(view, "request-original", "pci-current") || !vendorWorkDocumentAvailable(view, "request-replacement", "iso-replacement") {
		t.Fatal("merged review documents should remain open through their source request")
	}
	if vendorWorkDocumentAvailable(view, "request-replacement", "pci-current") {
		t.Fatal("an artifact must not be opened through a different capture request")
	}
	view.Documents[0].ArtifactStatus = evidence.ArtifactQuarantined
	if vendorWorkDocumentAvailable(view, "request-original", "pci-current") {
		t.Fatal("a quarantined artifact must not be opened")
	}
}
