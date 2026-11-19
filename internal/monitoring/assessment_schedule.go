package monitoring

import (
	"context"
	"fmt"
	"strings"
	"time"

	_ "time/tzdata"
)

const AssessmentScheduleWorkClass = "program-assessment-schedules"

type AssessmentScheduleInput struct {
	AdoptionID            string `json:"adoption_id"`
	FormTemplateID        string `json:"form_template_id"`
	FormTemplateVersion   int64  `json:"form_template_version"`
	RespondentPrincipalID string `json:"respondent_principal_id"`
	FirstPeriodStart      string `json:"first_period_start"`
	TimeZone              string `json:"time_zone"`
	IntervalMonths        int    `json:"interval_months"`
	IntervalDays          int    `json:"interval_days"`
	ResponseDays          int    `json:"response_days"`
	CycleLimit            int    `json:"cycle_limit"`
	Rationale             string `json:"rationale"`
}

type AssessmentSchedule struct {
	AssessmentScheduleInput
	ID            string    `json:"id"`
	ProgramID     string    `json:"program_id"`
	OwnerID       string    `json:"owner_id"`
	ApproverID    string    `json:"approver_id,omitempty"`
	State         string    `json:"state"`
	Version       int64     `json:"version"`
	CycleIndex    int       `json:"cycle_index"`
	NextAttemptAt time.Time `json:"next_attempt_at"`
	LastOutcome   string    `json:"last_outcome,omitempty"`
	LastRequestID string    `json:"last_request_id,omitempty"`
}

type assessmentScheduleRepository interface {
	ListAssessmentSchedules(context.Context, Actor, string) ([]AssessmentSchedule, error)
	InsertAssessmentSchedule(context.Context, Actor, string, AssessmentScheduleInput, time.Time) (AssessmentSchedule, error)
	ChangeAssessmentSchedule(context.Context, Actor, string, string, string, string, int64, time.Time) (AssessmentSchedule, error)
}

func (input AssessmentScheduleInput) recurrence() (AssessmentRecurrence, error) {
	if input.TimeZone == "" || input.TimeZone == "Local" {
		return AssessmentRecurrence{}, ErrInvalid
	}
	location, err := time.LoadLocation(input.TimeZone)
	if err != nil {
		return AssessmentRecurrence{}, ErrInvalid
	}
	start, err := time.ParseInLocation(time.DateOnly, input.FirstPeriodStart, location)
	if err != nil {
		return AssessmentRecurrence{}, ErrInvalid
	}
	return AssessmentRecurrence{FirstPeriodStart: start, IntervalMonths: input.IntervalMonths, IntervalDays: input.IntervalDays, ResponseDays: input.ResponseDays}, nil
}

func (s *Service) ListAssessmentSchedules(ctx context.Context, actor Actor, program string) ([]AssessmentSchedule, error) {
	repo, ok := s.repo.(assessmentScheduleRepository)
	if !ok {
		return nil, ErrInactive
	}
	return repo.ListAssessmentSchedules(ctx, actor, program)
}

func (s *Service) ProposeAssessmentSchedule(ctx context.Context, actor Actor, program string, input AssessmentScheduleInput) (AssessmentSchedule, error) {
	repo, ok := s.repo.(assessmentScheduleRepository)
	if !ok {
		return AssessmentSchedule{}, ErrInactive
	}
	input.Rationale = strings.TrimSpace(input.Rationale)
	if input.CycleLimit < 1 || input.CycleLimit > 12 || input.Rationale == "" || len(input.Rationale) > 2000 {
		return AssessmentSchedule{}, ErrInvalid
	}
	r, err := input.recurrence()
	if err != nil {
		return AssessmentSchedule{}, err
	}
	first, err := r.Occurrence(0)
	if err != nil || !first.Deadline.After(s.now()) {
		return AssessmentSchedule{}, fmt.Errorf("%w: choose a reporting period whose response deadline is in the future", ErrInvalid)
	}
	if _, err = r.Occurrence(input.CycleLimit - 1); err != nil {
		return AssessmentSchedule{}, err
	}
	if _, err = s.assessmentScheduleOption(ctx, actor, program, input); err != nil {
		return AssessmentSchedule{}, err
	}
	return repo.InsertAssessmentSchedule(ctx, actor, program, input, first.OpensAt)
}

func (s *Service) ChangeAssessmentSchedule(ctx context.Context, actor Actor, program, id, state, reason string, version int64) (AssessmentSchedule, error) {
	repo, ok := s.repo.(assessmentScheduleRepository)
	if !ok {
		return AssessmentSchedule{}, ErrInactive
	}
	reason = strings.TrimSpace(reason)
	if !formLinkUUID.MatchString(id) || (state != "ACTIVE" && state != "STOPPED") || reason == "" || len(reason) > 2000 || version < 1 {
		return AssessmentSchedule{}, ErrInvalid
	}
	if state == "ACTIVE" {
		schedules, err := repo.ListAssessmentSchedules(ctx, actor, program)
		if err != nil {
			return AssessmentSchedule{}, err
		}
		found := false
		for _, schedule := range schedules {
			if schedule.ID != id {
				continue
			}
			if schedule.State != "PENDING" || schedule.Version != version || schedule.OwnerID == actor.PrincipalID {
				return AssessmentSchedule{}, ErrConflict
			}
			recurrence, err := schedule.recurrence()
			if err != nil {
				return AssessmentSchedule{}, err
			}
			first, err := recurrence.Occurrence(0)
			if err != nil || !first.Deadline.After(s.now()) {
				return AssessmentSchedule{}, fmt.Errorf("%w: the first response window has closed", ErrConflict)
			}
			owner := actor
			owner.PrincipalID = schedule.OwnerID
			if _, err = s.assessmentScheduleOption(ctx, owner, program, schedule.AssessmentScheduleInput); err != nil {
				return AssessmentSchedule{}, err
			}
			found = true
			break
		}
		if !found {
			return AssessmentSchedule{}, ErrNotFound
		}
	}
	return repo.ChangeAssessmentSchedule(ctx, actor, program, id, state, reason, version, s.now())
}

func (s *Service) assessmentScheduleOption(ctx context.Context, actor Actor, program string, input AssessmentScheduleInput) (AdoptedAssessmentOption, error) {
	options, err := s.AdoptedAssessmentOptions(ctx, actor, program)
	if err != nil {
		return AdoptedAssessmentOption{}, err
	}
	for _, option := range options {
		if !option.Ready || option.AdoptionID != input.AdoptionID || option.FormID != input.FormTemplateID || option.FormVersion != input.FormTemplateVersion {
			continue
		}
		for _, person := range option.Respondents {
			if person.ID == input.RespondentPrincipalID && person.ID != option.ReviewerID {
				return option, nil
			}
		}
	}
	return AdoptedAssessmentOption{}, fmt.Errorf("%w: choose an approved questionnaire and an eligible respondent", ErrConflict)
}

// FindAssessmentSchedule reads one schedule without the register's population bound.
func (s *Service) FindAssessmentSchedule(ctx context.Context, actor Actor, program, id string) (AssessmentSchedule, error) {
	repo, ok := s.repo.(interface {
		FindAssessmentSchedule(context.Context, Actor, string, string) (AssessmentSchedule, error)
	})
	if !ok {
		return AssessmentSchedule{}, ErrInactive
	}
	return repo.FindAssessmentSchedule(ctx, actor, program, id)
}
