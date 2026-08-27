//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func loadExactActiveDistributionForm(ctx context.Context, tx pgx.Tx, input CreateDistributionInput) (DistributionFormRevision, error) {
	var form DistributionFormRevision
	var presentationJSON, sectionsJSON, fieldsJSON []byte
	err := tx.QueryRow(ctx, `
		SELECT f.id::text,f.tenant_id::text,f.legal_entity_id::text,f.version,f.sensitivity,f.presentation,f.sections,f.fields
		FROM monitoring_form_templates f
		JOIN tenants t ON t.id=f.tenant_id
		JOIN legal_entities le ON le.id=f.legal_entity_id AND le.tenant_id=f.tenant_id
		WHERE f.id=$1::uuid AND f.version=$2
		  AND (t.id::text=$3 OR t.slug=$3)
		  AND (le.id::text=$4 OR le.code=$4)
		  AND f.status='ACTIVE' AND f.is_current
		FOR KEY SHARE`, input.FormTemplateID, input.FormTemplateVersion, input.TenantID, input.LegalEntityID).Scan(
		&form.ID, &form.TenantID, &form.LegalEntityID, &form.Version, &form.Sensitivity,
		&presentationJSON, &sectionsJSON, &fieldsJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DistributionFormRevision{}, fmt.Errorf("form revision must be the exact active revision in the requested legal entity")
	}
	if err != nil {
		return DistributionFormRevision{}, fmt.Errorf("load active form revision: %w", err)
	}
	if err := json.Unmarshal(presentationJSON, &form.Presentation); err != nil {
		return DistributionFormRevision{}, err
	}
	if err := json.Unmarshal(sectionsJSON, &form.Sections); err != nil {
		return DistributionFormRevision{}, err
	}
	if err := json.Unmarshal(fieldsJSON, &form.Fields); err != nil {
		return DistributionFormRevision{}, err
	}
	form.Active = true
	return form, nil
}

func insertFormDistribution(ctx context.Context, tx pgx.Tx, distribution FormDistribution) error {
	reminderPolicy, err := json.Marshal(cloneAnyMap(distribution.ReminderPolicy))
	if err != nil {
		return fmt.Errorf("encode reminder policy: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_form_distributions(
			id,tenant_id,legal_entity_id,form_template_id,form_template_version,subject_type,subject_id,
			title,purpose,access_policy,status,deadline,route_expires_at,reminder_policy,created_by,version,created_at,updated_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7::uuid,$8,$9,$10,$11,$12,$13,$14::jsonb,$15::uuid,1,$16,$16
		)`, distribution.ID, distribution.TenantID, distribution.LegalEntityID, distribution.FormTemplateID,
		distribution.FormTemplateVersion, distribution.SubjectType, distribution.SubjectID, distribution.Title,
		distribution.Purpose, distribution.AccessPolicy, distribution.Status, distribution.Deadline,
		distribution.RouteExpiresAt, string(reminderPolicy), distribution.CreatedBy, distribution.CreatedAt); err != nil {
		return fmt.Errorf("insert form distribution: %w", err)
	}
	return nil
}

func insertDistributionRequest(ctx context.Context, tx pgx.Tx, distribution FormDistribution, recipient *postgresPreparedRecipient, form DistributionFormRevision, estimatedMinutes int, now time.Time) error {
	presentationJSON, err := json.Marshal(form.Presentation)
	if err != nil {
		return err
	}
	sectionsJSON, err := json.Marshal(form.Sections)
	if err != nil {
		return err
	}
	fieldsJSON, err := json.Marshal(form.Fields)
	if err != nil {
		return err
	}
	audienceType := "EXTERNAL"
	principalID := ""
	var audienceHash any
	hint := recipient.safe.AudienceHint
	if recipient.safe.Type == RecipientInternalPrincipal {
		audienceType = "INTERNAL"
		principalID = recipient.safe.PrincipalID
		hint = ""
	} else {
		audienceHash = recipient.protected.Hash
		if hint == "" {
			return fmt.Errorf("external TO recipient requires an audience hint")
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO capture_requests(
			id,tenant_id,legal_entity_id,distribution_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,
			recipient_type,recipient_principal_id,recipient_audience_hash,recipient_hint,recipient_state,recipient_revision,recipient_issue_reason,
			estimated_minutes,deadline,known_facts,presentation,sections,fields,source_bindings,form_template_id,form_template_version,
			status,created_by,version,created_at,updated_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$8,$9,$10,
			$11,NULLIF($12,'')::uuid,$13,$14,'ASSIGNED',1,'',$15,$16,'{}'::jsonb,$17::jsonb,$18::jsonb,$19::jsonb,'[]'::jsonb,$20::uuid,$21,
			'READY',$22::uuid,1,$23,$23
		)`, recipient.safe.RequestID, distribution.TenantID, distribution.LegalEntityID, distribution.ID,
		distribution.SubjectType, distribution.SubjectID, distribution.Title, distribution.Purpose, form.Sensitivity,
		audienceType, recipient.safe.Type, principalID, audienceHash, hint, estimatedMinutes, distribution.Deadline,
		string(presentationJSON), string(sectionsJSON), string(fieldsJSON), form.ID, form.Version, distribution.CreatedBy, now)
	if err != nil {
		return fmt.Errorf("insert TO capture request: %w", err)
	}
	return nil
}

