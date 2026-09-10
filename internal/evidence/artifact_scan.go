package evidence

import (
	"context"
	"errors"
	"io"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const ArtifactScanWorkClass = "capture-artifact-inspection"
const artifactScanMaxAttempts = 5
const artifactScanLease = time.Minute

var ErrArtifactScanLease = errors.New("artifact inspection lease is no longer current")
var ErrArtifactScannerUnavailable = errors.New("artifact inspection is unavailable")
var ErrArtifactScanIntegrity = errors.New("artifact bytes do not match the recorded size")

type ArtifactScanResult struct {
	Verdict string
	Scanner string
	Version string
}

type ArtifactScanner interface {
	Scan(context.Context, io.Reader, int64) (ArtifactScanResult, error)
}

type ArtifactScanJob struct {
	Artifact      Artifact
	RecoveryCycle int
	Attempt       int
	State         string
	WorkerID      string
	LeaseUntil    time.Time
	NextAttemptAt time.Time
	FailureCode   string
}

// ArtifactScanReceipt records one completed attempt. A failure receipt is never
// a clean result, even when the scanner itself reported clean before integrity failed.
type ArtifactScanReceipt struct {
	ArtifactID  string    `json:"artifact_id"`
	SHA256      string    `json:"sha256"`
	SizeBytes   int64     `json:"size_bytes"`
	Attempt     int       `json:"attempt"`
	Verdict     string    `json:"verdict"`
	Scanner     string    `json:"scanner,omitempty"`
	Version     string    `json:"version,omitempty"`
	InspectedAt time.Time `json:"inspected_at"`
	FailureCode string    `json:"failure_code,omitempty"`
}

type ArtifactScanRepository interface {
	ClaimArtifactScanJobs(context.Context, string, time.Time, int) ([]ArtifactScanJob, error)
	CompleteArtifactScan(context.Context, ArtifactScanJob, ArtifactScanReceipt, time.Time) error
	ArtifactScanQueueHealth(context.Context) (workflowruntime.QueueHealth, error)
}

func validScanReceipt(job ArtifactScanJob, receipt ArtifactScanReceipt, now time.Time) bool {
	if receipt.ArtifactID != job.Artifact.ID || receipt.SHA256 != job.Artifact.SHA256 || receipt.SizeBytes != job.Artifact.SizeBytes || receipt.Attempt != job.Attempt || receipt.InspectedAt.IsZero() || receipt.InspectedAt.After(now) {
		return false
	}
	switch receipt.Verdict {
	case "CLEAN", "INFECTED":
		return receipt.FailureCode == "" && receipt.Scanner != "" && len(receipt.Scanner) <= 100 && receipt.Version != "" && len(receipt.Version) <= 512
	case "UNAVAILABLE":
		switch receipt.FailureCode {
		case "SCANNER_UNAVAILABLE", "OBJECT_UNAVAILABLE", "INTEGRITY_MISMATCH":
			return true
		}
	}
	return false
}

func scanOutcome(job ArtifactScanJob, receipt ArtifactScanReceipt, now time.Time) (ArtifactScanJob, ArtifactStatus) {
	status := ArtifactStoredUnscanned
	job.WorkerID = ""
	job.LeaseUntil = time.Time{}
	job.FailureCode = receipt.FailureCode
	switch receipt.Verdict {
	case "CLEAN":
		job.State = "COMPLETED"
		status = ArtifactAvailable
	case "INFECTED":
		job.State = "COMPLETED"
		status = ArtifactQuarantined
	default:
		job.State = "PENDING"
		job.NextAttemptAt = now.Add(time.Duration(job.Attempt) * time.Minute)
		if job.Attempt >= artifactScanMaxAttempts || receipt.FailureCode == "INTEGRITY_MISMATCH" {
			job.State = "FAILED"
		}
	}
	return job, status
}
