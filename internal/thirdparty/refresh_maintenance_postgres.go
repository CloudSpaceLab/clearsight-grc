//go:build postgres

package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type postgresRefreshCandidate struct {
	tenantID, tenantSlug, legalEntityID, relationshipID, ownerPrincipalID string
	vendorVersion                                                         int64
	vendorUpdatedAt                                                       time.Time
}

type postgresRefreshDocument struct {
	id, assessmentID, artifactID, documentType string
	version                                    int64
	expiresOn                                  time.Time
}

func (r *PostgresRepository) MaintainVendorRefresh(ctx context.Context, now time.Time, policy RefreshMaintenancePolicy) (RefreshBatchReceipt, error) {
	if r == nil || r.pool == nil || !validRefreshMaintenancePolicy(policy) {
		return RefreshBatchReceipt{}, ErrInvalidRefreshMaintenancePolicy
	}
	now = now.UTC()
	today := calendarDay(now)
	leadDate := calendarDay(today.Add(policy.DocumentLead))
	factCutoff := now.Add(-policy.FactConfirmationInterval)
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return RefreshBatchReceipt{}, fmt.Errorf("begin vendor refresh maintenance: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT r.tenant_id::text,t.slug,r.legal_entity_id::text,r.id::text,r.business_owner_principal_id::text,p.version,p.updated_at
		FROM third_party_relationships r
		JOIN third_parties p ON p.id=r.vendor_id AND p.tenant_id=r.tenant_id
		JOIN tenants t ON t.id=r.tenant_id
		CROSS JOIN LATERAL (
			SELECT
				(CASE WHEN p.updated_at<=$1 THEN jsonb_build_object(
					'VENDOR.IDENTITY.LEGAL_NAME',p.version,
					'VENDOR.IDENTITY.TRADING_NAME',p.version,
					'VENDOR.IDENTITY.REGISTRATION_REFERENCE',p.version,
					'VENDOR.IDENTITY.JURISDICTION',p.version,
					'VENDOR.IDENTITY.REGISTERED_ADDRESS',p.version,
					'VENDOR.IDENTITY.WEBSITE_DOMAIN',p.version
				) ELSE '{}'::jsonb END)
				|| COALESCE((
					SELECT jsonb_object_agg(due.target_key,due.version ORDER BY due.expires_on,due.id)
					FROM (
						SELECT 'VENDOR.DOCUMENT.'||upper(btrim(d.document_type)) AS target_key,d.version,d.expires_on,d.id
						FROM third_party_documents d
						WHERE d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id AND d.relationship_id=r.id
						  AND d.status='VALIDATED' AND d.expires_on IS NOT NULL AND d.expires_on<=$2::date
						ORDER BY d.expires_on,d.id
					) due
				), '{}'::jsonb) AS observed_versions
		) candidate
		WHERE (p.updated_at <= $1
		   OR EXISTS (
			SELECT 1 FROM third_party_documents d
			WHERE d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id AND d.relationship_id=r.id
			  AND d.status='VALIDATED' AND d.expires_on IS NOT NULL AND d.expires_on <= $2::date
		   ))
		  AND NOT EXISTS (
			SELECT 1 FROM third_party_refresh_attentions attention
			WHERE attention.tenant_id=r.tenant_id AND attention.legal_entity_id=r.legal_entity_id
			  AND attention.relationship_id=r.id AND attention.state='OPEN'
			  AND attention.observed_versions @> candidate.observed_versions
		  )
		ORDER BY r.id
		LIMIT $3
		FOR UPDATE OF r,p SKIP LOCKED`, factCutoff, leadDate, policy.BatchSize)
	if err != nil {
		return RefreshBatchReceipt{}, fmt.Errorf("claim vendor refresh candidates: %w", err)
	}
	candidates := make([]postgresRefreshCandidate, 0, policy.BatchSize)
	for rows.Next() {
		var candidate postgresRefreshCandidate
		if err := rows.Scan(&candidate.tenantID, &candidate.tenantSlug, &candidate.legalEntityID, &candidate.relationshipID, &candidate.ownerPrincipalID, &candidate.vendorVersion, &candidate.vendorUpdatedAt); err != nil {
			rows.Close()
			return RefreshBatchReceipt{}, fmt.Errorf("scan vendor refresh candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return RefreshBatchReceipt{}, fmt.Errorf("read vendor refresh candidates: %w", err)
	}
	rows.Close()

	receipt := RefreshBatchReceipt{RelationshipsExamined: len(candidates)}
	proofs := make([]thirdPartyCommitProof, 0)
	for _, claimed := range candidates {
		candidate := RefreshCandidate{
			Scope:            Scope{TenantID: claimed.tenantSlug, LegalEntityID: claimed.legalEntityID},
			RelationshipID:   claimed.relationshipID,
			ObservedVersions: map[string]int64{},
		}
		reasons := make([]string, 0, 2)
		if !claimed.vendorUpdatedAt.After(factCutoff) {
			candidate.TargetKeys = append(candidate.TargetKeys, refreshIdentityTargetKeys...)
			for _, key := range refreshIdentityTargetKeys {
				candidate.ObservedVersions[key] = claimed.vendorVersion
			}
			reasons = append(reasons, "HELD_VENDOR_FACTS_CONFIRMATION_DUE")
		}

		documents, err := lockDueRefreshDocuments(ctx, tx, claimed, leadDate)
		if err != nil {
			return receipt, err
		}
		for _, document := range documents {
			targetKey := "VENDOR.DOCUMENT." + strings.ToUpper(strings.TrimSpace(document.documentType))
			candidate.TargetKeys = append(candidate.TargetKeys, targetKey)
			candidate.ObservedVersions[targetKey] = document.version
			reasons = append(reasons, "VENDOR_DOCUMENT_EXPIRY_DUE")
			if calendarDay(document.expiresOn.UTC()).After(today) {
				continue
			}
			proof, err := r.expireRefreshDocument(ctx, tx, claimed, document, now)
			if err != nil {
				return receipt, err
			}
			proofs = append(proofs, proof)
			receipt.DocumentsExpired++
		}

		candidate.TargetKeys = uniqueSortedStrings(candidate.TargetKeys)
		candidate.Reason = strings.Join(uniqueSortedStrings(reasons), "+")
		if len(candidate.TargetKeys) == 0 {
			continue
		}
		attention, eventID, created, err := insertRefreshAttention(ctx, tx, claimed, candidate, now)
		if err != nil {
			return receipt, err
		}
		if created {
			proofs = append(proofs, refreshAttentionCommitProof(eventID, attention, "VendorRefreshAttentionCreated"))
			receipt.AttentionsCreated++
		}
	}
	if len(proofs) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return receipt, fmt.Errorf("commit vendor refresh maintenance: %w", err)
		}
		return receipt, nil
	}
	if err := r.commitThirdPartyEvents(ctx, tx, proofs...); err != nil {
		return receipt, fmt.Errorf("commit vendor refresh maintenance: %w", err)
	}
	return receipt, nil
}

func lockDueRefreshDocuments(ctx context.Context, tx pgx.Tx, candidate postgresRefreshCandidate, leadDate time.Time) ([]postgresRefreshDocument, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text,assessment_id::text,artifact_id::text,document_type,version,expires_on
		FROM third_party_documents
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND relationship_id=$3::uuid
		  AND status='VALIDATED' AND expires_on IS NOT NULL AND expires_on <= $4::date
		ORDER BY expires_on,id
		FOR UPDATE`, candidate.tenantID, candidate.legalEntityID, candidate.relationshipID, leadDate)
	if err != nil {
		return nil, fmt.Errorf("lock due vendor documents: %w", err)
	}
	defer rows.Close()
	documents := make([]postgresRefreshDocument, 0)
	for rows.Next() {
		var document postgresRefreshDocument
		if err := rows.Scan(&document.id, &document.assessmentID, &document.artifactID, &document.documentType, &document.version, &document.expiresOn); err != nil {
			return nil, fmt.Errorf("scan due vendor document: %w", err)
		}
		documents = append(documents, document)
	}
	return documents, rows.Err()
}

