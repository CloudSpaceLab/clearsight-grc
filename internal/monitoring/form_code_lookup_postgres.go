//go:build postgres

package monitoring

import (
	"context"
	"errors"
	"fmt"
)

// The existing program-history index begins with tenant/entity/program/code.
// Limit two distinct form IDs so duplicate codes are rejected, not guessed.
func (r *PostgresRepository) LatestFormByCode(ctx context.Context, tenant, entity, program, code string) (FormTemplate, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT ON (f.id) `+formProjection+`
		FROM monitoring_form_templates f
		WHERE f.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND f.legal_entity_id=$2::uuid AND f.program_id=$3::uuid AND f.code=$4
		ORDER BY f.id,f.version DESC LIMIT 2`, tenant, entity, program, code)
	if err != nil {
		return FormTemplate{}, mapPostgresError(err)
	}
	defer rows.Close()
	var current FormTemplate
	for rows.Next() {
		form, err := scanForm(rows)
		if err != nil {
			return FormTemplate{}, err
		}
		if current.ID != "" {
			return FormTemplate{}, errors.Join(ErrInvalid, fmt.Errorf("form code identifies more than one form"))
		}
		current = form
	}
	if err := rows.Err(); err != nil {
		return FormTemplate{}, err
	}
	if current.ID == "" {
		return FormTemplate{}, ErrNotFound
	}
	return current, nil
}
