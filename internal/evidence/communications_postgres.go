//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type PostgresCommunicationStore struct {
	repo *PostgresRepository
}

func NewPostgresCommunicationStore(repo *PostgresRepository) *PostgresCommunicationStore {
	return &PostgresCommunicationStore{repo: repo}
}

func (store *PostgresCommunicationStore) CreateProfileRevision(ctx context.Context, input CreateCommunicationProfileInput, now time.Time) (CommunicationProfile, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return CommunicationProfile{}, ErrCommunicationInvalid
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return CommunicationProfile{}, err
	}
	defer tx.Rollback(ctx)

	tenantID, legalEntityID, err := resolveCommunicationScope(ctx, tx, input.TenantID, input.LegalEntityID)
	if err != nil {
		return CommunicationProfile{}, err
	}
	if err := lockCommunicationScope(ctx, tx, "profile", tenantID, legalEntityID, "", ""); err != nil {
		return CommunicationProfile{}, err
	}

	var version int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(max(version),0)+1 FROM form_communication_profiles WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid`, tenantID, legalEntityID).Scan(&version); err != nil {
		return CommunicationProfile{}, err
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO form_communication_profiles(
			tenant_id,legal_entity_id,version,default_locale,bank_name,support_contact,brand_asset_id,status,
			effective_from,effective_until,maker_id,created_at,updated_at
		) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,NULLIF($7,'')::uuid,'DRAFT',$8,$9,NULLIF($10,'')::uuid,$11,$11)
		RETURNING id::text,tenant_id::text,legal_entity_id::text,version,default_locale,bank_name,support_contact,
		          COALESCE(brand_asset_id::text,''),status,effective_from,effective_until,maker_id::text,
		          COALESCE(checker_id::text,''),COALESCE(rollback_origin_version,0),created_at,updated_at`,
		tenantID, legalEntityID, version, input.DefaultLocale, input.BankName, input.SupportContact, input.BrandAssetID,
		input.EffectiveFrom.UTC(), input.EffectiveUntil, input.MakerID, now.UTC())
	created, err := scanCommunicationProfile(row)
	if err != nil {
		return CommunicationProfile{}, mapCommunicationPostgresError(err)
	}
	if err := appendCommunicationGovernanceRecords(ctx, tx, created.TenantID, created.ID, "FORM_COMMUNICATION_PROFILE", "FORM_COMMUNICATION_PROFILE_REVISION_CREATED", input.MakerID, created.LegalEntityID, "", "", created.Version, created.Status, now); err != nil {
		return CommunicationProfile{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CommunicationProfile{}, err
	}
	return created, nil
}

func (store *PostgresCommunicationStore) GetProfile(ctx context.Context, tenantID, legalEntityID string, version int64) (CommunicationProfile, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil || version < 1 {
		return CommunicationProfile{}, ErrCommunicationNotFound
	}
	row := store.repo.pool.QueryRow(ctx, `
		SELECT p.id::text,p.tenant_id::text,p.legal_entity_id::text,p.version,p.default_locale,p.bank_name,p.support_contact,
		       COALESCE(p.brand_asset_id::text,''),p.status,p.effective_from,p.effective_until,p.maker_id::text,
		       COALESCE(p.checker_id::text,''),COALESCE(p.rollback_origin_version,0),p.created_at,p.updated_at
		FROM form_communication_profiles p
		JOIN tenants t ON t.id=p.tenant_id
		JOIN legal_entities le ON le.id=p.legal_entity_id AND le.tenant_id=p.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND p.version=$3`, tenantID, legalEntityID, version)
	value, err := scanCommunicationProfile(row)
	if err != nil {
		return CommunicationProfile{}, mapCommunicationPostgresError(err)
	}
	return value, nil
}

