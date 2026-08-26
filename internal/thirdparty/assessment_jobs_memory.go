package thirdparty

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func newMemoryAssessmentSetupJob(assessment Assessment) (AssessmentSetupJob, error) {
	jobID, err := id.NewUUIDv7()
	if err != nil {
		return AssessmentSetupJob{}, err
	}
	return AssessmentSetupJob{
		ID: jobID, TenantID: assessment.TenantID, LegalEntityID: assessment.LegalEntityID,
		AssessmentID: assessment.ID, JobType: AssessmentSetupJobType,
		DedupeKey: "assessment-setup:" + assessment.ID, State: AssessmentJobReady,
		AvailableAt: assessment.CreatedAt.UTC(), CreatedAt: assessment.CreatedAt.UTC(), UpdatedAt: assessment.CreatedAt.UTC(),
	}, nil
}

func (r *MemoryAssessmentRepository) ClaimAssessmentSetupJobs(_ context.Context, workerID string, now time.Time, lease time.Duration, maxAttempts, limit int) ([]AssessmentSetupJob, error) {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" || lease <= 0 || maxAttempts < 1 || maxAttempts > 20 {
		return nil, ErrInvalid
	}
	limit = boundedAssessmentJobLimit(limit)
	now = now.UTC()
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	candidates := make([]AssessmentSetupJob, 0)
	exhausted := make([]AssessmentSetupJob, 0)
	for _, job := range r.setupJobs {
		claimable := job.State == AssessmentJobReady || (job.State == AssessmentJobLeased && job.LeaseExpiresAt != nil && !job.LeaseExpiresAt.After(now))
		if claimable && job.Attempts >= maxAttempts {
			exhausted = append(exhausted, job)
			continue
		}
		if claimable && !job.AvailableAt.After(now) {
			candidates = append(candidates, job)
		}
	}
	sortAssessmentSetupJobs(exhausted)
	if len(exhausted) > limit {
		exhausted = exhausted[:limit]
	}
	for _, job := range exhausted {
		job.State = AssessmentJobFailed
		job.LeaseToken = ""
		job.LeaseExpiresAt = nil
		job.LastFailureCode = AssessmentSetupFailureAttemptsExhausted
		job.UpdatedAt = now
		r.setupJobs[job.ID] = job
	}
	sortAssessmentSetupJobs(candidates)
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	claimed := make([]AssessmentSetupJob, 0, len(candidates))
	for _, candidate := range candidates {
		leaseToken, err := id.NewUUIDv7()
		if err != nil {
			return nil, err
		}
		expires := now.Add(lease)
		candidate.State = AssessmentJobLeased
		candidate.Attempts++
		candidate.LeaseToken = leaseToken
		candidate.LeaseExpiresAt = &expires
		candidate.UpdatedAt = now
		r.setupJobs[candidate.ID] = candidate
		claimed = append(claimed, candidate)
	}
	return claimed, nil
}

func sortAssessmentSetupJobs(values []AssessmentSetupJob) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].AvailableAt.Equal(values[j].AvailableAt) {
			return values[i].ID < values[j].ID
		}
		return values[i].AvailableAt.Before(values[j].AvailableAt)
	})
}

