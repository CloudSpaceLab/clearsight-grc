package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

const OriginMonitoringCollection = "MONITORING_COLLECTION"

func validateRequestLineage(origin RequestOrigin, predecessorID string) error {
	origin = origin.normalized()
	predecessorID = strings.TrimSpace(predecessorID)
	if err := origin.validate(); err != nil {
		return err
	}
	if origin.empty() {
		if predecessorID != "" {
			return errors.Join(ErrVersionConflict, fmt.Errorf("predecessor requires a request origin"))
		}
		return nil
	}
	if origin.Type != OriginMonitoringCollection {
		if predecessorID != "" {
			return errors.Join(ErrVersionConflict, fmt.Errorf("predecessor is only supported for monitoring collection requests"))
		}
		return nil
	}
	if (origin.Version == 1) != (predecessorID == "") {
		return errors.Join(ErrVersionConflict, fmt.Errorf("request predecessor does not match the origin version"))
	}
	return nil
}

func validateRequestPredecessor(ctx context.Context, repo Repository, value Request) error {
	if err := validateRequestLineage(value.Origin, value.PredecessorRequestID); err != nil || value.Origin.Type != OriginMonitoringCollection || value.Origin.Version == 1 {
		return err
	}
	predecessor, err := repo.GetRequest(ctx, value.TenantID, value.PredecessorRequestID)
	if err != nil {
		return errors.Join(ErrVersionConflict, err)
	}
	if predecessor.TenantID != value.TenantID || predecessor.SubjectType != value.SubjectType || predecessor.SubjectID != value.SubjectID || predecessor.Origin.Type != value.Origin.Type || predecessor.Origin.ID != value.Origin.ID || predecessor.Origin.Version != value.Origin.Version-1 {
		return errors.Join(ErrVersionConflict, fmt.Errorf("request predecessor does not match the tenant, subject and previous origin version"))
	}
	return nil
}

func requestImmutableFingerprint(value Request) ([32]byte, error) {
	immutable := struct {
		LegalEntityID         string
		SubjectType           string
		SubjectID             string
		Title                 string
		Purpose               string
		WhyYou                string
		Sensitivity           string
		AudienceType          string
		Recipient             Recipient
		EstimatedMinutes      int
		Deadline              time.Time
		KnownFacts            map[string]string
		Presentation          formcontract.Presentation
		ScoringMode           formcontract.ScoringMode
		ScoreProfile          *formcontract.ScoreProfile
		Sections              []formcontract.Section
		Fields                []Field
		SourceBindings        []RequestBindingReference
		FormTemplateID        string
		FormTemplateVersion   int64
		CollectionPeriodStart *time.Time
		CollectionPeriodEnd   *time.Time
		Origin                RequestOrigin
		PredecessorRequestID  string
		CreatedBy             string
	}{
		LegalEntityID: value.LegalEntityID, SubjectType: value.SubjectType, SubjectID: value.SubjectID,
		Title: value.Title, Purpose: value.Purpose, WhyYou: value.WhyYou, Sensitivity: value.Sensitivity, AudienceType: value.AudienceType,
		Recipient: value.Recipient, EstimatedMinutes: value.EstimatedMinutes, Deadline: value.Deadline.UTC(), KnownFacts: value.KnownFacts,
		Presentation: value.Presentation, ScoringMode: value.ScoringMode, ScoreProfile: value.ScoreProfile, Sections: value.Sections,
		Fields: value.Fields, SourceBindings: value.SourceBindings, FormTemplateID: value.FormTemplateID, FormTemplateVersion: value.FormTemplateVersion,
		CollectionPeriodStart: value.CollectionPeriodStart, CollectionPeriodEnd: value.CollectionPeriodEnd,
		Origin: value.Origin.normalized(), PredecessorRequestID: strings.TrimSpace(value.PredecessorRequestID), CreatedBy: value.CreatedBy,
	}
	immutable.Recipient.DisplayName = ""
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

func (r *MemoryRepository) createRequestLocked(value Request) (Request, error) {
	if !value.Deadline.After(value.CreatedAt) {
		return Request{}, ErrRequestClosed
	}
	value.Origin = value.Origin.normalized()
	value.PredecessorRequestID = strings.TrimSpace(value.PredecessorRequestID)
	if err := validateRequestLineage(value.Origin, value.PredecessorRequestID); err != nil {
		return Request{}, err
	}
	if !value.Origin.empty() {
		for _, existing := range r.requests {
			if existing.TenantID == value.TenantID && existing.Origin == value.Origin {
				if sameImmutableRequest(existing, value) {
					return cloneRequest(existing), nil
				}
				return Request{}, ErrVersionConflict
			}
		}
	}
	if value.Origin.Type == OriginMonitoringCollection && value.Origin.Version > 1 {
		predecessor, exists := r.requests[value.PredecessorRequestID]
		if !exists || predecessor.TenantID != value.TenantID || predecessor.SubjectType != value.SubjectType || predecessor.SubjectID != value.SubjectID || predecessor.Origin.Type != value.Origin.Type || predecessor.Origin.ID != value.Origin.ID || predecessor.Origin.Version != value.Origin.Version-1 {
			return Request{}, ErrVersionConflict
		}
	}
	if _, exists := r.requests[value.ID]; exists {
		return Request{}, ErrVersionConflict
	}
	value.KnownFacts = cloneMap(value.KnownFacts)
	value.Sections = cloneSections(value.Sections)
	value.Fields = cloneFields(value.Fields)
	value.SourceBindings = cloneRequestBindings(value.SourceBindings)
	value.Recipient = cloneRecipient(value.Recipient)
	r.requests[value.ID] = value
	return cloneRequest(value), nil
}
