package thirdparty

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"
)

const VendorRefreshMaintenanceWorkClass = "third-party-vendor-refresh-maintenance"

var ErrInvalidRefreshMaintenancePolicy = errors.New("vendor refresh maintenance policy is invalid")

type RefreshCandidate struct {
	Scope
	RelationshipID   string           `json:"relationship_id"`
	TargetKeys       []string         `json:"target_keys"`
	Reason           string           `json:"reason"`
	ObservedVersions map[string]int64 `json:"observed_versions"`
}

type RefreshMaintenancePolicy struct {
	BatchSize                int
	Lease                    time.Duration
	DocumentLead             time.Duration
	FactConfirmationInterval time.Duration
}

type RefreshBatchReceipt struct {
	RelationshipsExamined int `json:"relationships_examined"`
	DocumentsExpired      int `json:"documents_expired"`
	AttentionsCreated     int `json:"attentions_created"`
}

type RefreshAttentionState string

const (
	RefreshAttentionOpen     RefreshAttentionState = "OPEN"
	RefreshAttentionResolved RefreshAttentionState = "RESOLVED"
)

type RefreshAttention struct {
	ID string `json:"id"`
	Scope
	RelationshipID   string                `json:"relationship_id"`
	OwnerPrincipalID string                `json:"owner_principal_id"`
	TargetKeys       []string              `json:"target_keys"`
	Reason           string                `json:"reason"`
	ObservedVersions map[string]int64      `json:"observed_versions"`
	DedupeKey        string                `json:"dedupe_key"`
	State            RefreshAttentionState `json:"state"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
	Version          int64                 `json:"version"`
}

type RefreshAttentionEvent struct {
	AttentionID    string    `json:"attention_id"`
	RelationshipID string    `json:"relationship_id"`
	EventType      string    `json:"event_type"`
	OccurredAt     time.Time `json:"occurred_at"`
}

type RefreshMaintenanceRepository interface {
	MaintainVendorRefresh(context.Context, time.Time, RefreshMaintenancePolicy) (RefreshBatchReceipt, error)
}

type RefreshMaintainer struct {
	repo   RefreshMaintenanceRepository
	policy RefreshMaintenancePolicy
	now    func() time.Time
}

func NewRefreshMaintainer(repo RefreshMaintenanceRepository, policy RefreshMaintenancePolicy) *RefreshMaintainer {
	return &RefreshMaintainer{repo: repo, policy: policy, now: time.Now}
}

func (m *RefreshMaintainer) RunBatch(ctx context.Context) (RefreshBatchReceipt, error) {
	if m == nil || m.repo == nil || !validRefreshMaintenancePolicy(m.policy) {
		return RefreshBatchReceipt{}, ErrInvalidRefreshMaintenancePolicy
	}
	return m.repo.MaintainVendorRefresh(ctx, m.now().UTC(), m.policy)
}

func (m *RefreshMaintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil {
		return 0, ErrInvalidRefreshMaintenancePolicy
	}
	policy := m.policy
	if limit > 0 && limit < policy.BatchSize {
		policy.BatchSize = limit
	}
	if m.repo == nil || !validRefreshMaintenancePolicy(policy) {
		return 0, ErrInvalidRefreshMaintenancePolicy
	}
	receipt, err := m.repo.MaintainVendorRefresh(ctx, now.UTC(), policy)
	return receipt.RelationshipsExamined, err
}

func validRefreshMaintenancePolicy(value RefreshMaintenancePolicy) bool {
	return value.BatchSize > 0 && value.BatchSize <= 500 && value.Lease >= time.Second && value.Lease <= time.Hour && value.DocumentLead >= 0 && value.DocumentLead <= 365*24*time.Hour && value.FactConfirmationInterval >= 24*time.Hour && value.FactConfirmationInterval <= 10*365*24*time.Hour
}

var refreshIdentityTargetKeys = []string{
	"VENDOR.IDENTITY.LEGAL_NAME", "VENDOR.IDENTITY.TRADING_NAME", "VENDOR.IDENTITY.REGISTRATION_REFERENCE",
	"VENDOR.IDENTITY.JURISDICTION", "VENDOR.IDENTITY.REGISTERED_ADDRESS", "VENDOR.IDENTITY.WEBSITE_DOMAIN",
}

func refreshDedupeKey(candidate RefreshCandidate) string {
	keys := append([]string(nil), candidate.TargetKeys...)
	sort.Strings(keys)
	parts := []string{candidate.TenantID, candidate.LegalEntityID, candidate.RelationshipID}
	for _, key := range keys {
		parts = append(parts, key+"="+formatVersion(candidate.ObservedVersions[key]))
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}

func formatVersion(value int64) string {
	if value == 0 {
		return "0"
	}
	const digits = "0123456789"
	buffer := [20]byte{}
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = digits[value%10]
		value /= 10
	}
	return string(buffer[index:])
}
