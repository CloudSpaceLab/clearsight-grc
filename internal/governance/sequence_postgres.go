//go:build postgres

package governance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type legalEntityAliasResolver interface {
	ResolveLifecycleLegalEntity(context.Context, string, string, time.Time) (string, string, error)
}

func (r *PostgresRepository) ResolveLifecycleLegalEntity(ctx context.Context, tenantID, ref string, at time.Time) (string, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	} else {
		at = at.UTC()
	}
	var id, code string
	err := r.pool.QueryRow(ctx, `
		SELECT le.id::text,le.code
		FROM legal_entities le
		WHERE le.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND le.valid_from<=$3
		  AND (le.valid_until IS NULL OR $3<le.valid_until)
		LIMIT 1`, tenantID, ref, at).Scan(&id, &code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("resolve lifecycle legal entity alias: %w", err)
	}
	return id, code, nil
}

var _ legalEntityAliasResolver = (*PostgresRepository)(nil)