func (store *PostgresCommunicationStore) ListProfiles(ctx context.Context, tenantID, legalEntityID string) ([]CommunicationProfile, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return nil, ErrCommunicationNotFound
	}
	rows, err := store.repo.pool.Query(ctx, `
		SELECT p.id::text,p.tenant_id::text,p.legal_entity_id::text,p.version,p.default_locale,p.bank_name,p.support_contact,
		       COALESCE(p.brand_asset_id::text,''),p.status,p.effective_from,p.effective_until,p.maker_id::text,
		       COALESCE(p.checker_id::text,''),COALESCE(p.rollback_origin_version,0),p.created_at,p.updated_at
		FROM form_communication_profiles p
		JOIN tenants t ON t.id=p.tenant_id
		JOIN legal_entities le ON le.id=p.legal_entity_id AND le.tenant_id=p.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2)
		ORDER BY p.version DESC`, tenantID, legalEntityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]CommunicationProfile, 0)
	for rows.Next() {
		value, scanErr := scanCommunicationProfile(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (store *PostgresCommunicationStore) TransitionProfile(ctx context.Context, tenantID, legalEntityID string, version int64, input CommunicationTransitionInput, now time.Time) (CommunicationProfile, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil || version < 1 || input.ExpectedVersion != version {
		return CommunicationProfile{}, ErrCommunicationConflict
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return CommunicationProfile{}, err
	}
	defer tx.Rollback(ctx)
	canonicalTenantID, canonicalLegalEntityID, err := resolveCommunicationScope(ctx, tx, tenantID, legalEntityID)
	if err != nil {
		return CommunicationProfile{}, err
	}
	if err := lockCommunicationScope(ctx, tx, "profile", canonicalTenantID, canonicalLegalEntityID, "", ""); err != nil {
		return CommunicationProfile{}, err
	}
	current, err := scanCommunicationProfile(tx.QueryRow(ctx, `
		SELECT id::text,tenant_id::text,legal_entity_id::text,version,default_locale,bank_name,support_contact,
		       COALESCE(brand_asset_id::text,''),status,effective_from,effective_until,maker_id::text,
		       COALESCE(checker_id::text,''),COALESCE(rollback_origin_version,0),created_at,updated_at
		FROM form_communication_profiles
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND version=$3 FOR UPDATE`, canonicalTenantID, canonicalLegalEntityID, version))
	if err != nil {
		return CommunicationProfile{}, mapCommunicationPostgresError(err)
	}
	status, checkerID, effectiveFrom, effectiveUntil, err := applyCommunicationTransition(current.Status, current.MakerID, input, current.EffectiveFrom, current.EffectiveUntil, now.UTC())
	if err != nil {
		return CommunicationProfile{}, err
	}
	if status == CommunicationActive {
		if _, err := tx.Exec(ctx, `
			UPDATE form_communication_profiles
			SET status='RETIRED',updated_at=$5
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id<>$3::uuid AND status='ACTIVE'
			  AND (effective_until IS NULL OR effective_until>$4)
			  AND ($6::timestamptz IS NULL OR $6::timestamptz>effective_from)`, canonicalTenantID, canonicalLegalEntityID, current.ID, effectiveFrom, now.UTC(), effectiveUntil); err != nil {
			return CommunicationProfile{}, err
		}
	}
	row := tx.QueryRow(ctx, `
		UPDATE form_communication_profiles
		SET status=$4,effective_from=$5,effective_until=$6,checker_id=CASE WHEN $7='' THEN checker_id ELSE $7::uuid END,updated_at=$8
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND version=$3
		RETURNING id::text,tenant_id::text,legal_entity_id::text,version,default_locale,bank_name,support_contact,
		          COALESCE(brand_asset_id::text,''),status,effective_from,effective_until,maker_id::text,
		          COALESCE(checker_id::text,''),COALESCE(rollback_origin_version,0),created_at,updated_at`,
		canonicalTenantID, canonicalLegalEntityID, version, status, effectiveFrom, effectiveUntil, checkerID, now.UTC())
	updated, err := scanCommunicationProfile(row)
	if err != nil {
		return CommunicationProfile{}, mapCommunicationPostgresError(err)
	}
	if err := appendCommunicationGovernanceRecords(ctx, tx, updated.TenantID, updated.ID, "FORM_COMMUNICATION_PROFILE", "FORM_COMMUNICATION_PROFILE_"+string(updated.Status), input.ActorID, updated.LegalEntityID, "", "", updated.Version, updated.Status, now); err != nil {
		return CommunicationProfile{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CommunicationProfile{}, err
	}
	return updated, nil
}

func (store *PostgresCommunicationStore) CreateTemplateRevision(ctx context.Context, input CreateCommunicationTemplateInput, now time.Time) (CommunicationTemplate, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	documentJSON, err := json.Marshal(input.Document)
	if err != nil {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return CommunicationTemplate{}, err
	}
	defer tx.Rollback(ctx)
	tenantID, legalEntityID, err := resolveCommunicationScope(ctx, tx, input.TenantID, input.LegalEntityID)
	if err != nil {
		return CommunicationTemplate{}, err
	}
	if err := lockCommunicationScope(ctx, tx, "template", tenantID, legalEntityID, input.Action, input.Locale); err != nil {
		return CommunicationTemplate{}, err
	}
	var version int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(max(version),0)+1 FROM form_communication_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND action=$3 AND locale=$4`, tenantID, legalEntityID, input.Action, input.Locale).Scan(&version); err != nil {
		return CommunicationTemplate{}, err
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO form_communication_templates(
			tenant_id,legal_entity_id,action,locale,version,subject_template,document,status,effective_from,effective_until,maker_id,created_at,updated_at
		) VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7::jsonb,'DRAFT',$8,$9,NULLIF($10,'')::uuid,$11,$11)
		RETURNING id::text,tenant_id::text,legal_entity_id::text,action,locale,version,subject_template,document,status,
		          effective_from,effective_until,maker_id::text,COALESCE(checker_id::text,''),COALESCE(rollback_origin_version,0),created_at,updated_at`,
		tenantID, legalEntityID, input.Action, input.Locale, version, input.SubjectTemplate, string(documentJSON), input.EffectiveFrom.UTC(), input.EffectiveUntil, input.MakerID, now.UTC())
	created, err := scanCommunicationTemplate(row)
	if err != nil {
		return CommunicationTemplate{}, mapCommunicationPostgresError(err)
	}
	if err := appendCommunicationGovernanceRecords(ctx, tx, created.TenantID, created.ID, "FORM_COMMUNICATION_TEMPLATE", "FORM_COMMUNICATION_TEMPLATE_REVISION_CREATED", input.MakerID, created.LegalEntityID, created.Action, created.Locale, created.Version, created.Status, now); err != nil {
		return CommunicationTemplate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CommunicationTemplate{}, err
	}
	return created, nil
}

func (store *PostgresCommunicationStore) GetTemplate(ctx context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, version int64) (CommunicationTemplate, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil || version < 1 {
		return CommunicationTemplate{}, ErrCommunicationNotFound
	}
	row := store.repo.pool.QueryRow(ctx, `
		SELECT c.id::text,c.tenant_id::text,c.legal_entity_id::text,c.action,c.locale,c.version,c.subject_template,c.document,c.status,
		       c.effective_from,c.effective_until,c.maker_id::text,COALESCE(c.checker_id::text,''),COALESCE(c.rollback_origin_version,0),c.created_at,c.updated_at
		FROM form_communication_templates c
		JOIN tenants t ON t.id=c.tenant_id
		JOIN legal_entities le ON le.id=c.legal_entity_id AND le.tenant_id=c.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND c.action=$3 AND c.locale=$4 AND c.version=$5`, tenantID, legalEntityID, action, locale, version)
	value, err := scanCommunicationTemplate(row)
	if err != nil {
		return CommunicationTemplate{}, mapCommunicationPostgresError(err)
	}
	return value, nil
}

func (store *PostgresCommunicationStore) ListTemplates(ctx context.Context, query CommunicationTemplateQuery) ([]CommunicationTemplate, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return nil, ErrCommunicationNotFound
	}
	rows, err := store.repo.pool.Query(ctx, `
		SELECT c.id::text,c.tenant_id::text,c.legal_entity_id::text,c.action,c.locale,c.version,c.subject_template,c.document,c.status,
		       c.effective_from,c.effective_until,c.maker_id::text,COALESCE(c.checker_id::text,''),COALESCE(c.rollback_origin_version,0),c.created_at,c.updated_at
		FROM form_communication_templates c
		JOIN tenants t ON t.id=c.tenant_id
		JOIN legal_entities le ON le.id=c.legal_entity_id AND le.tenant_id=c.tenant_id
		WHERE ($1='' OR t.id::text=$1 OR t.slug=$1)
		  AND ($2='' OR le.id::text=$2 OR le.code=$2)
		  AND ($3='' OR c.action=$3)
		  AND ($4='' OR c.locale=$4)
		  AND ($5='' OR c.status=$5)
		ORDER BY c.action,c.locale,c.version DESC`, query.TenantID, query.LegalEntityID, query.Action, query.Locale, query.Status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]CommunicationTemplate, 0)
	for rows.Next() {
		value, scanErr := scanCommunicationTemplate(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (store *PostgresCommunicationStore) TransitionTemplate(ctx context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, version int64, input CommunicationTransitionInput, now time.Time) (CommunicationTemplate, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil || version < 1 || input.ExpectedVersion != version {
		return CommunicationTemplate{}, ErrCommunicationConflict
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return CommunicationTemplate{}, err
	}
	defer tx.Rollback(ctx)
	canonicalTenantID, canonicalLegalEntityID, err := resolveCommunicationScope(ctx, tx, tenantID, legalEntityID)
	if err != nil {
		return CommunicationTemplate{}, err
	}
	if err := lockCommunicationScope(ctx, tx, "template", canonicalTenantID, canonicalLegalEntityID, action, locale); err != nil {
		return CommunicationTemplate{}, err
	}
	current, err := scanCommunicationTemplate(tx.QueryRow(ctx, `
		SELECT id::text,tenant_id::text,legal_entity_id::text,action,locale,version,subject_template,document,status,
		       effective_from,effective_until,maker_id::text,COALESCE(checker_id::text,''),COALESCE(rollback_origin_version,0),created_at,updated_at
		FROM form_communication_templates
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND action=$3 AND locale=$4 AND version=$5 FOR UPDATE`, canonicalTenantID, canonicalLegalEntityID, action, locale, version))
	if err != nil {
		return CommunicationTemplate{}, mapCommunicationPostgresError(err)
	}
	if input.To == CommunicationActive {
		if err := ValidateCommunicationTemplate(current); err != nil {
			return CommunicationTemplate{}, err
		}
	}
	status, checkerID, effectiveFrom, effectiveUntil, err := applyCommunicationTransition(current.Status, current.MakerID, input, current.EffectiveFrom, current.EffectiveUntil, now.UTC())
	if err != nil {
		return CommunicationTemplate{}, err
	}
	if status == CommunicationActive {
		if _, err := tx.Exec(ctx, `
			UPDATE form_communication_templates
			SET status='RETIRED',updated_at=$7
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND action=$3 AND locale=$4 AND id<>$5::uuid AND status='ACTIVE'
			  AND (effective_until IS NULL OR effective_until>$6)
			  AND ($8::timestamptz IS NULL OR $8::timestamptz>effective_from)`, canonicalTenantID, canonicalLegalEntityID, action, locale, current.ID, effectiveFrom, now.UTC(), effectiveUntil); err != nil {
			return CommunicationTemplate{}, err
		}
	}
	row := tx.QueryRow(ctx, `
		UPDATE form_communication_templates
		SET status=$6,effective_from=$7,effective_until=$8,checker_id=CASE WHEN $9='' THEN checker_id ELSE $9::uuid END,updated_at=$10
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND action=$3 AND locale=$4 AND version=$5
		RETURNING id::text,tenant_id::text,legal_entity_id::text,action,locale,version,subject_template,document,status,
		          effective_from,effective_until,maker_id::text,COALESCE(checker_id::text,''),COALESCE(rollback_origin_version,0),created_at,updated_at`,
		canonicalTenantID, canonicalLegalEntityID, action, locale, version, status, effectiveFrom, effectiveUntil, checkerID, now.UTC())
	updated, err := scanCommunicationTemplate(row)
	if err != nil {
		return CommunicationTemplate{}, mapCommunicationPostgresError(err)
	}
	if err := appendCommunicationGovernanceRecords(ctx, tx, updated.TenantID, updated.ID, "FORM_COMMUNICATION_TEMPLATE", "FORM_COMMUNICATION_TEMPLATE_"+string(updated.Status), input.ActorID, updated.LegalEntityID, updated.Action, updated.Locale, updated.Version, updated.Status, now); err != nil {
		return CommunicationTemplate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CommunicationTemplate{}, err
	}
	return updated, nil
}

func (store *PostgresCommunicationStore) MarkTemplateRollback(ctx context.Context, tenantID, legalEntityID string, action CommunicationAction, locale string, version, sourceVersion int64) (CommunicationTemplate, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil || version < 1 || sourceVersion < 1 {
		return CommunicationTemplate{}, ErrCommunicationInvalid
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return CommunicationTemplate{}, err
	}
	defer tx.Rollback(ctx)
	canonicalTenantID, canonicalLegalEntityID, err := resolveCommunicationScope(ctx, tx, tenantID, legalEntityID)
	if err != nil {
		return CommunicationTemplate{}, err
	}
	if err := lockCommunicationScope(ctx, tx, "template", canonicalTenantID, canonicalLegalEntityID, action, locale); err != nil {
		return CommunicationTemplate{}, err
	}
	var sourceExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM form_communication_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND action=$3 AND locale=$4 AND version=$5)`, canonicalTenantID, canonicalLegalEntityID, action, locale, sourceVersion).Scan(&sourceExists); err != nil {
		return CommunicationTemplate{}, err
	}
	if !sourceExists {
		return CommunicationTemplate{}, ErrCommunicationNotFound
	}
	row := tx.QueryRow(ctx, `
		UPDATE form_communication_templates
		SET rollback_origin_version=$6
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND action=$3 AND locale=$4 AND version=$5
		RETURNING id::text,tenant_id::text,legal_entity_id::text,action,locale,version,subject_template,document,status,
		          effective_from,effective_until,maker_id::text,COALESCE(checker_id::text,''),COALESCE(rollback_origin_version,0),created_at,updated_at`,
		canonicalTenantID, canonicalLegalEntityID, action, locale, version, sourceVersion)
	updated, err := scanCommunicationTemplate(row)
	if err != nil {
		return CommunicationTemplate{}, mapCommunicationPostgresError(err)
	}
	if err := appendCommunicationGovernanceRecords(ctx, tx, updated.TenantID, updated.ID, "FORM_COMMUNICATION_TEMPLATE", "FORM_COMMUNICATION_TEMPLATE_ROLLBACK_MARKED", updated.MakerID, updated.LegalEntityID, updated.Action, updated.Locale, updated.Version, updated.Status, time.Now().UTC()); err != nil {
		return CommunicationTemplate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CommunicationTemplate{}, err
	}
	return updated, nil
}

type communicationRow interface {
	Scan(dest ...any) error
}

func scanCommunicationProfile(row communicationRow) (CommunicationProfile, error) {
	var value CommunicationProfile
	if err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.Version, &value.DefaultLocale, &value.BankName, &value.SupportContact, &value.BrandAssetID, &value.Status, &value.EffectiveFrom, &value.EffectiveUntil, &value.MakerID, &value.CheckerID, &value.RollbackOriginVersion, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return CommunicationProfile{}, err
	}
	value.EffectiveFrom = value.EffectiveFrom.UTC()
	value.EffectiveUntil = cloneOptionalTime(value.EffectiveUntil)
	value.CreatedAt = value.CreatedAt.UTC()
	value.UpdatedAt = value.UpdatedAt.UTC()
	return value, nil
}

func scanCommunicationTemplate(row communicationRow) (CommunicationTemplate, error) {
	var value CommunicationTemplate
	var document []byte
	if err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.Action, &value.Locale, &value.Version, &value.SubjectTemplate, &document, &value.Status, &value.EffectiveFrom, &value.EffectiveUntil, &value.MakerID, &value.CheckerID, &value.RollbackOriginVersion, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return CommunicationTemplate{}, err
	}
	if err := json.Unmarshal(document, &value.Document); err != nil {
		return CommunicationTemplate{}, fmt.Errorf("decode communication document: %w", err)
	}
	value.Document = cloneCommunicationNodes(value.Document)
	value.EffectiveFrom = value.EffectiveFrom.UTC()
	value.EffectiveUntil = cloneOptionalTime(value.EffectiveUntil)
	value.CreatedAt = value.CreatedAt.UTC()
	value.UpdatedAt = value.UpdatedAt.UTC()
	return value, nil
}

func resolveCommunicationScope(ctx context.Context, tx pgx.Tx, tenantID, legalEntityID string) (string, string, error) {
	var canonicalTenantID, canonicalLegalEntityID string
	err := tx.QueryRow(ctx, `
		SELECT t.id::text,le.id::text
		FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2)
		ORDER BY le.valid_from DESC LIMIT 1`, strings.TrimSpace(tenantID), strings.TrimSpace(legalEntityID)).Scan(&canonicalTenantID, &canonicalLegalEntityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrCommunicationNotFound
	}
	return canonicalTenantID, canonicalLegalEntityID, err
}

func lockCommunicationScope(ctx context.Context, tx pgx.Tx, kind, tenantID, legalEntityID string, action CommunicationAction, locale string) error {
	key := strings.Join([]string{"form-communication", kind, tenantID, legalEntityID, string(action), locale}, "\x00")
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key)
	return err
}

func appendCommunicationGovernanceRecords(ctx context.Context, tx pgx.Tx, tenantID, aggregateID, aggregateType, eventType, actorID, legalEntityID string, action CommunicationAction, locale string, version int64, status CommunicationStatus, now time.Time) error {
	metadata := map[string]any{
		"legal_entity_id": legalEntityID,
		"version":         version,
		"status":          status,
	}
	if action != "" {
		metadata["action"] = action
	}
	if locale != "" {
		metadata["locale"] = locale
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(tenant_id,actor_id,event_type,subject_type,subject_id,purpose,safe_metadata,occurred_at)
		VALUES($1::uuid,NULLIF($2,'')::uuid,$3,$4,$5::uuid,'governed form communication configuration',$6::jsonb,$7)`, tenantID, actorID, eventType, aggregateType, aggregateID, string(payload), now.UTC()); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		VALUES($1::uuid,$2,$3::uuid,$4,$5::jsonb,$6,$6,$6)`, tenantID, aggregateType, aggregateID, eventType, string(payload), now.UTC())
	return err
}

func mapCommunicationPostgresError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCommunicationNotFound
	}
	return err
}

var _ communicationStore = (*PostgresCommunicationStore)(nil)
