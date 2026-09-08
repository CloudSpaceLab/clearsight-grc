package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"testing"
)

func TestVendorBatchRequiresDistinctBoundedRecipients(t *testing.T) {
	input := vendorFormBatchRequest{BatchID: "batch-2026-security", Targets: []vendorFormRequestTarget{{RelationshipID: "relationship-a", Recipient: evidence.DistributionRecipientInput{Role: evidence.RecipientTo}}}}
	if !validVendorFormBatch(input) {
		t.Fatal("valid batch rejected")
	}
	input.Targets = append(input.Targets, input.Targets[0])
	if validVendorFormBatch(input) {
		t.Fatal("duplicate vendor accepted")
	}
	input.Targets = input.Targets[:1]
	input.Targets[0].Recipient.Role = evidence.RecipientCC
	if validVendorFormBatch(input) {
		t.Fatal("CC recipient cannot complete form")
	}
}