func insertDistributionRecipient(ctx context.Context, tx pgx.Tx, distribution FormDistribution, recipient *postgresPreparedRecipient) error {
	var addressHash, ciphertext any
	keyID := ""
	if recipient.safe.Type == RecipientExternalAudience {
		addressHash = recipient.protected.Hash
		ciphertext = recipient.protected.Ciphertext
		keyID = recipient.protected.KeyID
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO capture_distribution_recipients(
			id,distribution_id,tenant_id,legal_entity_id,role,recipient_type,principal_id,request_id,
			address_hash,address_ciphertext,address_key_id,audience_hint,contact_label,state,version,created_at,updated_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,NULLIF($7,'')::uuid,NULLIF($8,'')::uuid,
			$9,$10,NULLIF($11,''),$12,$13,$14,1,$15,$15
		)`, recipient.safe.ID, distribution.ID, distribution.TenantID, distribution.LegalEntityID,
		recipient.safe.Role, recipient.safe.Type, recipient.safe.PrincipalID, recipient.safe.RequestID,
		addressHash, ciphertext, keyID, recipient.safe.AudienceHint, recipient.safe.ContactLabel,
		recipient.safe.State, recipient.safe.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert distribution recipient: %w", err)
	}
	return nil
}

func insertDistributionWorkspace(ctx context.Context, tx pgx.Tx, workspace ResponseWorkspace) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_response_workspaces(id,tenant_id,legal_entity_id,distribution_id,status,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,1,$6,$6)`, workspace.ID, workspace.TenantID,
		workspace.LegalEntityID, workspace.DistributionID, workspace.Status, workspace.CreatedAt); err != nil {
		return fmt.Errorf("insert response workspace: %w", err)
	}
	return nil
}

func insertDistributionCreatedEvents(ctx context.Context, tx pgx.Tx, distribution FormDistribution, recipientCount int, now time.Time) error {
	payloadJSON, err := json.Marshal(map[string]any{
		"version":               distribution.Version,
		"form_template_id":      distribution.FormTemplateID,
		"form_template_version": distribution.FormTemplateVersion,
		"recipient_count":       recipientCount,
	})
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO capture_distribution_events(
			tenant_id,legal_entity_id,distribution_id,distribution_version,event_type,payload,actor_id,occurred_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,1,'FORM_DISTRIBUTION_CREATED',$4::jsonb,$5::uuid,$6)`,
		distribution.TenantID, distribution.LegalEntityID, distribution.ID, string(payloadJSON), distribution.CreatedBy, now); err != nil {
		return fmt.Errorf("insert distribution event: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events(
			tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at
		) VALUES($1::uuid,'FORM_DISTRIBUTION',$2::uuid,'FORM_DISTRIBUTION_CREATED',$3::jsonb,$4,$4,$4)`,
		distribution.TenantID, distribution.ID, string(payloadJSON), now); err != nil {
		return fmt.Errorf("insert distribution outbox event: %w", err)
	}
	return nil
}
