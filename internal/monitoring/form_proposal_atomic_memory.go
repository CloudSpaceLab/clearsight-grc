//go:build !postgres

package monitoring

import "context"

// formProposalAtomicAcceptor is implemented only by the PostgreSQL store. The
// interface remains visible in memory builds so FormProposalService can select
// the transactional path without build-specific service code.
type formProposalAtomicAcceptor interface {
	AcceptWithDraft(context.Context, FormProposalReviewMutation, FormTemplate) (FormTemplateProposal, error)
}
