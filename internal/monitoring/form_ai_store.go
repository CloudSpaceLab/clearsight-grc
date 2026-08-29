package monitoring

import "context"

type formProposalAICreator interface {
	CreateAI(context.Context, FormTemplateProposal) (FormTemplateProposal, error)
}

func createAIProposal(ctx context.Context, store FormProposalStore, value FormTemplateProposal) (FormTemplateProposal, error) {
	if !store.QueuesGeneration() {
		return store.Create(ctx, value)
	}
	creator, ok := store.(formProposalAICreator)
	if !ok {
		return FormTemplateProposal{}, ErrFormAIUnavailable
	}
	return creator.CreateAI(ctx, value)
}
