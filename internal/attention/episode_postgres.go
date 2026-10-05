//go:build postgres

package attention

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EpisodeProjector struct {
	pool *pgxpool.Pool
}

type sourceSnapshot struct {
	ID             string
	TenantID       string
	LegalEntityID  string
	GeneratedAt    time.Time
	Definition     string
	SourceRevision string
}

type metricQuality struct {
	Complete bool
}

type metricMember struct {
	Condition      string
	MemberID       string
	SubjectType    string
	SubjectID      string
	ConditionState string
}

type openEpisode struct {
	ID             string
	Condition      string
	MemberID       string
	SubjectType    string
	SubjectID      string
	ConditionState string
	NoticeSequence int
}

func NewEpisodeProjector(pool *pgxpool.Pool) *EpisodeProjector {
	return &EpisodeProjector{pool: pool}
}

func (p *EpisodeProjector) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	sourceEvent, relevant, err := DecodeSourceEvent(event)
	if err != nil || !relevant {
		return err
	}
	if p == nil || p.pool == nil {
		return fmt.Errorf("attention episode projector is unavailable")
	}
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	inserted, err := recordSourceReceipt(ctx, tx, event)
	if err != nil || !inserted {
		if err == nil {
			err = tx.Commit(ctx)
		}
		return err
	}
	source, err := loadSourceSnapshot(ctx, tx, event, sourceEvent)
	if err != nil {
		return err
	}
	quality, err := loadMetricQuality(ctx, tx, source)
	if err != nil {
		return err
	}
	members, err := loadMetricMembers(ctx, tx, source)
	if err != nil {
		return err
	}

	current := make(map[string]metricMember, len(members))
	for _, member := range members {
		key := episodeMemberKey(member.Condition, member.MemberID)
		current[key] = member
		if err := p.observeMember(ctx, tx, source, member); err != nil {
			return err
		}
	}
	open, err := loadOpenEpisodes(ctx, tx, source)
	if err != nil {
		return err
	}
	for _, episode := range open {
		if _, ok := current[episodeMemberKey(episode.Condition, episode.MemberID)]; ok {
			continue
		}
		if !quality[episode.Condition].Complete {
			continue
		}
		if err := p.clearEpisode(ctx, tx, source, episode); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func recordSourceReceipt(ctx context.Context, tx pgx.Tx, event workflowruntime.OutboxEvent) (bool, error) {
	tag, err := tx.Exec(ctx, `
		INSERT INTO inbox_receipts(tenant_id,consumer,event_id,processed_at)
		SELECT tenant.id,$2,$3,$4
		FROM tenants tenant
		WHERE tenant.id::text=$1 OR tenant.slug=$1
		ON CONFLICT(tenant_id,consumer,event_id) DO NOTHING`,
		event.TenantID, ConsumerName, event.ID, event.OccurredAt.UTC())
	if err != nil {
		return false, fmt.Errorf("record attention source receipt: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func loadSourceSnapshot(ctx context.Context, tx pgx.Tx, event workflowruntime.OutboxEvent, expected SourceEvent) (sourceSnapshot, error) {
	var source sourceSnapshot
	err := tx.QueryRow(ctx, `
		SELECT source.id::text,tenant.id::text,source.legal_entity_id::text,source.generated_at,
		       source.definition_revision,source.source_revision
		FROM domain_metric_snapshots source
		JOIN tenants tenant ON tenant.id=source.tenant_id
		WHERE source.id=$2::uuid
		  AND (tenant.id::text=$1 OR tenant.slug=$1)
		  AND source.legal_entity_id=$3::uuid
		  AND source.definition_revision=$4
		  AND source.source_revision=$5`,
		event.TenantID, expected.SourceID, expected.LegalEntityID, expected.DefinitionRevision, expected.SourceRevision,
	).Scan(&source.ID, &source.TenantID, &source.LegalEntityID, &source.GeneratedAt, &source.Definition, &source.SourceRevision)
	if errors.Is(err, pgx.ErrNoRows) {
		return sourceSnapshot{}, fmt.Errorf("attention source snapshot is unavailable")
	}
	if err != nil {
		return sourceSnapshot{}, fmt.Errorf("load attention source snapshot: %w", err)
	}
	source.GeneratedAt = source.GeneratedAt.UTC()
	return source, nil
}

func loadMetricQuality(ctx context.Context, tx pgx.Tx, source sourceSnapshot) (map[string]metricQuality, error) {
	rows, err := tx.Query(ctx, `
		SELECT metric_id,completeness,COALESCE(unknown,0)
		FROM metric_observations
		WHERE tenant_id=$1::uuid
		  AND legal_entity_id=$2::uuid
		  AND source_kind='DOMAIN_SNAPSHOT'
		  AND source_id=$3::uuid
		  AND definition_revision=$4`,
		source.TenantID, source.LegalEntityID, source.ID, source.Definition)
	if err != nil {
		return nil, fmt.Errorf("load attention metric quality: %w", err)
	}
	defer rows.Close()
	result := map[string]metricQuality{}
	for rows.Next() {
		var metricID, completeness string
		var unknown int
		if err := rows.Scan(&metricID, &completeness, &unknown); err != nil {
			return nil, err
		}
		result[metricID] = metricQuality{Complete: completeness == "COMPLETE" && unknown == 0}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, condition := range []string{"risks_outside_appetite", "indicator_breaches", "assurance_failures", "losses_without_issue"} {
		if _, ok := result[condition]; !ok {
			return nil, fmt.Errorf("attention metric quality is incomplete for %s", condition)
		}
	}
	return result, nil
}

func loadMetricMembers(ctx context.Context, tx pgx.Tx, source sourceSnapshot) ([]metricMember, error) {
	rows, err := tx.Query(ctx, `
		SELECT metric_id,member_id::text,target_type,target_id::text,state
		FROM domain_metric_snapshot_memberships
		WHERE source_id=$1::uuid
		  AND definition_revision=$2
		ORDER BY metric_id,member_id`, source.ID, source.Definition)
	if err != nil {
		return nil, fmt.Errorf("load attention metric members: %w", err)
	}
	defer rows.Close()
	result := []metricMember{}
	for rows.Next() {
		var value metricMember
		if err := rows.Scan(&value.Condition, &value.MemberID, &value.SubjectType, &value.SubjectID, &value.ConditionState); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func loadOpenEpisodes(ctx context.Context, tx pgx.Tx, source sourceSnapshot) ([]openEpisode, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text,condition_key,member_id::text,subject_type,subject_id::text,last_condition_state,notice_sequence
		FROM attention_episodes
		WHERE tenant_id=$1::uuid
		  AND legal_entity_id=$2::uuid
		  AND state='OPEN'
		ORDER BY condition_key,member_id
		FOR UPDATE`, source.TenantID, source.LegalEntityID)
	if err != nil {
		return nil, fmt.Errorf("load open attention episodes: %w", err)
	}
	defer rows.Close()
	result := []openEpisode{}
	for rows.Next() {
		var value openEpisode
		if err := rows.Scan(
			&value.ID, &value.Condition, &value.MemberID, &value.SubjectType, &value.SubjectID,
			&value.ConditionState, &value.NoticeSequence,
		); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (p *EpisodeProjector) observeMember(ctx context.Context, tx pgx.Tx, source sourceSnapshot, member metricMember) error {
	var episode openEpisode
	err := tx.QueryRow(ctx, `
		SELECT id::text,condition_key,member_id::text,subject_type,subject_id::text,last_condition_state,notice_sequence
		FROM attention_episodes
		WHERE tenant_id=$1::uuid
		  AND legal_entity_id=$2::uuid
		  AND condition_key=$3
		  AND member_id=$4::uuid
		  AND state='OPEN'
		FOR UPDATE`, source.TenantID, source.LegalEntityID, member.Condition, member.MemberID).
		Scan(&episode.ID, &episode.Condition, &episode.MemberID, &episode.SubjectType, &episode.SubjectID, &episode.ConditionState, &episode.NoticeSequence)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return p.openEpisode(ctx, tx, source, member)
	case err != nil:
		return fmt.Errorf("load attention episode: %w", err)
	}

	worsened := conditionStateRank(member.ConditionState) > conditionStateRank(episode.ConditionState)
	nextSequence := episode.NoticeSequence
	if worsened {
		nextSequence++
	}
	if _, err := tx.Exec(ctx, `
		UPDATE attention_episodes
		SET last_source_id=$5::uuid,last_condition_state=$6,last_observed_at=$7,
		    notice_sequence=$8,updated_at=$7
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
		  AND id=$3::uuid AND state='OPEN' AND member_id=$4::uuid`,
		source.TenantID, source.LegalEntityID, episode.ID, member.MemberID,
		source.ID, member.ConditionState, source.GeneratedAt, nextSequence); err != nil {
		return fmt.Errorf("update attention episode: %w", err)
	}
	if !worsened {
		return nil
	}
	episode.ConditionState = member.ConditionState
	episode.NoticeSequence = nextSequence
	return p.emitEpisodeIntents(ctx, tx, source, episode, EventEpisodeWorsened)
}

func (p *EpisodeProjector) openEpisode(ctx context.Context, tx pgx.Tx, source sourceSnapshot, member metricMember) error {
	var episode openEpisode
	err := tx.QueryRow(ctx, `
		INSERT INTO attention_episodes(
			tenant_id,legal_entity_id,condition_key,member_id,subject_type,subject_id,state,
			last_condition_state,opened_source_id,last_source_id,opened_at,last_observed_at,notice_sequence
		) VALUES($1::uuid,$2::uuid,$3,$4::uuid,$5,$6::uuid,'OPEN',$7,$8::uuid,$8::uuid,$9,$9,1)
		RETURNING id::text,condition_key,member_id::text,subject_type,subject_id::text,last_condition_state,notice_sequence`,
		source.TenantID, source.LegalEntityID, member.Condition, member.MemberID,
		member.SubjectType, member.SubjectID, member.ConditionState, source.ID, source.GeneratedAt,
	).Scan(&episode.ID, &episode.Condition, &episode.MemberID, &episode.SubjectType, &episode.SubjectID, &episode.ConditionState, &episode.NoticeSequence)
	if err != nil {
		return fmt.Errorf("open attention episode: %w", err)
	}
	return p.emitEpisodeIntents(ctx, tx, source, episode, EventEpisodeOpened)
}

func (p *EpisodeProjector) clearEpisode(ctx context.Context, tx pgx.Tx, source sourceSnapshot, episode openEpisode) error {
	sequence := episode.NoticeSequence + 1
	tag, err := tx.Exec(ctx, `
		UPDATE attention_episodes
		SET state='CLEARED',last_source_id=$4::uuid,last_observed_at=$5,cleared_at=$5,
		    notice_sequence=$6,updated_at=$5
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND state='OPEN'`,
		source.TenantID, source.LegalEntityID, episode.ID, source.ID, source.GeneratedAt, sequence)
	if err != nil {
		return fmt.Errorf("clear attention episode: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return nil
	}
	episode.NoticeSequence = sequence
	return p.emitEpisodeIntents(ctx, tx, source, episode, EventEpisodeCleared)
}

func (p *EpisodeProjector) emitEpisodeIntents(
	ctx context.Context,
	tx pgx.Tx,
	source sourceSnapshot,
	episode openEpisode,
	eventType string,
) error {
	recipients, err := episodeRecipients(ctx, tx, source, episode)
	if err != nil {
		return err
	}
	for _, principalID := range recipients {
		payload, err := json.Marshal(Intent{
			EpisodeID: episode.ID, LegalEntityID: source.LegalEntityID, PrincipalID: principalID,
			Condition: episode.Condition, ConditionState: episode.ConditionState,
			SubjectType: episode.SubjectType, SubjectID: episode.SubjectID, SourceID: source.ID,
			NoticeSequence: episode.NoticeSequence,
		})
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO outbox_events(
				tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at
			) VALUES($1::uuid,$2,$3::uuid,$4,$5::jsonb,$6,$6,$6)`,
			source.TenantID, EpisodeAggregateType, episode.ID, eventType, payload, source.GeneratedAt); err != nil {
			return fmt.Errorf("emit attention episode intent: %w", err)
		}
	}
	return nil
}

func episodeRecipients(ctx context.Context, tx pgx.Tx, source sourceSnapshot, episode openEpisode) ([]string, error) {
	rows, err := tx.Query(ctx, `
		WITH candidates(principal_id) AS (
			SELECT risk.owner_principal_id
			FROM risks risk
			WHERE $3='RISK'
			  AND risk.tenant_id=$1::uuid
			  AND risk.legal_entity_id=$2::uuid
			  AND risk.id=$4::uuid

			UNION
			SELECT check_config.owner_principal_id
			FROM risk_indicator_links link
			JOIN monitoring_checks check_config
			  ON check_config.tenant_id=link.tenant_id
			 AND check_config.id=link.monitoring_check_id
			 AND check_config.version=link.monitoring_check_version
			 AND check_config.program_id=link.program_id
			WHERE $5='indicator_breaches'
			  AND link.tenant_id=$1::uuid
			  AND link.legal_entity_id=$2::uuid
			  AND link.id=$6::uuid

			UNION
			SELECT check_config.reviewer_principal_id
			FROM risk_indicator_links link
			JOIN monitoring_checks check_config
			  ON check_config.tenant_id=link.tenant_id
			 AND check_config.id=link.monitoring_check_id
			 AND check_config.version=link.monitoring_check_version
			 AND check_config.program_id=link.program_id
			WHERE $5='indicator_breaches'
			  AND link.tenant_id=$1::uuid
			  AND link.legal_entity_id=$2::uuid
			  AND link.id=$6::uuid

			UNION
			SELECT loss.owner_principal_id
			FROM operational_losses loss
			WHERE $3='LOSS'
			  AND loss.tenant_id=$1::uuid
			  AND loss.legal_entity_id=$2::uuid
			  AND loss.id=$4::uuid
		)
		SELECT DISTINCT candidate.principal_id::text
		FROM candidates candidate
		JOIN principals principal
		  ON principal.tenant_id=$1::uuid
		 AND principal.id=candidate.principal_id
		 AND principal.status='ACTIVE'
		 AND principal.valid_from<=$7
		 AND (principal.valid_until IS NULL OR $7<principal.valid_until)
		WHERE candidate.principal_id IS NOT NULL
		ORDER BY candidate.principal_id::text`,
		source.TenantID, source.LegalEntityID, episode.SubjectType, episode.SubjectID,
		episode.Condition, episode.MemberID, source.GeneratedAt)
	if err != nil {
		return nil, fmt.Errorf("resolve attention recipients: %w", err)
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var principalID string
		if err := rows.Scan(&principalID); err != nil {
			return nil, err
		}
		result = append(result, strings.TrimSpace(principalID))
	}
	return result, rows.Err()
}

func conditionStateRank(value string) int {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "CRITICAL":
		return 3
	case "HIGH":
		return 2
	case "BREACHED", "FAILED", "WITHOUT_ISSUE":
		return 1
	default:
		return 0
	}
}

func episodeMemberKey(condition, memberID string) string {
	return strings.TrimSpace(condition) + "\x00" + strings.TrimSpace(memberID)
}

var _ workflowruntime.Publisher = (*EpisodeProjector)(nil)