func (r *PostgresRepository) expireRefreshDocument(ctx context.Context, tx pgx.Tx, claimed postgresRefreshCandidate, document postgresRefreshDocument, now time.Time) (thirdPartyCommitProof, error) {
	assessment, err := lockAssessment(ctx, tx, claimed.tenantID, claimed.legalEntityID, document.assessmentID)
	if err != nil {
		return thirdPartyCommitProof{}, err
	}
	result, err := tx.Exec(ctx, `
		UPDATE third_party_documents SET status='EXPIRED',version=version+1,updated_at=$5
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$4 AND status='VALIDATED'`,
		claimed.tenantID, claimed.legalEntityID, document.id, document.version, now)
	if err != nil {
		return thirdPartyCommitProof{}, fmt.Errorf("expire vendor document: %w", err)
	}
	if result.RowsAffected() != 1 {
		return thirdPartyCommitProof{}, ErrVersionConflict
	}
	assessment.Version++
	assessment.UpdatedAt = now
	if err := updateAssessment(ctx, tx, claimed.tenantID, assessment); err != nil {
		return thirdPartyCommitProof{}, err
	}
	value := AssessmentDocument{
		ID: document.id, Scope: Scope{TenantID: claimed.tenantSlug, LegalEntityID: claimed.legalEntityID}, RelationshipID: claimed.relationshipID,
		AssessmentID: assessment.ID, ArtifactID: document.artifactID, DocumentType: document.documentType, Status: AssessmentDocumentExpired,
		Version: document.version + 1, UpdatedAt: now,
	}
	eventID, err := appendExpiredAssessmentDocumentEvent(ctx, tx, claimed.tenantID, assessment, value)
	if err != nil {
		return thirdPartyCommitProof{}, err
	}
	return assessmentCommitProof(eventID, assessment, "AssessmentDocumentExpired"), nil
}

