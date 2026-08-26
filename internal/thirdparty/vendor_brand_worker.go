package thirdparty

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const (
	VendorBrandFailureUnsafeDestination = "UNSAFE_DESTINATION"
	VendorBrandFailureUnsafeURL         = "UNSAFE_URL"
	VendorBrandFailureTimeout           = "FETCH_TIMEOUT"
	VendorBrandFailureTooLarge          = "RESPONSE_TOO_LARGE"
	VendorBrandFailureMedia             = "UNSUPPORTED_MEDIA"
	VendorBrandFailureImage             = "INVALID_IMAGE"
	VendorBrandFailureUnavailable       = "ICON_UNAVAILABLE"
	VendorBrandFailureStorage           = "ARTIFACT_STORAGE_FAILED"
	VendorBrandFailureCompletion        = "COMPLETION_FAILED"
	VendorBrandFailureStale             = "STALE_VENDOR_IDENTITY"
	VendorBrandFailureAttemptsExhausted = "ATTEMPTS_EXHAUSTED"

	defaultVendorBrandJobLease    = time.Minute
	defaultVendorBrandJobAttempts = 5
	defaultVendorBrandJobBackoff  = 5 * time.Minute
	defaultVendorBrandRefresh     = 30 * 24 * time.Hour
)

var (
	ErrVendorBrandJobLeaseLost = errors.New("vendor brand job lease is no longer current")
	ErrVendorBrandJobStale     = errors.New("vendor brand job no longer matches the vendor identity")
)

type VendorBrandEvent struct {
	TenantID      string
	VendorID      string
	AssetID       string
	AssetVersion  int64
	VendorVersion int64
	EventType     string
	ArtifactKey   string
	SourceDigest  string
	OccurredAt    time.Time
	EventVersion  int64
}

const VendorBrandDiscoveredEvent = "VendorBrandDiscovered"

const (
	VendorBrandApprovedEvent = "VendorBrandApproved"
	VendorBrandRemovedEvent  = "VendorBrandRemoved"
)

type VendorBrandReceipt struct {
	TenantID        string
	VendorID        string
	IdempotencyKey  string
	Command         string
	ExpectedVersion int64
	ResultVersion   int64
}

type VendorBrandWorkerRepository interface {
	GetVendorForBrandDiscovery(context.Context, string, string) (Vendor, error)
	ClaimVendorBrandJobs(context.Context, string, time.Time, time.Duration, int, int) ([]VendorBrandJob, error)
	CompleteVendorBrandJob(context.Context, VendorBrandJob, VendorBrandAsset, time.Time) (VendorBrandAsset, error)
	CancelVendorBrandJob(context.Context, VendorBrandJob, string, time.Time) error
	FailVendorBrandJob(context.Context, VendorBrandJob, int, string, time.Time, time.Time) (VendorBrandJob, error)
	VendorBrandQueueHealth(context.Context) (workflowruntime.QueueHealth, error)
}

type VendorBrandWorker struct {
	repository  VendorBrandWorkerRepository
	store       evidence.ObjectStore
	discoverer  *VendorBrandDiscoverer
	workerID    string
	lease       time.Duration
	maxAttempts int
	maxBackoff  time.Duration
	refresh     time.Duration
	newID       func() (string, error)
	now         func() time.Time
	afterStore  func()
}

func NewVendorBrandWorker(repository VendorBrandWorkerRepository, store evidence.ObjectStore, discoverer *VendorBrandDiscoverer, workerID string) *VendorBrandWorker {
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		workerID = "vendor-brand-worker"
	}
	return &VendorBrandWorker{
		repository: repository, store: store, discoverer: discoverer, workerID: workerID,
		lease: defaultVendorBrandJobLease, maxAttempts: defaultVendorBrandJobAttempts, maxBackoff: defaultVendorBrandJobBackoff,
		refresh: defaultVendorBrandRefresh, newID: id.NewUUIDv7, now: time.Now,
	}
}

func (w *VendorBrandWorker) QueueHealth(ctx context.Context) (workflowruntime.QueueHealth, error) {
	if w == nil || w.repository == nil {
		return workflowruntime.QueueHealth{}, errors.New("vendor brand discovery is not configured")
	}
	return w.repository.VendorBrandQueueHealth(ctx)
}

func (w *VendorBrandWorker) Configure(lease time.Duration, maxAttempts int, maxBackoff time.Duration) {
	if lease > vendorBrandRequestTimeout {
		w.lease = lease
	}
	if maxAttempts >= 1 && maxAttempts <= 20 {
		w.maxAttempts = maxAttempts
	}
	if maxBackoff > 0 {
		w.maxBackoff = maxBackoff
	}
}

