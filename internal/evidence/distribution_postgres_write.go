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
	var presentationJSON, scoreProfileJSON, sectionsJSON, fieldsJSON []byte
	err := tx.QueryRow(ctx, `
		SELECT f.id::text,f.tenant_id::text,f.legal_entity_id::text,f.version,f.sensitivity,f.presentation,f.scoring_mode,f.score_profile,f.sections,f.fields
		FROM monitoring_form_templates f
		JOIN tenants t ON t.id=f.tenant_id
		JOIN legal_entities le ON le.id=f.legal_entity_id AND le.tenant_id=f.tenant_id
		WHERE f.id=$1::uuid AND f.version=$2
		  AND (t.id::text=$3 OR t.slug=$3)
		  AND (le.id::text=$4 OR le.code=$4)
		  AND f.status='ACTIVE' AND f.is_current
		FOR KEY SHARE`, input.FormTemplateID, input.FormTemplateVersion, input.TenantID, input.LegalEntityID).Scan(
		&form.ID, &form.TenantID, &form.LegalEntityID, &form.Version, &form.Sensitivity,
		&presentationJSON, &form.ScoringMode, &scoreProfileJSON, &sectionsJSON, &fieldsJSON,
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
	if len(scoreProfileJSON) > 0 {
		if err := json.Unmarshal(scoreProfileJSON, &form.ScoreProfile); err != nil {
			return DistributionFormRevision{}, err
		}
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

func insertDistributionRequest(ctx context.Context, tx pgx.Tx, distribution FormDistribution, recipient *postgresPreparedRecipient, form DistributionFormRevision, input CreateDistributionInput, now time.Time) error {
	request, err := materializeDistributionRequest(recipient.safe.RequestID, distribution, recipient.safe, form, input, now)
	if err != nil {
		return err
	}
	presentationJSON, err := json.Marshal(request.Presentation)
	if err != nil {
		return err
	}
	sectionsJSON, err := json.Marshal(request.Sections)
	if err != nil {
		return err
	}
	fieldsJSON, err := json.Marshal(request.Fields)
	if err != nil {
		return err
	}
	knownFactsJSON, err := json.Marshal(request.KnownFacts)
	if err != nil {
		return err
	}
	sourceBindingsJSON, err := json.Marshal(request.SourceBindings)
	if err != nil {
		return err
	}
	scoreProfileJSON, err := marshalScoreProfile(request.ScoreProfile)
	if err != nil {
		return err
	}
	audienceType := request.AudienceType
	principalID := ""
	var audienceHash any
	hint := recipient.safe.AudienceHint
	if recipient.safe.Type == RecipientInternalPrincipal {
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
			estimated_minutes,deadline,known_facts,presentation,scoring_mode,score_profile,sections,fields,source_bindings,form_template_id,form_template_version,
			collection_period_start,collection_period_end,origin_type,origin_id,origin_version,status,created_by,version,created_at,updated_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,$11,
			$12,NULLIF($13,'')::uuid,$14,$15,'ASSIGNED',1,'',$16,$17,$18::jsonb,$19::jsonb,$20,$21::jsonb,$22::jsonb,$23::jsonb,$24::jsonb,$25::uuid,$26,
			$27,$28,NULLIF($29,''),NULLIF($30,''),NULLIF($31,0),'READY',$32::uuid,1,$33,$33
		)`, recipient.safe.RequestID, distribution.TenantID, distribution.LegalEntityID, distribution.ID,
		request.SubjectType, request.SubjectID, request.Title, request.Purpose, request.WhyYou, request.Sensitivity,
		audienceType, recipient.safe.Type, principalID, audienceHash, hint, request.EstimatedMinutes, request.Deadline,
		string(knownFactsJSON), string(presentationJSON), request.ScoringMode, scoreProfileJSON, string(sectionsJSON), string(fieldsJSON), string(sourceBindingsJSON), form.ID, form.Version,
		request.CollectionPeriodStart, request.CollectionPeriodEnd, request.Origin.Type, request.Origin.ID, request.Origin.Version, distribution.CreatedBy, now)
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
