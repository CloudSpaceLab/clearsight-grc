package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type RequestOriginType string

const OriginMonitoringCollection RequestOriginType = "MONITORING_COLLECTION"

type RequestOrigin struct {
	Type     RequestOriginType `json:"type"`
	ID       string            `json:"id"`
	Sequence int64             `json:"sequence"`
}

func validateRequestOrigin(origin *RequestOrigin, predecessorID string) error {
	if origin == nil {
		if strings.TrimSpace(predecessorID) != "" {
			return errors.Join(ErrVersionConflict, fmt.Errorf("predecessor requires a request origin"))
		}
		return nil
	}
	if err := validateRequestOriginIdentity(*origin); err != nil {
		return err
	}
	if (origin.Sequence == 1) != (strings.TrimSpace(predecessorID) == "") {
		return errors.Join(ErrVersionConflict, fmt.Errorf("request predecessor does not match the origin sequence"))
	}
	return nil
}

func validateRequestOriginIdentity(origin RequestOrigin) error {
	if origin.Type != OriginMonitoringCollection || strings.TrimSpace(origin.ID) == "" || origin.Sequence < 1 {
		return errors.Join(ErrVersionConflict, fmt.Errorf("request origin is invalid"))
	}
	return nil
}

func validateRequestPredecessor(ctx context.Context, repo Repository, value Request) error {
	if err := validateRequestOrigin(value.Origin, value.PredecessorRequestID); err != nil || value.Origin == nil || value.Origin.Sequence == 1 {
		return err
	}
	predecessor, err := repo.GetRequest(ctx, value.TenantID, value.PredecessorRequestID)
	if err != nil {
		return errors.Join(ErrVersionConflict, err)
	}
	if predecessor.TenantID != value.TenantID || predecessor.SubjectType != value.SubjectType || predecessor.SubjectID != value.SubjectID || predecessor.Origin == nil || predecessor.Origin.Type != value.Origin.Type || predecessor.Origin.ID != value.Origin.ID || predecessor.Origin.Sequence != value.Origin.Sequence-1 {
		return errors.Join(ErrVersionConflict, fmt.Errorf("request predecessor does not match the tenant, subject and previous origin sequence"))
	}
	return nil
}

func requestImmutableFingerprint(value Request) ([32]byte, error) {
	immutable := struct {
		SubjectType  string
		SubjectID    string
		Title        string
		Purpose      string
		WhyYou       string
		Sensitivity  string
		AudienceType string
		Recipient    struct {
			Type         RecipientType
			PrincipalID  string
			AudienceHint string
			AudienceHash []byte
		}
		EstimatedMinutes      int
		Deadline              time.Time
		KnownFacts            map[string]string
		Fields                []Field
		SourceBindings        []RequestBindingReference
		FormTemplateID        string
		FormTemplateVersion   int64
		CollectionPeriodStart *time.Time
		CollectionPeriodEnd   *time.Time
		CreatedBy             string
		Origin                *RequestOrigin
		PredecessorRequestID  string
	}{
		SubjectType: value.SubjectType, SubjectID: value.SubjectID, Title: value.Title,
		Purpose: value.Purpose, WhyYou: value.WhyYou, Sensitivity: value.Sensitivity, AudienceType: value.AudienceType,
		EstimatedMinutes: value.EstimatedMinutes, Deadline: value.Deadline.UTC(), KnownFacts: value.KnownFacts,
		Fields: value.Fields, SourceBindings: value.SourceBindings, FormTemplateID: value.FormTemplateID, FormTemplateVersion: value.FormTemplateVersion,
		CollectionPeriodStart: value.CollectionPeriodStart, CollectionPeriodEnd: value.CollectionPeriodEnd, CreatedBy: value.CreatedBy,
		Origin: value.Origin, PredecessorRequestID: value.PredecessorRequestID,
	}
	immutable.Recipient.Type = value.Recipient.Type
	immutable.Recipient.PrincipalID = value.Recipient.PrincipalID
	immutable.Recipient.AudienceHint = value.Recipient.AudienceHint
	immutable.Recipient.AudienceHash = append([]byte(nil), value.Recipient.AudienceHash...)
	encoded, err := json.Marshal(immutable)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(encoded), nil
}

func sameImmutableRequest(left, right Request) bool {
	leftFingerprint, leftErr := requestImmutableFingerprint(left)
	rightFingerprint, rightErr := requestImmutableFingerprint(right)
	return leftErr == nil && rightErr == nil && leftFingerprint == rightFingerprint
}

func cloneRequestOrigin(value *RequestOrigin) *RequestOrigin {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func requestOriginKey(tenant string, origin RequestOrigin) string {
	return tenant + "\x00" + string(origin.Type) + "\x00" + origin.ID + fmt.Sprintf("\x00%d", origin.Sequence)
}

func (r *MemoryRepository) GetRequestByOrigin(_ context.Context, tenant string, origin RequestOrigin) (Request, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	requestID, ok := r.requestOrigins[requestOriginKey(tenant, origin)]
	if !ok {
		return Request{}, ErrNotFound
	}
	value, ok := r.requests[requestID]
	if !ok || value.TenantID != tenant {
		return Request{}, ErrNotFound
	}
	return cloneRequest(value), nil
}

func (r *MemoryRepository) createRequestLocked(value Request) (Request, error) {
	if !value.Deadline.After(value.CreatedAt) {
		return Request{}, ErrRequestClosed
	}
	if err := validateRequestOrigin(value.Origin, value.PredecessorRequestID); err != nil {
		return Request{}, err
	}
	if value.Origin != nil {
		key := requestOriginKey(value.TenantID, *value.Origin)
		if requestID, exists := r.requestOrigins[key]; exists {
			existing := r.requests[requestID]
			if sameImmutableRequest(existing, value) {
				return cloneRequest(existing), nil
			}
			return Request{}, ErrVersionConflict
		}
		if value.Origin.Sequence > 1 {
			predecessor, exists := r.requests[value.PredecessorRequestID]
			if !exists || predecessor.TenantID != value.TenantID || predecessor.SubjectType != value.SubjectType || predecessor.SubjectID != value.SubjectID || predecessor.Origin == nil || predecessor.Origin.Type != value.Origin.Type || predecessor.Origin.ID != value.Origin.ID || predecessor.Origin.Sequence != value.Origin.Sequence-1 {
				return Request{}, ErrVersionConflict
			}
		}
		r.requestOrigins[key] = value.ID
	}
	if _, exists := r.requests[value.ID]; exists {
		return Request{}, ErrVersionConflict
	}
	value.KnownFacts = cloneMap(value.KnownFacts)
	value.Fields = cloneFields(value.Fields)
	value.SourceBindings = cloneRequestBindings(value.SourceBindings)
	value.Recipient = cloneRecipient(value.Recipient)
	value.Origin = cloneRequestOrigin(value.Origin)
	r.requests[value.ID] = value
	return cloneRequest(value), nil
}
