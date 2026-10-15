//go:build postgres

package frameworkcatalog

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository is an internal trusted-catalog boundary. It is deliberately
// not wired to tenant commands or public HTTP routes: publication approval and
// licensing must be established before any source can invoke Publish.
type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Publish is atomic, deterministic and idempotent for identical edition
// content. A conflicting edition revision is rejected, never overwritten.
func (r *PostgresRepository) Publish(ctx context.Context, publication Publication, approvalReference string) (Edition, error) {
	if r == nil || r.pool == nil || !trimmedLength(approvalReference, 1, 240) {
		return Edition{}, ErrInvalidPublication
	}
	digest, err := ValidatePublication(publication)
	if err != nil {
		return Edition{}, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Edition{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO framework_catalog_editions (
			publisher_code,framework_code,edition_code,title,jurisdiction,
			source_url,source_sha256,source_published_on,effective_on,
			license_basis,license_reference,release_sha256,approval_reference)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::date,NULLIF($9,'')::date,$10,$11,$12,$13)
		ON CONFLICT (publisher_code,framework_code,edition_code) DO NOTHING
		RETURNING id::text`,
		publication.PublisherCode, publication.FrameworkCode, publication.EditionCode,
		publication.Title, publication.Jurisdiction, publication.SourceURL,
		publication.SourceSHA256, publication.SourcePublishedOn, publication.EffectiveOn,
		publication.LicenseBasis, publication.LicenseReference, digest, approvalReference,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		edition, getErr := getEditionByKey(ctx, tx, publication.PublisherCode, publication.FrameworkCode, publication.EditionCode)
		if getErr != nil {
			return Edition{}, getErr
		}
		if edition.ReleaseSHA256 != digest {
			return Edition{}, ErrEditionConflict
		}
		return edition, nil
	}
	if err != nil {
		return Edition{}, fmt.Errorf("insert framework edition: %w", err)
	}

	if len(publication.Clauses) > 0 {
		_, err = tx.CopyFrom(ctx, pgx.Identifier{"framework_catalog_clauses"},
			[]string{"edition_id", "code", "parent_code", "title", "source_anchor", "summary", "ordinal"},
			pgx.CopyFromSlice(len(publication.Clauses), func(i int) ([]any, error) {
				clause := publication.Clauses[i]
				var parent any
				if clause.ParentCode != "" {
					parent = clause.ParentCode
				}
				return []any{id, clause.Code, parent, clause.Title, clause.SourceAnchor, clause.Summary, clause.Ordinal}, nil
			}))
		if err != nil {
			return Edition{}, fmt.Errorf("insert framework clauses: %w", err)
		}
	}
	edition, err := getEditionByKey(ctx, tx, publication.PublisherCode, publication.FrameworkCode, publication.EditionCode)
	if err != nil {
		return Edition{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Edition{}, err
	}
	return edition, nil
}

type editionReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func getEditionByKey(ctx context.Context, db editionReader, publisher, code, edition string) (Edition, error) {
	return scanEdition(db.QueryRow(ctx, `
		SELECT id::text,publisher_code,framework_code,edition_code,title,
			   jurisdiction,source_url,source_sha256,source_published_on::text,
			   COALESCE(effective_on::text,''),license_basis,license_reference,
			   release_sha256,approval_reference
		FROM framework_catalog_editions
		WHERE publisher_code=$1 AND framework_code=$2 AND edition_code=$3`,
		publisher, code, edition))
}

func (r *PostgresRepository) GetEdition(ctx context.Context, publisher, code, revision string) (Edition, error) {
	if r == nil || r.pool == nil ||
		!publicationCodePattern.MatchString(publisher) ||
		!publicationCodePattern.MatchString(code) ||
		!publicationCodePattern.MatchString(revision) {
		return Edition{}, ErrInvalidPublication
	}
	return getEditionByKey(ctx, r.pool, publisher, code, revision)
}

// ListEditions returns only public reference metadata; no tenant data, raw
// source documents, internal assessments or unpublished review drafts.
func (r *PostgresRepository) ListEditions(ctx context.Context, jurisdiction, afterID string, limit int) ([]Edition, error) {
	if r == nil || r.pool == nil || !publicationCodePattern.MatchString(jurisdiction) ||
		(afterID != "" && !validUUIDText(afterID)) {
		return nil, ErrInvalidPublication
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text,publisher_code,framework_code,edition_code,title,
			   jurisdiction,source_url,source_sha256,source_published_on::text,
			   COALESCE(effective_on::text,''),license_basis,license_reference,
			   release_sha256,approval_reference
		FROM framework_catalog_editions
		WHERE jurisdiction=$1 AND ($2::text='' OR id > NULLIF($2::text,'')::uuid)
		ORDER BY id LIMIT $3`, jurisdiction, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Edition, 0, limit)
	for rows.Next() {
		value, scanErr := scanEdition(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *PostgresRepository) ListClauses(ctx context.Context, editionID, afterCode string, limit int) ([]Clause, error) {
	if r == nil || r.pool == nil || !validUUIDText(editionID) ||
		(afterCode != "" && !trimmedLength(afterCode, 1, 128)) {
		return nil, ErrInvalidPublication
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := r.pool.Query(ctx, `
		SELECT code,COALESCE(parent_code,''),title,source_anchor,summary,ordinal
		FROM framework_catalog_clauses
		WHERE edition_id=$1::uuid AND code>$2 ORDER BY code LIMIT $3`,
		editionID, afterCode, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Clause, 0, limit)
	for rows.Next() {
		var value Clause
		if err := rows.Scan(&value.Code, &value.ParentCode, &value.Title,
			&value.SourceAnchor, &value.Summary, &value.Ordinal); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

type editionScanner interface{ Scan(...any) error }

func scanEdition(row editionScanner) (Edition, error) {
	var value Edition
	err := row.Scan(&value.ID, &value.PublisherCode, &value.FrameworkCode,
		&value.EditionCode, &value.Title, &value.Jurisdiction, &value.SourceURL,
		&value.SourceSHA256, &value.SourcePublishedOn, &value.EffectiveOn,
		&value.LicenseBasis, &value.LicenseReference, &value.ReleaseSHA256,
		&value.ApprovalReference)
	if errors.Is(err, pgx.ErrNoRows) {
		return Edition{}, fmt.Errorf("framework edition not found: %w", err)
	}
	return value, err
}

func validUUIDText(value string) bool {
	// No SQL dynamic interpolation; require canonical identifier shape before cast.
	if len(value) != 36 || strings.ToLower(value) != value {
		return false
	}
	for i, char := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if char != '-' {
				return false
			}
		} else if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return true
}
