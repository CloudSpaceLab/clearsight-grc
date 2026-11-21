package apiquestion

import (
	"strings"
	"testing"
)

func TestTransitionManifestBindsExactVersionsAndIsCanonical(t *testing.T) {
	a := bindingManifestItem{FieldID: "a", QuestionHash: strings.Repeat("a", 64), DraftID: "d1", DraftVersion: 1, ProposalID: "p1", ProposalVersion: 1, ApprovalID: "a1", QuestionMakerID: "maker", ProposalMakerID: "proposer", ReviewerID: "reviewer", ApprovalRuleID: "rule", ApprovalPolicyVersion: "v1", BindingID: "b1", BindingVersion: 1, SourceID: "s1", ViewID: "v1", ViewVersion: 1, ConnectionID: "c1", ConnectionVersion: 1, EndpointHash: strings.Repeat("b", 64), ResultType: "BOOLEAN"}
	b := a
	b.FieldID = "b"
	b.ProposalID = "p2"
	b.ApprovalID = "a2"
	manifest := bindingManifest{TenantID: "tenant", LegalEntityID: "entity", FormID: "form", DraftRevisionID: "draft", PendingRevisionID: "pending", DraftVersion: 1, PendingVersion: 2, Items: []bindingManifestItem{b, a}}
	digest, err := bindingManifestDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.Items = []bindingManifestItem{a, b}
	reordered, err := bindingManifestDigest(manifest)
	if err != nil || reordered != digest {
		t.Fatalf("reordered digest %q %v", reordered, err)
	}
	manifest.Items[0].BindingVersion = 2
	changed, err := bindingManifestDigest(manifest)
	if err != nil || changed == digest {
		t.Fatalf("changed binding digest %q %v", changed, err)
	}
	manifest.Items = []bindingManifestItem{a, a}
	if _, err = bindingManifestDigest(manifest); err == nil {
		t.Fatal("duplicate field accepted")
	}
	manifest.Items = []bindingManifestItem{a}
	manifest.PendingVersion = 3
	if _, err = bindingManifestDigest(manifest); err == nil {
		t.Fatal("non-successor revision accepted")
	}
	manifest.PendingVersion = 2
	manifest.Items[0].EndpointHash = "not-a-hash"
	if _, err = bindingManifestDigest(manifest); err == nil {
		t.Fatal("invalid endpoint hash accepted")
	}
}

func TestTransitionManifestRequiresIndependentApprovalAndExactFormRevisions(t *testing.T) {
	item := bindingManifestItem{FieldID: "q", QuestionHash: strings.Repeat("a", 64), DraftID: "d", DraftVersion: 1, ProposalID: "p", ProposalVersion: 1, ApprovalID: "approval", BindingID: "b", BindingVersion: 1, SourceID: "s", ViewID: "v", ViewVersion: 1, ConnectionID: "c", ConnectionVersion: 1, EndpointHash: strings.Repeat("b", 64), ResultType: "BOOLEAN", QuestionMakerID: "question-maker", ProposalMakerID: "proposal-maker", ReviewerID: "reviewer", ApprovalRuleID: "rule", ApprovalPolicyVersion: "v1"}
	manifest := bindingManifest{TenantID: "tenant", LegalEntityID: "entity", FormID: "form", DraftVersion: 1, PendingVersion: 2, DraftRevisionID: "draft-revision", PendingRevisionID: "pending-revision", Items: []bindingManifestItem{item}}
	digest, err := bindingManifestDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*bindingManifest){
		func(m *bindingManifest) { m.DraftRevisionID = "" },
		func(m *bindingManifest) { m.PendingRevisionID = m.DraftRevisionID },
		func(m *bindingManifest) { m.Items[0].ReviewerID = m.Items[0].QuestionMakerID },
		func(m *bindingManifest) { m.Items[0].ReviewerID = m.Items[0].ProposalMakerID },
		func(m *bindingManifest) { m.Items[0].ApprovalRuleID = "" },
		func(m *bindingManifest) { m.Items[0].ApprovalPolicyVersion = "" },
	} {
		candidate := manifest
		candidate.Items = append([]bindingManifestItem(nil), manifest.Items...)
		mutate(&candidate)
		if _, err := bindingManifestDigest(candidate); err == nil {
			t.Fatal("incomplete or conflicted approval accepted")
		}
	}
	manifest.Items[0].ReviewerID = "another-reviewer"
	changed, err := bindingManifestDigest(manifest)
	if err != nil || changed == digest {
		t.Fatal("digest does not bind the reviewer")
	}
	manifest.Items[0].ReviewerID = "reviewer"
	for _, mutate := range []func(*bindingManifest){
		func(m *bindingManifest) { m.DraftRevisionID = "new-draft" },
		func(m *bindingManifest) { m.PendingRevisionID = "new-pending" },
		func(m *bindingManifest) { m.Items[0].ApprovalRuleID = "new-rule" },
		func(m *bindingManifest) { m.Items[0].ApprovalPolicyVersion = "v2" },
		func(m *bindingManifest) { m.Items[0].QuestionMakerID = "new-question-maker" },
		func(m *bindingManifest) { m.Items[0].ProposalMakerID = "new-proposal-maker" },
	} {
		candidate := manifest
		candidate.Items = append([]bindingManifestItem(nil), manifest.Items...)
		mutate(&candidate)
		if got, err := bindingManifestDigest(candidate); err != nil || got == digest {
			t.Fatalf("changed provenance not bound: %q %v", got, err)
		}
	}
}
