package monitoring

func proposalAcceptanceSourceSHA256(proposal FormTemplateProposal) (string, bool) {
	if proposal.SourceDocumentID == "" {
		return "", false
	}
	switch proposal.SourceKind {
	case FormProposalSourceDocument:
		if validProposalSHA256(proposal.SourceSHA256) {
			return proposal.SourceSHA256, true
		}
	case FormProposalSourceAI:
		if proposal.Provenance.AI != nil && validProposalSHA256(proposal.Provenance.AI.SourceDocumentSHA256) {
			return proposal.Provenance.AI.SourceDocumentSHA256, true
		}
	}
	return "", true
}
