package monitoring

import (
	"errors"
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
)

func validateFindingFollowUpSelection(proposal FormTemplateProposal, ids []string, confirmed bool) error {
	if proposal.Provenance.ProposalVersion != documentimport.FindingFollowUpVersion {
		return nil
	}
	fail := func(message string) error { return errors.Join(ErrFormProposalSelection, fmt.Errorf("%s", message)) }
	if !confirmed {
		return fail("Confirm the assessment group against the original register before creating the draft.")
	}
	if proposal.BaseTemplateID != "" {
		return fail("Create a separate follow-up form for this assessment instead of adding findings to an existing form.")
	}
	selected := map[string]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	group := ""
	for _, change := range proposal.FieldChanges {
		if !selected[change.ID] {
			continue
		}
		if change.GroupID == "" {
			return fail("A selected question has no assessment group.")
		}
		if group != "" && group != change.GroupID {
			return fail("Choose one assessment group per vendor follow-up form.")
		}
		group = change.GroupID
	}
	if group == "" {
		return fail("Choose an assessment group before creating the draft.")
	}
	for _, change := range proposal.FieldChanges {
		if change.GroupID == group && !selected[change.ID] {
			return fail("Include all findings and response questions for the selected assessment.")
		}
	}
	return nil
}
