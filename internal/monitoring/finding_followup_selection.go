package monitoring

// A follow-up receipt contains exactly one complete source assessment. Field
// edits happen on the ordinary draft, not by silently dropping findings here.
func validateFindingFollowUpAcceptance(proposal FormTemplateProposal, selected []string, confirmed bool) error {
	if proposal.FindingAssessmentID == "" {
		return nil
	}
	if !confirmed || proposal.BaseTemplateID != "" || len(selected) == 0 || len(selected) != len(proposal.FieldChanges) {
		return ErrFormProposalSelection
	}
	available := make(map[string]bool, len(proposal.FieldChanges))
	for _, change := range proposal.FieldChanges {
		available[change.ID] = true
	}
	for _, id := range selected {
		if !available[id] {
			return ErrFormProposalSelection
		}
		delete(available, id)
	}
	return nil
}
