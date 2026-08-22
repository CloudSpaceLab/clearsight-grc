package httpapi

import (
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
)

func TestValidateDocumentHandoffApprovalFailsBeforeConversion(t *testing.T) {
	base := documentimport.Proposal{
		Status: documentimport.ProposalAccepted,
		Handoff: &documentimport.ProposalHandoff{
			Status:                documentimport.HandoffAwaitingAuthorization,
			IntakePrincipalID:     "intake-a",
			ReviewerPrincipalID:   "reviewer-b",
			TargetType:            documentimport.ConversionRequirement,
			TargetProgramID:       "program-1",
			TargetProgramVersion:  3,
			DraftTitle:            "Quarterly access review",
			DraftStatement:        "The bank shall review privileged access quarterly.",
		},
	}

	cases := []struct {
		name    string
		actorID string
		note    string
		mutate  func(*documentimport.Proposal)
		want    error
	}{
		{name: "blank rationale", actorID: "authorizer-c", want: documentimport.ErrInvalidHandoff},
		{name: "intake actor", actorID: "intake-a", note: "approve", want: documentimport.ErrHandoffSegregation},
		{name: "reviewer", actorID: "reviewer-b", note: "approve", want: documentimport.ErrHandoffSegregation},
		{name: "missing target", actorID: "authorizer-c", note: "approve", mutate: func(value *documentimport.Proposal) { value.Handoff.TargetProgramID = "" }, want: documentimport.ErrInvalidHandoff},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			value := base
			handoff := *base.Handoff
			value.Handoff = &handoff
			if test.mutate != nil {
				test.mutate(&value)
			}
			if err := validateDocumentHandoffApproval(&value, test.actorID, test.note); !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}

	if err := validateDocumentHandoffApproval(&base, "authorizer-c", "approved after independent review"); err != nil {
		t.Fatalf("independent valid approval was rejected: %v", err)
	}
}