func (w *VendorBrandWorker) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if w == nil || w.repository == nil || w.store == nil || w.discoverer == nil {
		return 0, errors.New("vendor brand discovery is not configured")
	}
	now = now.UTC()
	jobs, err := w.repository.ClaimVendorBrandJobs(ctx, w.workerID, now, w.lease, w.maxAttempts, boundedVendorBrandJobLimit(limit))
	if err != nil {
		return 0, err
	}
	processed := 0
	var failures []error
	for _, job := range jobs {
		if ctx.Err() != nil {
			return processed, errors.Join(append(failures, ctx.Err())...)
		}
		processed++
		if err := w.process(ctx, job, now); err != nil {
			failures = append(failures, fmt.Errorf("process vendor brand job %s: %w", job.ID, err))
		}
	}
	return processed, errors.Join(failures...)
}

func (w *VendorBrandWorker) process(ctx context.Context, job VendorBrandJob, now time.Time) error {
	vendor, err := w.repository.GetVendorForBrandDiscovery(ctx, job.TenantID, job.VendorID)
	if err != nil {
		return w.release(ctx, job, VendorBrandFailureCompletion, now, err)
	}
	if vendor.Version != job.VendorVersion || vendor.WebsiteDomain != job.WebsiteDomain || job.WebsiteDomain == "" {
		if err := w.repository.CancelVendorBrandJob(ctx, job, VendorBrandFailureStale, now); err != nil {
			return err
		}
		return nil
	}
	result, err := w.discoverer.Discover(ctx, job.WebsiteDomain)
	if err != nil {
		return w.release(ctx, job, vendorBrandFailureCode(err), now, err)
	}
	if err := validateDiscoveredVendorBrand(result); err != nil {
		return w.release(ctx, job, VendorBrandFailureImage, now, err)
	}
	assetID, err := w.newID()
	if err != nil {
		return w.release(ctx, job, VendorBrandFailureCompletion, now, err)
	}
	key := vendorBrandObjectKey(job.TenantID, job.VendorID, job.VendorVersion, result.SourceDigest)
	object, err := w.putObject(ctx, key, result.PNG)
	if err != nil {
		return w.release(ctx, job, VendorBrandFailureStorage, now, err)
	}
	if w.afterStore != nil {
		w.afterStore()
	}
	completedAt := w.now().UTC()
	nextRefresh := completedAt.Add(w.refresh)
	asset := VendorBrandAsset{
		ID: assetID, TenantID: job.TenantID, VendorID: job.VendorID,
		SourceKind: VendorBrandAssetDiscovered, State: VendorBrandAssetCurrent, SourceDomain: job.WebsiteDomain,
		ArtifactKey: object.Key, SourceDigest: result.SourceDigest, MediaType: result.MediaType,
		PixelWidth: result.PixelWidth, PixelHeight: result.PixelHeight, ByteSize: object.SizeBytes,
		RetrievedAt: &completedAt, NextRefreshAt: &nextRefresh, CreatedAt: completedAt, UpdatedAt: completedAt, Version: 1,
	}
	asset.AssetToken = brandAssetToken(asset)
	if _, err := w.repository.CompleteVendorBrandJob(ctx, job, asset, completedAt); err != nil {
		// Canonical bytes are content-addressed and immutable. Keep an
		// unreferenced copy when the database outcome is uncertain: deleting it
		// could remove bytes referenced by a completion whose acknowledgement was
		// lost. A retry writes the same key, and storage reconciliation can remove
		// keys that have no authoritative asset record.
		if errors.Is(err, ErrVendorBrandJobStale) {
			if cancelErr := w.repository.CancelVendorBrandJob(ctx, job, VendorBrandFailureStale, completedAt); cancelErr == nil {
				return nil
			} else {
				return errors.Join(err, cancelErr)
			}
		}
		if errors.Is(err, ErrVendorBrandJobLeaseLost) {
			return err
		}
		return err
	}
	return nil
}

func (w *VendorBrandWorker) putObject(ctx context.Context, key string, content []byte) (evidence.ObjectInfo, error) {
	if len(content) == 0 || len(content) > vendorBrandImageLimit {
		return evidence.ObjectInfo{}, ErrInvalidVendorBrandImage
	}
	if existing, err := w.openMatchingObject(ctx, key, content); err == nil {
		return existing, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return evidence.ObjectInfo{}, err
	}
	stored, err := w.store.Put(ctx, key, bytes.NewReader(content), vendorBrandImageLimit)
	if err == nil {
		return stored, nil
	}
	// Another writer may have won between Open and Put. Re-open and accept only
	// the exact content-addressed bytes.
	if existing, openErr := w.openMatchingObject(ctx, key, content); openErr == nil {
		return existing, nil
	}
	return evidence.ObjectInfo{}, err
}

func (w *VendorBrandWorker) openMatchingObject(ctx context.Context, key string, content []byte) (evidence.ObjectInfo, error) {
	opened, err := w.store.Open(ctx, key)
	if err != nil {
		return evidence.ObjectInfo{}, err
	}
	defer opened.Close()
	limited := &io.LimitedReader{R: opened, N: vendorBrandImageLimit + 1}
	existing, err := io.ReadAll(limited)
	if err != nil {
		return evidence.ObjectInfo{}, err
	}
	if !bytes.Equal(existing, content) {
		return evidence.ObjectInfo{}, errors.New("vendor brand content address already contains different bytes")
	}
	digest := sha256.Sum256(existing)
	return evidence.ObjectInfo{Key: key, SizeBytes: int64(len(existing)), SHA256: hex.EncodeToString(digest[:])}, nil
}

