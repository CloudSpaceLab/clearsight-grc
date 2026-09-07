package evidence

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestArtifactCreationSchedulesInspection(t *testing.T) {
	now := time.Now().UTC()
	repo := NewMemoryRepository(nil, []Request{{ID: "request", TenantID: "tenant", Status: RequestReady, Deadline: now.Add(time.Hour)}})
	_, err := repo.CreateArtifact(context.Background(), Artifact{ID: "artifact", RequestID: "request", TenantID: "tenant", Status: ArtifactStoredUnscanned, CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	// The upload transaction must own a durable inspection job before returning.
	jobs, err := repo.ClaimArtifactScanJobs(context.Background(), "worker", now, 1)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("artifact creation has no inspection job: %v %v", jobs, err)
	}
}

type artifactScannerFunc func(context.Context, io.Reader, int64) (ArtifactScanResult, error)

func (f artifactScannerFunc) Scan(ctx context.Context, r io.Reader, n int64) (ArtifactScanResult, error) {
	return f(ctx, r, n)
}

func scanFixture(t *testing.T) (*MemoryRepository, *MemoryObjectStore, Artifact, time.Time) {
	t.Helper()
	now := time.Now().UTC()
	repo := NewMemoryRepository(nil, []Request{{ID: "request", TenantID: "tenant", Status: RequestReady, Deadline: now.Add(time.Hour)}})
	store := NewMemoryObjectStore()
	info, err := store.Put(context.Background(), "key", bytes.NewBufferString("safe bytes"), 100)
	if err != nil {
		t.Fatal(err)
	}
	a, err := repo.CreateArtifact(context.Background(), Artifact{ID: "artifact", RequestID: "request", TenantID: "tenant", SHA256: info.SHA256, SizeBytes: info.SizeBytes, StorageKey: info.Key, Status: ArtifactStoredUnscanned, CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	return repo, store, a, now
}

func TestArtifactScanWorkerKeepsFailuresUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name, verdict, bytes string
		unavailable, partial bool
		want                 ArtifactStatus
		failure              string
	}{
		{name: "clean", verdict: "CLEAN", want: ArtifactAvailable},
		{name: "infected", verdict: "INFECTED", want: ArtifactQuarantined},
		{name: "unavailable", unavailable: true, want: ArtifactStoredUnscanned, failure: "SCANNER_UNAVAILABLE"},
		{name: "truncated", verdict: "CLEAN", bytes: "short", want: ArtifactStoredUnscanned, failure: "INTEGRITY_MISMATCH"},
		{name: "oversize", verdict: "CLEAN", bytes: "safe bytes plus", want: ArtifactStoredUnscanned, failure: "INTEGRITY_MISMATCH"},
		{name: "wrong digest", verdict: "CLEAN", bytes: "evil bytes", want: ArtifactStoredUnscanned, failure: "INTEGRITY_MISMATCH"},
		{name: "partial scan", verdict: "CLEAN", partial: true, want: ArtifactStoredUnscanned, failure: "INTEGRITY_MISMATCH"},
		{name: "unknown verdict", verdict: "OKAY", want: ArtifactStoredUnscanned, failure: "SCANNER_UNAVAILABLE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, store, a, now := scanFixture(t)
			if tc.bytes != "" {
				_, _ = store.Put(context.Background(), a.StorageKey, bytes.NewBufferString(tc.bytes), 100)
			}
			scanner := artifactScannerFunc(func(_ context.Context, r io.Reader, _ int64) (ArtifactScanResult, error) {
				if tc.unavailable {
					return ArtifactScanResult{}, errors.New("offline")
				}
				if !tc.partial {
					_, _ = io.Copy(io.Discard, r)
				}
				return ArtifactScanResult{Verdict: tc.verdict, Scanner: "test-fixture", Version: "1"}, nil
			})
			worker := NewArtifactScanWorker(repo, store, scanner, "worker", 100)
			worker.now = func() time.Time { return now }
			_, err := worker.Maintain(context.Background(), now, 1)
			if err != nil && tc.failure == "" {
				t.Fatal(err)
			}
			if got := repo.artifacts[a.ID].Status; got != tc.want {
				t.Fatalf("status=%s want=%s", got, tc.want)
			}
			if len(repo.scanReceipts) != 1 {
				t.Fatalf("receipts=%v", repo.scanReceipts)
			}
			receipt := repo.scanReceipts[0]
			if receipt.ArtifactID != a.ID || receipt.SHA256 != a.SHA256 || receipt.FailureCode != tc.failure {
				t.Fatalf("receipt=%+v", receipt)
			}
		})
	}
}

