package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

const (
	AssessmentSetupWorkClass     = "third-party-assessment-setup"
	defaultAssessmentJobLease    = 2 * time.Minute
	defaultAssessmentJobAttempts = 5
	defaultAssessmentJobBackoff  = 5 * time.Minute
)

type AssessmentMatterService interface {
	CreateMatter(context.Context, continuity.CreateMatterInput) (continuity.MatterAggregate, error)
	MatterByTriggerKey(context.Context, string, string) (continuity.MatterAggregate, error)
}

type AssessmentProvisioner struct {
	repository  AssessmentSetupRepository
	matters     AssessmentMatterService
	workerID    string
	lease       time.Duration
	maxAttempts int
	maxBackoff  time.Duration
}

func NewAssessmentProvisioner(repository AssessmentSetupRepository, matters AssessmentMatterService, workerID string) *AssessmentProvisioner {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		workerID = "assessment-setup-worker"
	}
	return &AssessmentProvisioner{
		repository: repository, matters: matters, workerID: workerID,
		lease: defaultAssessmentJobLease, maxAttempts: defaultAssessmentJobAttempts, maxBackoff: defaultAssessmentJobBackoff,
	}
}

func (p *AssessmentProvisioner) Configure(lease time.Duration, maxAttempts int, maxBackoff time.Duration) {
	if lease > 0 {
		p.lease = lease
	}
	if maxAttempts > 0 && maxAttempts <= 20 {
		p.maxAttempts = maxAttempts
	}
	if maxBackoff > 0 {
		p.maxBackoff = maxBackoff
	}
}

func (p *AssessmentProvisioner) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if p == nil || p.repository == nil || p.matters == nil {
		return 0, errors.New("third-party assessment setup is not configured")
	}
	limit = boundedAssessmentJobLimit(limit)
	now = now.UTC()
	jobs, err := p.repository.ClaimAssessmentSetupJobs(ctx, p.workerID, now, p.lease, p.maxAttempts, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	failures := make([]error, 0)
	for _, job := range jobs {
		if ctx.Err() != nil {
			return processed, errors.Join(append(failures, ctx.Err())...)
		}
		processed++
		if err := p.provision(ctx, job, now); err != nil {
			failures = append(failures, fmt.Errorf("provision assessment setup job %s: %w", job.ID, err))
		}
	}
	return processed, errors.Join(failures...)
}

func (p *AssessmentProvisioner) provision(ctx context.Context, job AssessmentSetupJob, now time.Time) error {
	scope := Scope{TenantID: job.TenantID, LegalEntityID: job.LegalEntityID}
	assessment, err := p.repository.GetAssessment(ctx, scope, job.AssessmentID)
	if err != nil {
		return p.release(ctx, job, AssessmentSetupFailureRead, now, err)
	}
	relationship, err := p.repository.GetRelationship(ctx, scope, assessment.RelationshipID)
	if err != nil {
		return p.release(ctx, job, AssessmentSetupFailureRelationship, now, err)
	}
	triggerKey := "thirdparty-assessment:" + assessment.ID
	matter, err := p.matters.CreateMatter(ctx, assessmentMatterInput(assessment, relationship, triggerKey))
	if errors.Is(err, continuity.ErrDuplicate) {
		matter, err = p.matters.MatterByTriggerKey(ctx, assessment.TenantID, triggerKey)
	}
	if err != nil {
		return p.release(ctx, job, AssessmentSetupFailureMatter, now, err)
	}
	if matter.Matter.ID == "" || matter.Matter.TriggerKey != triggerKey || matter.Matter.Type != continuity.MatterVendorReview {
		return p.release(ctx, job, AssessmentSetupFailureMatter, now, errors.New("canonical review matter did not match the assessment trigger"))
	}
	if _, err := p.repository.CompleteAssessmentSetupJob(ctx, job, assessment.Version, matter.Matter.ID, now); err != nil {
		return p.release(ctx, job, AssessmentSetupFailureCompletion, now, err)
	}
	return nil
}

func (p *AssessmentProvisioner) release(ctx context.Context, job AssessmentSetupJob, code string, now time.Time, cause error) error {
	next := now.Add(assessmentJobRetryDelay(job.Attempts, p.maxBackoff))
	if _, err := p.repository.FailAssessmentSetupJob(ctx, job, p.maxAttempts, code, now, next); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

func assessmentMatterInput(assessment Assessment, aggregate Aggregate, triggerKey string) continuity.CreateMatterInput {
	priority := 3
	switch aggregate.Relationship.Criticality {
	case CriticalityImportant:
		priority = 4
	case CriticalityCritical:
		priority = 5
	}
	vendorName := strings.TrimSpace(aggregate.Vendor.LegalName)
	serviceName := strings.TrimSpace(aggregate.Relationship.ServiceName)
	scope, _ := json.Marshal(map[string]any{
		"assessment_id": assessment.ID, "relationship_id": assessment.RelationshipID,
		"vendor_id": aggregate.Vendor.ID, "legal_entity_id": assessment.LegalEntityID,
		"review_kind":           assessment.ReviewKind,
		"access":                continuity.MatterAccessRestricted,
		"allowed_principal_ids": uniqueAssessmentPrincipals(aggregate.Relationship.BusinessOwnerPrincipalID, assessment.StartedByPrincipalID),
	})
	known, _ := json.Marshal(map[string]any{
		"vendor_legal_name": vendorName, "service_name": serviceName,
		"criticality": aggregate.Relationship.Criticality, "privacy_role": aggregate.Relationship.PrivacyRole,
		"review_due_at": assessment.ReviewDueAt,
	})
	missing := json.RawMessage(`["vendor response","supporting evidence","review conclusion"]`)
	dueAt := assessment.ReviewDueAt
	return continuity.CreateMatterInput{
		TenantID: assessment.TenantID, Type: continuity.MatterVendorReview, Priority: priority,
		Title:   boundedAssessmentMatterTitle("Review " + vendorName + " for " + serviceName),
		Summary: "Review the vendor response and supporting evidence before this service relationship is approved.",
		Scope:   scope, SourceType: "THIRD_PARTY_ASSESSMENT", SourceID: assessment.ID,
		TriggerType: "VENDOR_DUE_DILIGENCE_STARTED", TriggerID: assessment.ID, TriggerKey: triggerKey,
		KnownFacts: known, MissingFacts: missing, Contradictions: json.RawMessage(`[]`),
		OwnerPrincipalID: aggregate.Relationship.BusinessOwnerPrincipalID, RequiredAuthority: "REVIEWER", DueAt: &dueAt,
	}
}

func uniqueAssessmentPrincipals(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func boundedAssessmentMatterTitle(value string) string {
	const maximum = 200
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > maximum {
		runes = runes[:maximum]
	}
	return string(runes)
}

func assessmentJobRetryDelay(attempt int, maximum time.Duration) time.Duration {
	if maximum <= 0 {
		maximum = defaultAssessmentJobBackoff
	}
	delay := time.Second
	for index := 1; index < attempt && delay < maximum; index++ {
		delay *= 2
		if delay <= 0 || delay >= maximum {
			return maximum
		}
	}
	if delay > maximum {
		return maximum
	}
	return delay
}

func boundedAssessmentJobLimit(limit int) int {
	if limit < 1 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}
