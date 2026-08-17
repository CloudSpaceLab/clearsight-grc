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

const HardMaxCheckpointPositionBytes = 16 << 10

var (
	ErrCheckpointConflict        = errors.New("source binding checkpoint changed")
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
	Generation     int64              `json:"generation"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type CheckpointRepository interface {
	EnsureBindingCheckpoint(context.Context, string, string, string, int64, time.Time) (BindingCheckpoint, error)
	BindingCheckpoint(context.Context, string, string, int64) (BindingCheckpoint, error)
	AdvanceBindingCheckpoint(context.Context, BindingCheckpoint, CheckpointPosition, time.Time) (BindingCheckpoint, error)
}

// InboxReceiptReader is deliberately structural: runtime repositories satisfy
// it without sourceaccess importing the runtime package. Runtime owns leases,
// retries and backoff. The checkpoint stores only durable source position.
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

func (s *CheckpointService) Get(ctx context.Context, tenantID, bindingID string, bindingVersion int64) (BindingCheckpoint, error) {
	if s == nil || s.repo == nil {
		return BindingCheckpoint{}, ErrCatalogStorage
	}
	return s.repo.BindingCheckpoint(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(bindingID), bindingVersion)
}

// CheckpointInboxEventID derives the durable idempotency receipt identity from
// the exact Binding plus old->new checkpoint transition. Runtime retries of the
// same batch therefore reuse one inbox receipt, while an unrelated successful
// event cannot authorize a different cursor or watermark advancement.
func CheckpointInboxEventID(checkpoint BindingCheckpoint, consumer string, position CheckpointPosition) (string, error) {
	consumer = strings.TrimSpace(consumer)
	if !validCheckpointToken(checkpoint.TenantID, HardMaxIdentifierBytes) ||
		!validCheckpointToken(checkpoint.SourceID, HardMaxIdentifierBytes) ||
		!validCheckpointToken(checkpoint.BindingID, HardMaxIdentifierBytes) ||
		checkpoint.BindingVersion < 1 || checkpoint.Generation < 0 ||
		!validCheckpointToken(consumer, HardMaxIdentifierBytes) {
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
	_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%s\x1f%d\x1f%d\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s",
		checkpoint.TenantID,
		checkpoint.SourceID,
		checkpoint.BindingID,
		checkpoint.BindingVersion,
		checkpoint.Generation,
		consumer,
		checkpoint.Position.Kind,
		checkpoint.Position.Value,
		position.Kind,
		position.Value,
	)
	return "source-binding:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *CheckpointService) AdvanceAfterInbox(ctx context.Context, checkpoint BindingCheckpoint, consumer string, position CheckpointPosition, at time.Time) (BindingCheckpoint, error) {
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
	return s.repo.AdvanceBindingCheckpoint(ctx, checkpoint, position, at.UTC())
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

func checkpointPositionEqual(left, right CheckpointPosition) bool {
	return left.Kind == right.Kind && left.Value == right.Value
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
