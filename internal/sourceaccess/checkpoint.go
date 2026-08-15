package sourceaccess

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

const (
	HardMaxCheckpointPositionBytes  = 16 << 10
	HardMaxCheckpointErrorCodeBytes = 128
	HardMaxCheckpointWorkerBytes    = 128
)

var (
	ErrCheckpointClaimLost       = errors.New("source binding checkpoint claim lost")
	ErrCheckpointProcessingProof = errors.New("source binding checkpoint processing proof is unavailable")
)

type CheckpointPositionKind string

const (
	CheckpointCursor    CheckpointPositionKind = "CURSOR"
	CheckpointETag      CheckpointPositionKind = "ETAG"
	CheckpointWatermark CheckpointPositionKind = "WATERMARK"
	CheckpointEventID   CheckpointPositionKind = "EVENT_ID"
)

type CheckpointPosition struct {
	Kind  CheckpointPositionKind `json:"kind"`
	Value string                 `json:"value"`
}

type BindingCheckpoint struct {
	TenantID       string             `json:"tenant_id"`
	SourceID       string             `json:"source_id"`
	BindingID      string             `json:"binding_id"`
	BindingVersion int64              `json:"binding_version"`
	Position       CheckpointPosition `json:"position"`
	Attempts       int                `json:"attempts"`
	LockedBy       string             `json:"locked_by,omitempty"`
	LeaseUntil     *time.Time         `json:"lease_until,omitempty"`
	NextAttemptAt  time.Time          `json:"next_attempt_at"`
	LastErrorCode  string             `json:"last_error_code,omitempty"`
	FailedAt       *time.Time         `json:"failed_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type CheckpointRepository interface {
	EnsureBindingCheckpoint(context.Context, string, string, string, int64, time.Time) (BindingCheckpoint, error)
	BindingCheckpoint(context.Context, string, string, int64) (BindingCheckpoint, error)
	ClaimBindingCheckpoints(context.Context, string, time.Time, time.Duration, int) ([]BindingCheckpoint, error)
	AdvanceBindingCheckpoint(context.Context, BindingCheckpoint, CheckpointPosition, time.Time, time.Time) (BindingCheckpoint, error)
	FailBindingCheckpoint(context.Context, BindingCheckpoint, int, string, time.Time, time.Time) (bool, error)
}

// InboxReceiptReader is deliberately structural: internal/runtime.PostgresRepository
// satisfies it without sourceaccess depending on runtime. A checkpoint may advance
// only after the durable consumer receipt exists, so a crash before advancement
// replays safely rather than skipping source data.
type InboxReceiptReader interface {
	InboxProcessed(context.Context, string, string, string) (bool, error)
}

type CheckpointService struct {
	repo  CheckpointRepository
	inbox InboxReceiptReader
}

func NewCheckpointService(repo CheckpointRepository, inbox InboxReceiptReader) *CheckpointService {
	return &CheckpointService{repo: repo, inbox: inbox}
}

func (s *CheckpointService) Ensure(ctx context.Context, tenantID, sourceID, bindingID string, bindingVersion int64, now time.Time) (BindingCheckpoint, error) {
	if s == nil || s.repo == nil {
		return BindingCheckpoint{}, ErrCatalogStorage
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return s.repo.EnsureBindingCheckpoint(ctx, tenantID, sourceID, bindingID, bindingVersion, now.UTC())
}

func (s *CheckpointService) Claim(ctx context.Context, worker string, now time.Time, lease time.Duration, limit int) ([]BindingCheckpoint, error) {
	if s == nil || s.repo == nil {
		return nil, ErrCatalogStorage
	}
	worker = strings.TrimSpace(worker)
	if !validCheckpointToken(worker, HardMaxCheckpointWorkerBytes) || lease <= 0 || lease > 30*time.Minute {
		return nil, ErrCatalogInvalid
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if limit < 1 {
		limit = 100
	}
	if limit > HardMaxCatalogListRows {
		limit = HardMaxCatalogListRows
	}
	return s.repo.ClaimBindingCheckpoints(ctx, worker, now.UTC(), lease, limit)
}

// CheckpointInboxEventID derives the durable idempotency receipt identity from
// the exact Binding plus old->new checkpoint transition. Replays of the same
// batch therefore reuse one inbox receipt, while an unrelated successful event
// cannot authorize a different cursor or watermark advancement.
func CheckpointInboxEventID(checkpoint BindingCheckpoint, consumer string, position CheckpointPosition) (string, error) {
	consumer = strings.TrimSpace(consumer)
	if !validCheckpointToken(checkpoint.TenantID, HardMaxIdentifierBytes) ||
		!validCheckpointToken(checkpoint.SourceID, HardMaxIdentifierBytes) ||
		!validCheckpointToken(checkpoint.BindingID, HardMaxIdentifierBytes) ||
		checkpoint.BindingVersion < 1 ||
		!validCheckpointToken(consumer, HardMaxCheckpointWorkerBytes) {
		return "", ErrCatalogInvalid
	}
	if checkpoint.Position.Kind != "" || checkpoint.Position.Value != "" {
		if err := validateCheckpointPosition(checkpoint.Position); err != nil {
			return "", err
		}
	}
	if err := validateCheckpointPosition(position); err != nil {
		return "", err
	}
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%s\x1f%d\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s",
		checkpoint.TenantID,
		checkpoint.SourceID,
		checkpoint.BindingID,
		checkpoint.BindingVersion,
		consumer,
		checkpoint.Position.Kind,
		checkpoint.Position.Value,
		position.Kind,
		position.Value,
	)
	return "source-binding:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *CheckpointService) AdvanceAfterInbox(ctx context.Context, checkpoint BindingCheckpoint, consumer string, position CheckpointPosition, at, next time.Time) (BindingCheckpoint, error) {
	if s == nil || s.repo == nil || s.inbox == nil {
		return BindingCheckpoint{}, ErrCheckpointProcessingProof
	}
	eventID, err := CheckpointInboxEventID(checkpoint, consumer, position)
	if err != nil {
		return BindingCheckpoint{}, err
	}
	processed, err := s.inbox.InboxProcessed(ctx, checkpoint.TenantID, strings.TrimSpace(consumer), eventID)
	if err != nil {
		return BindingCheckpoint{}, err
	}
	if !processed {
		return BindingCheckpoint{}, ErrCheckpointProcessingProof
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if next.IsZero() {
		next = at
	}
	if next.Before(at) {
		return BindingCheckpoint{}, ErrCatalogInvalid
	}
	return s.repo.AdvanceBindingCheckpoint(ctx, checkpoint, position, at.UTC(), next.UTC())
}

func (s *CheckpointService) Fail(ctx context.Context, checkpoint BindingCheckpoint, maxAttempts int, errorCode string, at, next time.Time) (bool, error) {
	if s == nil || s.repo == nil {
		return false, ErrCatalogStorage
	}
	errorCode = strings.TrimSpace(errorCode)
	if maxAttempts < 1 || maxAttempts > 100 || !validCheckpointToken(errorCode, HardMaxCheckpointErrorCodeBytes) {
		return false, ErrCatalogInvalid
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if next.IsZero() || next.Before(at) {
		return false, ErrCatalogInvalid
	}
	return s.repo.FailBindingCheckpoint(ctx, checkpoint, maxAttempts, errorCode, at.UTC(), next.UTC())
}

func validateCheckpointPosition(position CheckpointPosition) error {
	value := strings.TrimSpace(position.Value)
	if value != position.Value || value == "" || len(value) > HardMaxCheckpointPositionBytes || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return fmt.Errorf("%w: checkpoint value is invalid", ErrCatalogInvalid)
	}
	switch position.Kind {
	case CheckpointCursor, CheckpointETag, CheckpointWatermark, CheckpointEventID:
		return nil
	default:
		return fmt.Errorf("%w: checkpoint position kind is invalid", ErrCatalogInvalid)
	}
}

func validCheckpointToken(value string, maximum int) bool {
	if value == "" || len(value) > maximum || value != strings.TrimSpace(value) {
		return false
	}
	return strings.IndexFunc(value, unicode.IsControl) < 0
}

func statefulBindingRevision(binding BindingRevision) bool {
	if !binding.IsCurrent || binding.Status != RevisionActive {
		return false
	}
	for _, operation := range binding.Operations {
		if operation == OperationPage || operation == OperationChanges {
			return true
		}
	}
	return false
}
