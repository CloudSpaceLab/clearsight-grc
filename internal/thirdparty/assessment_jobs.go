package thirdparty

import (
	"context"
	"errors"
	"time"
)

type AssessmentJobState string

const (
	AssessmentJobReady     AssessmentJobState = "READY"
	AssessmentJobLeased    AssessmentJobState = "LEASED"
	AssessmentJobCompleted AssessmentJobState = "COMPLETED"
	AssessmentJobFailed    AssessmentJobState = "FAILED"

	AssessmentSetupJobType = "SETUP_REVIEW"

	AssessmentSetupFailureRead              = "ASSESSMENT_READ_FAILED"
	AssessmentSetupFailureRelationship      = "RELATIONSHIP_READ_FAILED"
	AssessmentSetupFailureMatter            = "MATTER_CREATE_FAILED"
	AssessmentSetupFailureCompletion        = "ASSESSMENT_SETUP_FAILED"
	AssessmentSetupFailureAuthority         = "AUTHORITY_ROUTE_UNAVAILABLE"
	AssessmentSetupFailureAttemptsExhausted = "ATTEMPTS_EXHAUSTED"
	AssessmentSetupRetryCommand             = "thirdparty.assessment.setup.retry"
)

var ErrAssessmentJobLeaseLost = errors.New("third-party assessment job lease is no longer current")

var ErrAssessmentSetupStatusUnavailable = errors.New("third-party assessment setup status is unavailable")

type AssessmentSetupJob struct {
	ID              string             `json:"id"`
	TenantID        string             `json:"tenant_id"`
	LegalEntityID   string             `json:"legal_entity_id"`
	AssessmentID    string             `json:"assessment_id"`
	JobType         string             `json:"job_type"`
	DedupeKey       string             `json:"dedupe_key"`
	State           AssessmentJobState `json:"state"`
	Attempts        int                `json:"attempts"`
	AvailableAt     time.Time          `json:"available_at"`
	LeaseToken      string             `json:"lease_token,omitempty"`
	LeaseExpiresAt  *time.Time         `json:"lease_expires_at,omitempty"`
	LastFailureCode string             `json:"last_failure_code,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type AssessmentSetupStatus struct {
	AssessmentID  string             `json:"assessment_id"`
	State         AssessmentJobState `json:"state"`
	Attempts      int                `json:"attempts"`
	NextAttemptAt *time.Time         `json:"next_attempt_at,omitempty"`
	LeaseUntil    *time.Time         `json:"lease_until,omitempty"`
	TerminalAt    *time.Time         `json:"terminal_at,omitempty"`
	FailureCode   string             `json:"failure_code,omitempty"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

type RetryAssessmentSetupInput struct {
	ExpectedVersion int64 `json:"expected_version"`
}

type AssessmentSetupRetryOutcome struct {
	Assessment Assessment            `json:"assessment"`
	Setup      AssessmentSetupStatus `json:"setup"`
}

type AssessmentSetupRepository interface {
	GetAssessment(context.Context, Scope, string) (Assessment, error)
	GetRelationship(context.Context, Scope, string) (Aggregate, error)
	ClaimAssessmentSetupJobs(context.Context, string, time.Time, time.Duration, int, int) ([]AssessmentSetupJob, error)
	CompleteAssessmentSetupJob(context.Context, AssessmentSetupJob, int64, string, time.Time) (Assessment, error)
	FailAssessmentSetupJob(context.Context, AssessmentSetupJob, int, string, time.Time, time.Time) (AssessmentSetupJob, error)
}

type assessmentSetupStatusRepository interface {
	GetAssessmentSetupJob(context.Context, Scope, string) (AssessmentSetupJob, error)
}
