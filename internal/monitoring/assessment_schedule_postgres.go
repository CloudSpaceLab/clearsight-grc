//go:build postgres

package monitoring

import (
	"context"

	"encoding/json"
	"errors"
	"fmt"

	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const scheduleColumns = `s.id::text,s.program_id::text,s.owner_id::text,coalesce(s.approver_id::text,''),s.state,s.version,
 s.cycle_index,s.next_attempt_at,s.last_outcome,coalesce(s.last_request_id::text,''),s.form_selection,
 s.first_period_start::text,s.time_zone,s.interval_months,s.interval_days,s.response_days,s.cycle_limit,s.rationale`

func scanAssessmentSchedule(row pgx.Row) (AssessmentSchedule, error) {
	var result AssessmentSchedule
	var selection []byte
	err := row.Scan(&result.ID, &result.ProgramID, &result.OwnerID, &result.ApproverID, &result.State, &result.Version,
		&result.CycleIndex, &result.NextAttemptAt, &result.LastOutcome, &result.LastRequestID, &selection,
		&result.FirstPeriodStart, &result.TimeZone, &result.IntervalMonths, &result.IntervalDays, &result.ResponseDays, &result.CycleLimit, &result.Rationale)
	if err != nil {
		return result, mapPostgresError(err)
	}
	return result, json.Unmarshal(selection, &result.AssessmentScheduleInput)
}

func (r *PostgresRepository) ListAssessmentSchedules(ctx context.Context, actor Actor, program string) ([]AssessmentSchedule, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+scheduleColumns+` FROM program_assessment_schedules s
 JOIN programs p ON p.id=s.program_id
 WHERE s.tenant_id=$1::uuid AND s.legal_entity_id=$2::uuid AND s.program_id=$3::uuid
 AND $4::uuid IN (p.owner_principal_id,p.authority_principal_id) ORDER BY s.created_at DESC,s.id DESC LIMIT 100`,
		actor.TenantID, actor.LegalEntityID, program, actor.PrincipalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AssessmentSchedule{}
	for rows.Next() {
		s, e := scanAssessmentSchedule(rows)
		if e != nil {
			return nil, e
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) InsertAssessmentSchedule(ctx context.Context, actor Actor, program string, input AssessmentScheduleInput, opens time.Time) (AssessmentSchedule, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AssessmentSchedule{}, err
	}
	defer tx.Rollback(ctx)
	selection, err := json.Marshal(map[string]any{
		"adoption_id": input.AdoptionID, "form_template_id": input.FormTemplateID,
		"form_template_version": input.FormTemplateVersion, "respondent_principal_id": input.RespondentPrincipalID,
	})
	if err != nil {
		return AssessmentSchedule{}, err
	}
	s, err := scanAssessmentSchedule(tx.QueryRow(ctx, `INSERT INTO program_assessment_schedules AS s
 (tenant_id,legal_entity_id,program_id,owner_id,form_selection,first_period_start,time_zone,interval_months,interval_days,response_days,cycle_limit,next_attempt_at,rationale)
 SELECT tenant_id,legal_entity_id,id,owner_principal_id,$5::jsonb,$6::date,$7,$8,$9,$10,$11,$12,$13 FROM programs
 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND owner_principal_id=$4::uuid AND status='ACTIVE'
 RETURNING `+scheduleColumns, actor.TenantID, actor.LegalEntityID, program, actor.PrincipalID, selection, input.FirstPeriodStart, input.TimeZone, input.IntervalMonths, input.IntervalDays, input.ResponseDays, input.CycleLimit, opens, input.Rationale))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "program_assessment_schedule_current" {
			return AssessmentSchedule{}, fmt.Errorf("%w: a schedule already exists for this adoption, form and respondent", ErrConflict)
		}
		return AssessmentSchedule{}, err
	}
	if err = assessmentScheduleEvent(ctx, tx, actor, s, "ProgramAssessmentScheduleProposed", input.Rationale); err != nil {
		return AssessmentSchedule{}, err
	}
	return s, tx.Commit(ctx)
}

func (r *PostgresRepository) ChangeAssessmentSchedule(ctx context.Context, actor Actor, program, id, state, reason string, version int64, now time.Time) (AssessmentSchedule, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AssessmentSchedule{}, err
	}
	defer tx.Rollback(ctx)
	s, err := scanAssessmentSchedule(tx.QueryRow(ctx, `UPDATE program_assessment_schedules s SET state=$6,version=s.version+1,
 approver_id=CASE WHEN $6='ACTIVE' THEN $4::uuid ELSE s.approver_id END,last_outcome=''
 FROM programs p WHERE p.id=s.program_id AND s.tenant_id=$1::uuid AND s.legal_entity_id=$2::uuid AND s.program_id=$3::uuid AND s.id=$5::uuid
 AND s.version=$7 AND (
 ($6='ACTIVE' AND s.state='PENDING' AND p.status='ACTIVE' AND p.owner_principal_id=s.owner_id
  AND p.authority_principal_id=$4::uuid AND s.owner_id<>$4::uuid )
 OR ($6='STOPPED' AND s.state IN ('PENDING','ACTIVE','BLOCKED') AND $4::uuid IN (p.owner_principal_id,p.authority_principal_id)))
 RETURNING `+scheduleColumns, actor.TenantID, actor.LegalEntityID, program, actor.PrincipalID, id, state, version))
	if errors.Is(err, ErrNotFound) {
		return AssessmentSchedule{}, fmt.Errorf("%w: the schedule changed or cannot be acted on by this account", ErrConflict)
	}
	if err != nil {
		return AssessmentSchedule{}, err
	}
	event := "ProgramAssessmentScheduleStopped"
	if state == "ACTIVE" {
		event = "ProgramAssessmentScheduleApproved"
	}
	if state == "ACTIVE" {
		recurrence, recurrenceErr := s.recurrence()
		if recurrenceErr != nil {
			return AssessmentSchedule{}, recurrenceErr
		}
		last, recurrenceErr := recurrence.Occurrence(s.CycleLimit - 1)
		if recurrenceErr != nil {
			return AssessmentSchedule{}, recurrenceErr
		}
		_, err = tx.Exec(ctx, `INSERT INTO automation_policies(id,tenant_id,code,name,action_class,eligibility,
 blast_radius_limit,verification_contract,status,rollout_mode,maker_id,checker_id,submitted_at,approved_at,activated_at,effective_from,effective_until)
 SELECT id,tenant_id,'ASSESSMENT-'||id::text,'Recurring Program assessment','PROGRAM_ASSESSMENT_DISPATCH',
 form_selection||jsonb_build_object('program_id',program_id,'legal_entity_id',legal_entity_id),
 jsonb_build_object('cycles',cycle_limit,'requests_per_cycle',1),
 '{"outcome":"assigned_request","dedupe":"schedule_cycle_respondent","compensation":"stop_future_dispatch"}'::jsonb,
 'ACTIVE','ENFORCE',owner_id,approver_id,created_at,$2,$2,$2,$3 FROM program_assessment_schedules WHERE id=$1::uuid`, s.ID, now, last.Deadline)
	} else {
		_, err = tx.Exec(ctx, `UPDATE automation_policies SET status='SUSPENDED',suspended_at=$2,updated_at=$2,record_version=record_version+1
 WHERE id=$1::uuid AND action_class='PROGRAM_ASSESSMENT_DISPATCH' AND status='ACTIVE'`, s.ID, now)
	}
	if err != nil {
		return AssessmentSchedule{}, err
	}
	if err = assessmentScheduleEvent(ctx, tx, actor, s, event, reason); err != nil {
		return AssessmentSchedule{}, err
	}
	return s, tx.Commit(ctx)
}

