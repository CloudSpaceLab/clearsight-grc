package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"time"

	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type ArtifactScanWorker struct {
	repo     ArtifactScanRepository
	store    ObjectStore
	scanner  ArtifactScanner
	workerID string
	maxBytes int64
	now      func() time.Time
}

func NewArtifactScanWorker(repo ArtifactScanRepository, store ObjectStore, scanner ArtifactScanner, workerID string, maxBytes int64) *ArtifactScanWorker {
	return &ArtifactScanWorker{repo: repo, store: store, scanner: scanner, workerID: workerID, maxBytes: maxBytes, now: time.Now}
}

func (w *ArtifactScanWorker) Maintain(ctx context.Context, _ time.Time, limit int) (int, error) {
	if w.repo == nil || w.store == nil || w.workerID == "" {
		return 0, ErrArtifactScannerUnavailable
	}
	if limit > 10 {
		limit = 10
	}
	count := 0
	var failures []error
	for i := 0; i < limit; i++ {
		if err := ctx.Err(); err != nil {
			return count, errors.Join(append(failures, err)...)
		}
		jobs, err := w.repo.ClaimArtifactScanJobs(ctx, w.workerID, w.now().UTC(), 1)
		if err != nil {
			return count, errors.Join(append(failures, err)...)
		}
		if len(jobs) == 0 {
			break
		}
		job := jobs[0]
		scanCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		receipt := w.inspect(scanCtx, job)
		cancel()
		now := w.now().UTC()
		receipt.InspectedAt = now
		if err := w.repo.CompleteArtifactScan(ctx, job, receipt, now); err != nil {
			failures = append(failures, err)
			continue
		}
		count++
		if receipt.Verdict == "UNAVAILABLE" {
			failures = append(failures, ErrArtifactScannerUnavailable)
		}
	}
	return count, errors.Join(failures...)
}

func (w *ArtifactScanWorker) QueueHealth(ctx context.Context) (workflowruntime.QueueHealth, error) {
	if w.repo == nil {
		return workflowruntime.QueueHealth{}, ErrArtifactScannerUnavailable
	}
	return w.repo.ArtifactScanQueueHealth(ctx)
}

func (w *ArtifactScanWorker) inspect(ctx context.Context, job ArtifactScanJob) ArtifactScanReceipt {
	a := job.Artifact
	receipt := ArtifactScanReceipt{ArtifactID: a.ID, SHA256: a.SHA256, SizeBytes: a.SizeBytes, Attempt: job.Attempt, Verdict: "UNAVAILABLE", FailureCode: "SCANNER_UNAVAILABLE"}
	if w.scanner == nil {
		return receipt
	}
	if a.SizeBytes <= 0 || a.SizeBytes > w.maxBytes {
		receipt.FailureCode = "INTEGRITY_MISMATCH"
		return receipt
	}
	reader, err := w.store.Open(ctx, a.StorageKey)
	if err != nil {
		receipt.FailureCode = "OBJECT_UNAVAILABLE"
		return receipt
	}
	defer reader.Close()
	// Closing on cancellation releases adapters that honour cancellation by Close.
	stop := context.AfterFunc(ctx, func() { _ = reader.Close() })
	defer stop()
	limited := &io.LimitedReader{R: reader, N: a.SizeBytes + 1}
	digest := sha256.New()
	result, err := w.scanner.Scan(ctx, io.TeeReader(limited, digest), a.SizeBytes)
	if errors.Is(err, ErrArtifactScanIntegrity) {
		receipt.FailureCode = "INTEGRITY_MISMATCH"
		return receipt
	}
	if err != nil || ctx.Err() != nil {
		return receipt
	}
	consumed := a.SizeBytes + 1 - limited.N
	var extra [1]byte
	n, readErr := limited.Read(extra[:])
	if ctx.Err() != nil {
		return receipt
	}
	if consumed != a.SizeBytes || n != 0 || readErr != io.EOF || hex.EncodeToString(digest.Sum(nil)) != a.SHA256 {
		receipt.FailureCode = "INTEGRITY_MISMATCH"
		return receipt
	}
	receipt.Verdict = result.Verdict
	receipt.Scanner = result.Scanner
	receipt.Version = result.Version
	receipt.FailureCode = ""
	receipt.InspectedAt = w.now().UTC()
	if !validScanReceipt(job, receipt, receipt.InspectedAt) || (result.Verdict != "CLEAN" && result.Verdict != "INFECTED") {
		receipt.Verdict = "UNAVAILABLE"
		receipt.FailureCode = "SCANNER_UNAVAILABLE"
		receipt.Scanner = ""
		receipt.Version = ""
	}
	return receipt
}
