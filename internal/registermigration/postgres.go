//go:build postgres

package registermigration

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"reflect"
	"time"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{pool} }

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func readDraft(ctx context.Context, q queryer, tenant, entity, digest string, lock bool) (Draft, error) {
	actor, err := identity.Require(ctx)
	if err != nil || actor.TenantID != tenant || actor.LegalEntityID != entity {
		return Draft{}, ErrNotFound
	}
	sql := `SELECT document_id::text,source_version,version,status,selection,receipts,updated_by::text,updated_at FROM risk_register_migrations WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND legal_entity_id=$2::uuid AND source_sha256=$3`
	if lock {
		sql += " FOR UPDATE"
	}
	d := Draft{TenantID: tenant, LegalEntityID: entity, Digest: digest}
	var selection, receipts []byte
	err = q.QueryRow(ctx, sql, tenant, entity, digest).Scan(&d.DocumentID, &d.SourceVersion, &d.Version, &d.Status, &selection, &receipts, &d.UpdatedBy, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}
	if err = json.Unmarshal(selection, &d.Selection); err != nil {
		return d, err
	}
	err = json.Unmarshal(receipts, &d.Receipts)
	return d, err
}
func (r *PostgresRepository) Get(ctx context.Context, tenant, entity, digest string) (Draft, error) {
	return readDraft(ctx, r.pool, tenant, entity, digest, false)
}
func sourceCurrent(ctx context.Context, tx pgx.Tx, d Draft) error {
	var version int64
	err := tx.QueryRow(ctx, `SELECT version FROM document_imports WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND legal_entity_id=$3::uuid AND sha256=$4 AND extraction_status='EXTRACTED' AND artifact_status NOT IN ('QUARANTINED','DELETED') AND NOT content_truncated FOR SHARE`, d.DocumentID, d.TenantID, d.LegalEntityID, d.Digest).Scan(&version)
	if err != nil {
		return ErrNotFound
	}
	if version != d.SourceVersion {
		return ErrConflict
	}
	return nil
}
func writeDraft(ctx context.Context, tx pgx.Tx, d Draft, expected int64) error {
	selection, _ := json.Marshal(d.Selection)
	receipts, _ := json.Marshal(d.Receipts)
	tag, err := tx.Exec(ctx, `INSERT INTO risk_register_migrations(tenant_id,legal_entity_id,source_sha256,document_id,source_version,version,status,selection,receipts,updated_by,updated_at)
 VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3,$4::uuid,$5,$6,$7,$8,$9,$10::uuid,$11)
 ON CONFLICT(tenant_id,legal_entity_id,source_sha256) DO UPDATE SET document_id=EXCLUDED.document_id,source_version=EXCLUDED.source_version,version=EXCLUDED.version,status=EXCLUDED.status,selection=EXCLUDED.selection,receipts=EXCLUDED.receipts,updated_by=EXCLUDED.updated_by,updated_at=EXCLUDED.updated_at
 WHERE risk_register_migrations.version=$12 AND risk_register_migrations.status='DRAFT'`, d.TenantID, d.LegalEntityID, d.Digest, d.DocumentID, d.SourceVersion, d.Version, d.Status, selection, receipts, d.UpdatedBy, d.UpdatedAt, expected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	_, err = tx.Exec(ctx, `INSERT INTO risk_register_migration_revisions SELECT tenant_id,legal_entity_id,source_sha256,version,to_jsonb(m) FROM risk_register_migrations m WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND legal_entity_id=$2::uuid AND source_sha256=$3`, d.TenantID, d.LegalEntityID, d.Digest)
	if err != nil {
		return err
	}
	event := "RiskRegisterMigrationSaved"
	if d.Status == "IMPORTED" {
		event = "RiskRegisterImported"
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'DOCUMENT_IMPORT',$2::uuid,$3,jsonb_build_object('migration_version',$4::bigint,'status',$5::text,'finding_count',$6::int),$7,$7)`, d.TenantID, d.DocumentID, event, d.Version, d.Status, len(d.Receipts), d.UpdatedAt)
	return err
}
func (r *PostgresRepository) Save(ctx context.Context, d Draft, expected int64) (Draft, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Draft{}, err
	}
	defer tx.Rollback(ctx)
	old, err := readDraft(ctx, tx, d.TenantID, d.LegalEntityID, d.Digest, true)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Draft{}, err
	}
	if old.Version != expected || old.Status == "IMPORTED" {
		return Draft{}, ErrConflict
	}
	if err = sourceCurrent(ctx, tx, d); err != nil {
		return Draft{}, err
	}
	d.Version = expected + 1
	if d.Revalidate == nil {
		return Draft{}, ErrAuthority
	}
	if err = d.Revalidate(authority.WithPostgresTransaction(ctx, tx)); err != nil {
		return Draft{}, err
	}
	if err = writeDraft(ctx, tx, d, expected); err != nil {
		return Draft{}, err
	}
	if err = r.commit(ctx, tx, d); err != nil {
		return Draft{}, err
	}
	return d, nil
}
func (r *PostgresRepository) Commit(ctx context.Context, c Commit) (Draft, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Draft{}, err
	}
	defer tx.Rollback(ctx)
	d := c.Draft
	old, err := readDraft(ctx, tx, d.TenantID, d.LegalEntityID, d.Digest, true)
	if err != nil {
		return Draft{}, err
	}
	if old.Status == "IMPORTED" {
		return old, nil
	}
	if old.Version != c.ExpectedVersion {
		return Draft{}, ErrConflict
	}
	if err = sourceCurrent(ctx, tx, d); err != nil {
		return Draft{}, err
	}
	for _, g := range c.Groups {
		var version int64
		err = tx.QueryRow(ctx, `SELECT r.version FROM third_party_relationships r JOIN third_parties v ON v.id=r.vendor_id AND v.tenant_id=r.tenant_id WHERE r.id=$1::uuid AND r.tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND r.legal_entity_id=$3::uuid AND r.status<>'TERMINATED' AND v.status='ACTIVE' FOR SHARE OF r,v`, g.RelationshipID, d.TenantID, d.LegalEntityID).Scan(&version)
		if err != nil || version != g.RelationshipVersion {
			return Draft{}, ErrConflict
		}
	}
	if c.Revalidate == nil {
		return Draft{}, ErrAuthority
	}
	if err = c.Revalidate(authority.WithPostgresTransaction(ctx, tx)); err != nil {
		return Draft{}, err
	}
	for _, item := range c.Findings {
		if err = continuity.WriteImportedFindingTx(ctx, tx, item); err != nil {
			return Draft{}, err
		}
	}
	for _, link := range c.Links {
		if err = thirdparty.WriteImportedLinkTx(ctx, tx, link); err != nil {
			return Draft{}, err
		}
	}
	d.Version = c.ExpectedVersion + 1
	if err = writeDraft(ctx, tx, d, c.ExpectedVersion); err != nil {
		return Draft{}, err
	}
	if err = r.commit(ctx, tx, d); err != nil {
		return Draft{}, err
	}
	return d, nil
}
func (r *PostgresRepository) commit(ctx context.Context, tx pgx.Tx, d Draft) error {
	if err := tx.Commit(ctx); err != nil {
		probe, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		current, readErr := r.Get(probe, d.TenantID, d.LegalEntityID, d.Digest)
		if readErr == nil && current.Version == d.Version && current.Status == d.Status && current.DocumentID == d.DocumentID && current.SourceVersion == d.SourceVersion && current.UpdatedBy == d.UpdatedBy && reflect.DeepEqual(current.Selection, d.Selection) && reflect.DeepEqual(current.Receipts, d.Receipts) {
			return nil
		}
		return err
	}
	return nil
}