func assessmentScheduleEvent(ctx context.Context, tx pgx.Tx, actor Actor, s AssessmentSchedule, event, reason string) error {
	_, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
 VALUES($1::uuid,'PROGRAM_ASSESSMENT_SCHEDULE',$2::uuid,$3,jsonb_build_object('program_id',$4::text,'actor_id',$5::text,
 'version',$6::bigint,'state',$7::text,'reason',$8::text,'cycle_index',$9::integer,'request_id',$10::text),now(),now(),now())`,
		actor.TenantID, s.ID, event, s.ProgramID, actor.PrincipalID, s.Version, s.State, reason, s.CycleIndex, s.LastRequestID)
	return err
}

// AssessmentScheduleMaintainer dispatches one assigned assessment per due cycle.
type AssessmentScheduleMaintainer struct {
	Repository *PostgresRepository
	Service    *Service
	Requests   *evidence.Service
	Authority  authority.Service
}

func (m *AssessmentScheduleMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	for i := 0; i < limit; i++ {
		found, err := m.dispatchNext(ctx, now)
		if err != nil || !found {
			return i, err
		}
	}
	return limit, nil
}

func (m *AssessmentScheduleMaintainer) dispatchNext(ctx context.Context, now time.Time) (bool, error) {
	tx, err := m.Repository.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var id, tenant, entity, owner string
	var attempts int
	err = tx.QueryRow(ctx, `SELECT id::text,tenant_id::text,legal_entity_id::text,owner_id::text,attempts
 FROM program_assessment_schedules WHERE state='ACTIVE' AND next_attempt_at<=$1
 ORDER BY next_attempt_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`, now).Scan(&id, &tenant, &entity, &owner, &attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	s, err := scanAssessmentSchedule(tx.QueryRow(ctx, `SELECT `+scheduleColumns+` FROM program_assessment_schedules s WHERE id=$1::uuid`, id))
	if err != nil {
		return false, err
	}
	actor := Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner}
	var policyActive bool
	err = tx.QueryRow(ctx, `SELECT status='ACTIVE' AND rollout_mode='ENFORCE' AND checker_id=$2::uuid AND maker_id=$3::uuid
 AND effective_from<=$4 AND eligibility=(SELECT form_selection||jsonb_build_object('program_id',program_id,'legal_entity_id',legal_entity_id) FROM program_assessment_schedules WHERE id=$1::uuid)
 FROM automation_policies WHERE id=$1::uuid AND tenant_id=$5::uuid AND action_class='PROGRAM_ASSESSMENT_DISPATCH' FOR SHARE`,
		s.ID, s.ApproverID, s.OwnerID, now, tenant).Scan(&policyActive)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	var outcome string
	var nextErr error
	if !policyActive {
		nextErr = ErrConflict
	} else {
		outcome, nextErr = m.dispatchOccurrence(ctx, actor, &s, now)
	}
	if nextErr != nil {
		attempts++
		s.NextAttemptAt = now.Add(time.Duration(attempts) * time.Minute)
		if attempts >= 5 || errors.Is(nextErr, ErrConflict) || errors.Is(nextErr, ErrInvalid) || errors.Is(nextErr, ErrNotFound) {
			s.State = "BLOCKED"
			outcome = "Dispatch blocked. Check the Program owner, approver, adopted form, and respondent before replacing this schedule."
		} else {
			outcome = "Dispatch could not complete. The worker will retry."
		}
	} else {
		attempts = 0
	}
	s.Version++
	s.LastOutcome = outcome
	_, err = tx.Exec(ctx, `UPDATE program_assessment_schedules SET cycle_index=$2,next_attempt_at=$3,state=$4,attempts=$5,
 last_outcome=$6,last_request_id=nullif($7,'')::uuid,version=$8 WHERE id=$1::uuid`,
		id, s.CycleIndex, s.NextAttemptAt, s.State, attempts, outcome, s.LastRequestID, s.Version)
	if err != nil {
		return false, err
	}
	if err = assessmentScheduleEvent(ctx, tx, actor, s, "ProgramAssessmentScheduleCycleProcessed", outcome); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (m *AssessmentScheduleMaintainer) dispatchOccurrence(ctx context.Context, actor Actor, s *AssessmentSchedule, now time.Time) (string, error) {
	recurrence, err := s.recurrence()
	if err != nil {
		return "", err
	}
	cycle, err := recurrence.Occurrence(s.CycleIndex)
	if err != nil {
		return "", err
	}
	key := cycle.AssessmentKey(s.ID, s.RespondentPrincipalID)
	origin := adoptedAssessmentOrigin(s.AdoptionID, key, s.RespondentPrincipalID)
	request, err := m.Requests.GetRequestByOrigin(ctx, actor.TenantID, origin)
	if err != nil && !errors.Is(err, evidence.ErrNotFound) {
		return "", err
	}
	outcome := "Assessment assigned."
	if errors.Is(err, evidence.ErrNotFound) {
		if !now.Before(cycle.Deadline) {
			outcome = "Assessment skipped because the response window closed."
		} else {
			option, optionErr := m.Service.assessmentScheduleOption(ctx, actor, s.ProgramID, s.AssessmentScheduleInput)
			if optionErr != nil {
				return "", optionErr
			}
			if m.Authority == nil {
				return "", ErrInactive
			}
			for _, route := range []struct {
				role        authority.Responsibility
				decision    string
				principal   string
				materiality int
			}{
				{authority.ResponsibilityOwner, "program.assessment.create", actor.PrincipalID, 2},
				{authority.ResponsibilityAuthorizer, "program.assessment.schedule.approve", s.ApproverID, 3},
			} {
				resolved, resolveErr := m.Authority.Resolve(ctx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "PROGRAM", ObjectID: s.ProgramID, Responsibility: route.role, DecisionType: route.decision, Materiality: route.materiality, At: now})
				if resolveErr != nil {
					if errors.Is(resolveErr, authority.ErrNoRoute) || errors.Is(resolveErr, authority.ErrAmbiguousRoute) {
						return "", ErrConflict
					}
					return "", resolveErr
				}
				if !resolved.AllowsPrincipal(route.principal) {
					return "", ErrConflict
				}
			}
			if option.ReviewerID != s.ApproverID {
				return "", ErrConflict
			}
			request, err = m.Service.StartCollection(ctx, actor, StartCollectionInput{
				AssessmentScheduleID: s.ID,
				AdoptionID:           s.AdoptionID, AssessmentKey: key, ExpectedProgramVersion: option.ProgramVersion,
				FormTemplateID: s.FormTemplateID, FormTemplateVersion: s.FormTemplateVersion,
				ProgramID: s.ProgramID, LegalEntityID: actor.LegalEntityID, RespondentPrincipalID: s.RespondentPrincipalID,
				PeriodStart: cycle.PeriodStart, PeriodEnd: cycle.PeriodEnd, Deadline: cycle.Deadline,
			})
			if err != nil {
				return "", err
			}
		}
	}
	if request.ID != "" {
		s.LastRequestID = request.ID
	}
	s.CycleIndex++
	if s.CycleIndex == s.CycleLimit {
		s.State = "COMPLETE"
		return outcome, nil
	}
	next, err := recurrence.Occurrence(s.CycleIndex)
	if err != nil {
		return "", err
	}
	s.NextAttemptAt = next.OpensAt
	return outcome, nil
}

func (r *PostgresRepository) FindAssessmentSchedule(ctx context.Context, actor Actor, program, id string) (AssessmentSchedule, error) {
	return scanAssessmentSchedule(r.pool.QueryRow(ctx, `SELECT `+scheduleColumns+` FROM program_assessment_schedules s JOIN programs p ON p.id=s.program_id AND p.tenant_id=s.tenant_id
 WHERE s.tenant_id=$1::uuid AND s.legal_entity_id=$2::uuid AND s.program_id=$3::uuid AND s.id::text=$5 AND $4::uuid IN (p.owner_principal_id,p.authority_principal_id)`, actor.TenantID, actor.LegalEntityID, program, actor.PrincipalID, id))
}