func (w *VendorBrandWorker) release(ctx context.Context, job VendorBrandJob, code string, now time.Time, cause error) error {
	if w.now != nil {
		now = w.now().UTC()
	}
	next := now.Add(vendorBrandJobRetryDelay(job.Attempts, w.maxBackoff))
	if _, err := w.repository.FailVendorBrandJob(ctx, job, w.maxAttempts, code, now, next); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

func vendorBrandFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrUnsafeVendorBrandDestination):
		return VendorBrandFailureUnsafeDestination
	case errors.Is(err, ErrUnsafeVendorBrandURL):
		return VendorBrandFailureUnsafeURL
	case errors.Is(err, ErrVendorBrandTimeout):
		return VendorBrandFailureTimeout
	case errors.Is(err, ErrVendorBrandResponseTooLarge):
		return VendorBrandFailureTooLarge
	case errors.Is(err, ErrUnsupportedVendorBrandMedia):
		return VendorBrandFailureMedia
	case errors.Is(err, ErrInvalidVendorBrandImage):
		return VendorBrandFailureImage
	default:
		return VendorBrandFailureUnavailable
	}
}

func vendorBrandObjectKey(tenantID, vendorID string, vendorVersion int64, sourceDigest string) string {
	tenantDigest := stringsDigest(tenantID)
	vendorDigest := stringsDigest(vendorID)
	return fmt.Sprintf("vendor-brands/%s/%s/v%d/%s.png", tenantDigest[:16], vendorDigest[:16], vendorVersion, sourceDigest)
}

func stringsDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func vendorBrandJobRetryDelay(attempt int, maximum time.Duration) time.Duration {
	if maximum <= 0 {
		maximum = defaultVendorBrandJobBackoff
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

func boundedVendorBrandJobLimit(limit int) int {
	if limit < 1 {
		return 25
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func validVendorBrandFailureCode(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && trimmed == value && len(value) <= 128
}

func validVendorBrandAssetCompletion(claim VendorBrandJob, asset VendorBrandAsset) bool {
	key := strings.TrimSpace(asset.ArtifactKey)
	if strings.TrimSpace(asset.ID) == "" || asset.TenantID != claim.TenantID || asset.VendorID != claim.VendorID || asset.SourceKind != VendorBrandAssetDiscovered || asset.State != VendorBrandAssetCurrent || asset.SourceDomain != claim.WebsiteDomain || key == "" || key != asset.ArtifactKey || len(key) > 1024 || asset.MediaType != "image/png" || asset.PixelWidth < 1 || asset.PixelWidth > vendorBrandOutputDimension || asset.PixelHeight < 1 || asset.PixelHeight > vendorBrandOutputDimension || asset.ByteSize < 1 || asset.ByteSize > vendorBrandImageLimit || asset.RetrievedAt == nil || asset.RetrievedAt.IsZero() || asset.NextRefreshAt == nil || !asset.NextRefreshAt.After(*asset.RetrievedAt) || asset.ApprovedByPrincipalID != "" || asset.CreatedAt.IsZero() || asset.UpdatedAt.Before(asset.CreatedAt) || asset.Version != 1 || len(asset.SourceDigest) != sha256.Size*2 {
		return false
	}
	if strings.ToLower(asset.SourceDigest) != asset.SourceDigest {
		return false
	}
	_, err := hex.DecodeString(asset.SourceDigest)
	return err == nil && asset.ArtifactKey == vendorBrandObjectKey(claim.TenantID, claim.VendorID, claim.VendorVersion, asset.SourceDigest)
}

func validateDiscoveredVendorBrand(value DiscoveredVendorBrand) error {
	if value.MediaType != "image/png" || value.PixelWidth < 1 || value.PixelHeight < 1 || value.PixelWidth > vendorBrandOutputDimension || value.PixelHeight > vendorBrandOutputDimension || len(value.PNG) < 8 || len(value.PNG) > vendorBrandImageLimit || vendorBrandMagicMedia(value.PNG) != "image/png" || len(value.SourceDigest) != sha256.Size*2 {
		return ErrInvalidVendorBrandImage
	}
	if _, err := hex.DecodeString(value.SourceDigest); err != nil {
		return ErrInvalidVendorBrandImage
	}
	contentDigest := sha256.Sum256(value.PNG)
	if value.SourceDigest != hex.EncodeToString(contentDigest[:]) {
		return ErrInvalidVendorBrandImage
	}
	config, err := png.DecodeConfig(bytes.NewReader(value.PNG))
	if err != nil || config.Width != value.PixelWidth || config.Height != value.PixelHeight {
		return ErrInvalidVendorBrandImage
	}
	return nil
}
