//go:build postgres && postgresintegration

// Package testfixture contains teardown helpers for disposable integration data only.
package testfixture

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var fixtureDelete = regexp.MustCompile(`(?is)^\s*DELETE FROM (legal_entities|monitoring_form_templates) WHERE (.+?)\s*;?\s*$`)

// Exec removes trigger-installed starter dependencies before a fixture's original
// scoped DELETE. It never disables foreign keys, changes production deletion, or
// widens the caller's predicate. Other teardown statements execute unchanged.
func Exec(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (pgconn.CommandTag, error) {
	match := fixtureDelete.FindStringSubmatch(query)
	if match == nil || strings.Contains(match[2], ";") {
		return pool.Exec(ctx, query, args...)
	}
	table, where := strings.ToLower(match[1]), strings.TrimSuffix(match[2], ";")
	tx, err := pool.Begin(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	defer tx.Rollback(ctx)
	var links, forms string
	if table == "legal_entities" {
		targets := `(SELECT tenant_id,id FROM legal_entities WHERE ` + where + `)`
		links = `DELETE FROM framework_starter_requirement_links WHERE (tenant_id,legal_entity_id) IN ` + targets
		forms = `DELETE FROM monitoring_form_templates WHERE starter_catalog_code LIKE 'FRAMEWORK_%' AND (tenant_id,legal_entity_id) IN ` + targets
	} else {
		targets := `(SELECT tenant_id,id,version FROM monitoring_form_templates WHERE ` + where + `)`
		links = `DELETE FROM framework_starter_requirement_links WHERE (tenant_id,form_template_id,form_template_version) IN ` + targets
	}
	if _, err = tx.Exec(ctx, links, args...); err != nil {
		return pgconn.CommandTag{}, err
	}
	if forms != "" {
		if _, err = tx.Exec(ctx, forms, args...); err != nil {
			return pgconn.CommandTag{}, err
		}
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	return tag, tx.Commit(ctx)
}