func appendExpiredAssessmentDocumentEvent(ctx context.Context, tx pgx.Tx, tenantID string, assessment Assessment, document AssessmentDocument) (string, error) {
	var eventID string
	err := tx.QueryRow(ctx, `
		INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at)
		VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,$3,NULL,'AssessmentDocumentExpired',
			jsonb_build_object('status',$4::text,'relationship_id',$5::text,'artifact_id',$6::text,'document_id',$7::text,'document_status','EXPIRED'),$8)
		RETURNING id::text`, tenantID, assessment.ID, assessment.Version, assessment.Status, assessment.RelationshipID, document.ArtifactID, document.ID, assessment.UpdatedAt).Scan(&eventID)
	if err != nil {
		return "", fmt.Errorf("append assessment document expiry event: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,'AssessmentDocumentExpired',
			jsonb_build_object('version',$3::bigint,'status',$4::text,'relationship_id',$5::text,'artifact_id',$6::text,'document_id',$7::text,'document_status','EXPIRED'),$8,$8)`,
		tenantID, assessment.ID, assessment.Version, assessment.Status, assessment.RelationshipID, document.ArtifactID, document.ID, assessment.UpdatedAt)
	if err != nil {
		return "", fmt.Errorf("append assessment document expiry outbox event: %w", err)
	}
	return eventID, nil
}

func insertRefreshAttention(ctx context.Context, tx pgx.Tx, claimed postgresRefreshCandidate, candidate RefreshCandidate, now time.Time) (RefreshAttention, string, bool, error) {
	observedJSON, err := json.Marshal(candidate.ObservedVersions)
	if err != nil {
		return RefreshAttention{}, "", false, fmt.Errorf("encode refresh versions: %w", err)
	}
	var covered bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM third_party_refresh_attentions
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND relationship_id=$3::uuid
			  AND state='OPEN' AND observed_versions @> $4::jsonb
		)`, claimed.tenantID, claimed.legalEntityID, claimed.relationshipID, string(observedJSON)).Scan(&covered)
	if err != nil {
		return RefreshAttention{}, "", false, fmt.Errorf("check vendor refresh attention coverage: %w", err)
	}
	if covered {
		return RefreshAttention{}, "", false, nil
	}
	targets := append([]string(nil), candidate.TargetKeys...)
	sort.Strings(targets)
	targetsJSON, err := json.Marshal(targets)
	if err != nil {
		return RefreshAttention{}, "", false, fmt.Errorf("encode refresh targets: %w", err)
	}
	attention := RefreshAttention{
		Scope: Scope{TenantID: claimed.tenantSlug, LegalEntityID: claimed.legalEntityID}, RelationshipID: claimed.relationshipID,
		OwnerPrincipalID: claimed.ownerPrincipalID, TargetKeys: targets, Reason: candidate.Reason,
		ObservedVersions: cloneVersionMap(candidate.ObservedVersions), DedupeKey: refreshDedupeKey(candidate), State: RefreshAttentionOpen,
		CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO third_party_refresh_attentions(
			tenant_id,legal_entity_id,relationship_id,owner_principal_id,target_keys,reason,observed_versions,dedupe_key,state,version,created_at,updated_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::jsonb,$6,$7::jsonb,$8,'OPEN',1,$9,$9)
		ON CONFLICT (tenant_id,dedupe_key) DO NOTHING
		RETURNING id::text`, claimed.tenantID, claimed.legalEntityID, claimed.relationshipID, claimed.ownerPrincipalID, string(targetsJSON), candidate.Reason, string(observedJSON), attention.DedupeKey, now).Scan(&attention.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return RefreshAttention{}, "", false, nil
	}
	if err != nil {
		return RefreshAttention{}, "", false, fmt.Errorf("store vendor refresh attention: %w", err)
	}
	var eventID string
	err = tx.QueryRow(ctx, `
		INSERT INTO third_party_refresh_attention_events(
			tenant_id,legal_entity_id,attention_id,relationship_id,attention_version,event_type,payload,occurred_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,1,'VendorRefreshAttentionCreated',
			jsonb_build_object('target_keys',$5::jsonb,'reason',$6::text,'observed_versions',$7::jsonb),$8)
		RETURNING id::text`, claimed.tenantID, claimed.legalEntityID, attention.ID, claimed.relationshipID, string(targetsJSON), candidate.Reason, string(observedJSON), now).Scan(&eventID)
	if err != nil {
		return RefreshAttention{}, "", false, fmt.Errorf("append vendor refresh attention event: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,'THIRD_PARTY_REFRESH_ATTENTION',$2::uuid,'VendorRefreshAttentionCreated',
			jsonb_build_object('version',1,'legal_entity_id',$3::text,'relationship_id',$4::text,'owner_principal_id',$5::text,'target_keys',$6::jsonb,'reason',$7::text),$8,$8)`,
		claimed.tenantID, attention.ID, claimed.legalEntityID, claimed.relationshipID, claimed.ownerPrincipalID, string(targetsJSON), candidate.Reason, now)
	if err != nil {
		return RefreshAttention{}, "", false, fmt.Errorf("append vendor refresh attention outbox event: %w", err)
	}
	return attention, eventID, true, nil
}