func (r *MemoryAssessmentRepository) CompleteAssessmentSetupJob(_ context.Context, job AssessmentSetupJob, expectedVersion int64, matterID string, at time.Time) (Assessment, error) {
	at = at.UTC()
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	currentJob, ok := r.setupJobs[job.ID]
	if !ok || currentJob.TenantID != job.TenantID || currentJob.LegalEntityID != job.LegalEntityID || currentJob.AssessmentID != job.AssessmentID {
		return Assessment{}, ErrNotFound
	}
	if !sameAssessmentJobLease(currentJob, job, at) {
		return Assessment{}, ErrAssessmentJobLeaseLost
	}
	current, ok := r.assessments[job.AssessmentID]
	if !ok || current.TenantID != job.TenantID || current.LegalEntityID != job.LegalEntityID {
		return Assessment{}, ErrNotFound
	}
	if current.Version != expectedVersion {
		return Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentSetupPending || !validAssessmentIdentifier(matterID) {
		return Assessment{}, ErrInvalidAssessmentTransition
	}
	causationID := currentJob.ID
	reaction := AssessmentReactionRecord{Scope: Scope{TenantID: job.TenantID, LegalEntityID: job.LegalEntityID}, AssessmentID: current.ID, ExpectedVersion: expectedVersion, Kind: AssessmentReactionSetupCompleted, CausationID: causationID, JobID: currentJob.ID, MatterID: matterID, At: at}
	key := assessmentReactionIndex(reaction)
	if receipt, exists := r.reactions[key]; exists {
		return receipt.assessment, nil
	}
	current.Status = AssessmentReadyToSend
	current.ReviewMatterID = matterID
	current.UpdatedAt = at
	current.Version++
	currentJob.State = AssessmentJobCompleted
	currentJob.LeaseToken = ""
	currentJob.LeaseExpiresAt = nil
	currentJob.LastFailureCode = ""
	currentJob.UpdatedAt = at
	r.assessments[current.ID] = current
	r.setupJobs[currentJob.ID] = currentJob
	r.reactions[key] = assessmentReactionReceipt{record: reaction, assessment: current}
	return current, nil
}

func (r *MemoryAssessmentRepository) FailAssessmentSetupJob(_ context.Context, job AssessmentSetupJob, maxAttempts int, failureCode string, at, availableAt time.Time) (AssessmentSetupJob, error) {
	at, availableAt = at.UTC(), availableAt.UTC()
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	current, ok := r.setupJobs[job.ID]
	if !ok || current.TenantID != job.TenantID || current.LegalEntityID != job.LegalEntityID || current.AssessmentID != job.AssessmentID {
		return AssessmentSetupJob{}, ErrNotFound
	}
	if !sameAssessmentJobLease(current, job, at) {
		return AssessmentSetupJob{}, ErrAssessmentJobLeaseLost
	}
	if maxAttempts < 1 || maxAttempts > 20 || !validAssessmentFailureCode(failureCode) || availableAt.Before(at) {
		return AssessmentSetupJob{}, ErrInvalid
	}
	if current.Attempts >= maxAttempts {
		current.State = AssessmentJobFailed
	} else {
		current.State = AssessmentJobReady
		current.AvailableAt = availableAt
	}
	current.LeaseToken = ""
	current.LeaseExpiresAt = nil
	current.LastFailureCode = failureCode
	current.UpdatedAt = at
	r.setupJobs[current.ID] = current
	return current, nil
}

func (r *MemoryAssessmentRepository) RequeueAssessmentSetup(_ context.Context, record RequeueAssessmentSetupRecord) (AssessmentSetupJob, Assessment, error) {
	if !validAssessmentIdentifiers(record.AssessmentID, record.ActorPrincipalID) || record.ExpectedVersion < 1 || record.QueuedAt.IsZero() {
		return AssessmentSetupJob{}, Assessment{}, ErrInvalid
	}
	record.QueuedAt = record.QueuedAt.UTC()
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	current, ok := r.assessments[record.AssessmentID]
	if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID {
		return AssessmentSetupJob{}, Assessment{}, ErrNotFound
	}
	var job AssessmentSetupJob
	for _, candidate := range r.setupJobs {
		if candidate.TenantID == record.TenantID && candidate.LegalEntityID == record.LegalEntityID && candidate.AssessmentID == record.AssessmentID && candidate.JobType == AssessmentSetupJobType {
			job = candidate
			break
		}
	}
	if job.ID == "" {
		return AssessmentSetupJob{}, Assessment{}, ErrNotFound
	}
	if current.Version != record.ExpectedVersion {
		if current.Version == record.ExpectedVersion+1 && assessmentSetupRetryRecorded(r.assessmentEvents, current.ID, current.Version) {
			return job, current, nil
		}
		return AssessmentSetupJob{}, Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentSetupPending || job.State != AssessmentJobFailed || job.LeaseToken != "" || job.LeaseExpiresAt != nil || !validAssessmentFailureCode(job.LastFailureCode) {
		return AssessmentSetupJob{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	previousFailureCode := job.LastFailureCode
	job.State = AssessmentJobReady
	job.Attempts = 0
	job.AvailableAt = record.QueuedAt
	job.LeaseToken = ""
	job.LeaseExpiresAt = nil
	job.LastFailureCode = ""
	job.UpdatedAt = record.QueuedAt
	current.Version++
	current.UpdatedAt = record.QueuedAt
	r.setupJobs[job.ID] = job
	r.assessments[current.ID] = current
	r.appendMemoryAssessmentAudit(current, record.ActorPrincipalID, "AssessmentSetupRetryQueued")
	payload := r.assessmentEvents[len(r.assessmentEvents)-1].Payload
	payload["setup_job_id"] = job.ID
	payload["previous_failure_code"] = previousFailureCode
	return job, current, nil
}

func assessmentSetupRetryRecorded(events []memoryAssessmentAudit, assessmentID string, version int64) bool {
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		if event.AssessmentID == assessmentID && event.AssessmentVersion == version {
			return event.Type == "AssessmentSetupRetryQueued"
		}
	}
	return false
}

func (r *MemoryAssessmentRepository) ListAssessmentSetupJobs(_ context.Context, scope Scope, assessmentID string) ([]AssessmentSetupJob, error) {
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	values := make([]AssessmentSetupJob, 0)
	for _, job := range r.setupJobs {
		if job.TenantID == scope.TenantID && job.LegalEntityID == scope.LegalEntityID && (assessmentID == "" || job.AssessmentID == assessmentID) {
			values = append(values, job)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].CreatedAt.Before(values[j].CreatedAt) })
	return values, nil
}

func (r *MemoryAssessmentRepository) GetAssessmentSetupJob(_ context.Context, scope Scope, assessmentID string) (AssessmentSetupJob, error) {
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	for _, job := range r.setupJobs {
		if job.TenantID == scope.TenantID && job.LegalEntityID == scope.LegalEntityID && job.AssessmentID == assessmentID {
			return job, nil
		}
	}
	return AssessmentSetupJob{}, ErrNotFound
}

func sameAssessmentJobLease(current, claim AssessmentSetupJob, at time.Time) bool {
	return current.State == AssessmentJobLeased && current.LeaseToken != "" && current.LeaseToken == claim.LeaseToken &&
		current.LeaseExpiresAt != nil && current.LeaseExpiresAt.After(at)
}

func validAssessmentFailureCode(value string) bool {
	clean := strings.TrimSpace(value)
	return value == clean && clean != "" && len(clean) <= 128
}

var _ AssessmentSetupRepository = (*MemoryAssessmentRepository)(nil)
