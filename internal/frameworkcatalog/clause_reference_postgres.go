//go:build postgres

package frameworkcatalog

import (
	"context"
	"errors"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/jackc/pgx/v5"
)

// GetFrameworkClause resolves the immutable clause identity and its exact
// published edition; the caller determines tenant adoption/applicability.
func (r *PostgresRepository) GetFrameworkClause(ctx context.Context, editionID, clauseID string) (continuity.FrameworkClauseReference, error) {
	if r == nil || r.pool == nil || !validUUIDText(editionID) || !validUUIDText(clauseID) {
		return continuity.FrameworkClauseReference{}, ErrInvalidPublication
	}
	var value continuity.FrameworkClauseReference
	err := r.pool.QueryRow(ctx, `
		SELECT id::text,edition_id::text
		FROM framework_catalog_clauses
		WHERE id=$1::uuid AND edition_id=$2::uuid`,
		clauseID, editionID).Scan(&value.ID, &value.EditionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return continuity.FrameworkClauseReference{}, ErrInvalidPublication
	}
	return value, err
}
