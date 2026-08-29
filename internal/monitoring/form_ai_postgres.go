//go:build postgres

package monitoring

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// CreateAI persists the review lifecycle record without publishing the Task 15
// deterministic-generation event. The governed AI call remains synchronous so
// the bounded objective never has to be persisted as worker payload.
func (s *PostgresFormProposalStore) CreateAI(ctx context.Context, value FormTemplateProposal) (FormTemplateProposal, error) {
	if value.SourceKind != FormProposalSourceAI {
		return FormTemplateProposal{}, ErrInvalid
	}
	if err := validateNewFormProposal(value); err != nil {
		return FormTemplateProposal{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return FormTemplateProposal{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)

	created, err := insertFormProposal(ctx, tx, value)
	if errors.Is(err, pgx.ErrNoRows) {
		created, err = selectFormProposalBySource(ctx, tx, value)
	}
	if err != nil {
		return FormTemplateProposal{}, mapPostgresError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return FormTemplateProposal{}, mapPostgresError(err)
	}
	return created, nil
}

var _ formProposalAICreator = (*PostgresFormProposalStore)(nil)