func TestArtifactScanLeaseExpiryAndReplay(t *testing.T) {
	repo, _, a, now := scanFixture(t)
	ctx := context.Background()
	jobs, _ := repo.ClaimArtifactScanJobs(ctx, "worker", now, 1)
	job := jobs[0]
	receipt := ArtifactScanReceipt{ArtifactID: a.ID, SHA256: a.SHA256, SizeBytes: a.SizeBytes, Attempt: job.Attempt, Verdict: "CLEAN", Scanner: "fixture", Version: "1", InspectedAt: now}
	if err := repo.CompleteArtifactScan(ctx, job, receipt, now.Add(time.Minute)); !errors.Is(err, ErrArtifactScanLease) {
		t.Fatalf("expired completion=%v", err)
	}
	jobs, _ = repo.ClaimArtifactScanJobs(ctx, "worker", now.Add(time.Minute), 1)
	if err := repo.CompleteArtifactScan(ctx, job, receipt, now.Add(time.Minute)); !errors.Is(err, ErrArtifactScanLease) {
		t.Fatalf("old attempt=%v", err)
	}
	job = jobs[0]
	receipt.Attempt = job.Attempt
	receipt.InspectedAt = now.Add(time.Minute)
	if err := repo.CompleteArtifactScan(ctx, job, receipt, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repo.CompleteArtifactScan(ctx, job, receipt, now.Add(time.Minute)); !errors.Is(err, ErrArtifactScanLease) {
		t.Fatalf("replay=%v", err)
	}
	if len(repo.scanReceipts) != 1 {
		t.Fatal("replayed receipt")
	}
}

func TestArtifactScannerUnavailableExhaustsBoundedRetries(t *testing.T) {
	repo, store, a, now := scanFixture(t)
	worker := NewArtifactScanWorker(repo, store, nil, "worker", 100)
	worker.now = func() time.Time { return now }
	for i := 0; i < 10; i++ {
		_, _ = worker.Maintain(context.Background(), now, 1)
		now = now.Add(10 * time.Minute)
	}
	if job := repo.scanJobs[a.ID]; job.State != "FAILED" || job.Attempt != 5 || len(repo.scanReceipts) != 5 {
		t.Fatalf("retry budget=%+v receipts=%d", job, len(repo.scanReceipts))
	}
	if repo.artifacts[a.ID].Status != ArtifactStoredUnscanned {
		t.Fatal("failed scanner released artifact")
	}
	health, err := worker.QueueHealth(context.Background())
	if err != nil || health.Terminal != 1 || health.HighestAttempts != 5 {
		t.Fatalf("health=%+v err=%v", health, err)
	}
}

func TestArtifactCannotOverwriteInspectionIdentity(t *testing.T) {
	repo, _, a, _ := scanFixture(t)
	if _, err := repo.CreateArtifact(context.Background(), a); err == nil {
		t.Fatal("duplicate manifest replaced inspection job")
	}
}

type cancelEOFStore struct {
	ObjectStore
	cancel context.CancelFunc
}
type cancelEOFReader struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (s cancelEOFStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	r, err := s.ObjectStore.Open(ctx, key)
	if err != nil {
		return nil, err
	}
	return cancelEOFReader{r, s.cancel}, nil
}
func (r cancelEOFReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if err == io.EOF {
		r.cancel()
	}
	return n, err
}

func TestArtifactScanCancellationDuringFinalIntegrityCheckCannotRelease(t *testing.T) {
	repo, store, a, now := scanFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scanner := artifactScannerFunc(func(_ context.Context, r io.Reader, n int64) (ArtifactScanResult, error) {
		_, err := io.CopyN(io.Discard, r, n)
		return ArtifactScanResult{Verdict: "CLEAN", Scanner: "fixture", Version: "1"}, err
	})
	worker := NewArtifactScanWorker(repo, cancelEOFStore{store, cancel}, scanner, "worker", 100)
	worker.now = func() time.Time { return now }
	receipt := worker.inspect(ctx, ArtifactScanJob{Artifact: a, Attempt: 1})
	if receipt.Verdict == "CLEAN" {
		t.Fatal("expired inspection released an artifact")
	}
}
