package thirdparty

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (r *MemoryAssessmentRepository) MaintainVendorRefresh(_ context.Context, now time.Time, policy RefreshMaintenancePolicy) (RefreshBatchReceipt, error) {
	if !validRefreshMaintenancePolicy(policy) {
		return RefreshBatchReceipt{}, ErrInvalidRefreshMaintenancePolicy
	}
	now = now.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	factCutoff := now.Add(-policy.FactConfirmationInterval)
	leadDate := today.Add(policy.DocumentLead)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()

	eligible := make([]string, 0)
	for relationshipID, relationship := range r.relationships {
		vendor, exists := r.vendors[relationship.VendorID]
		if !exists || vendor.TenantID != relationship.TenantID {
			continue
		}
		due := !vendor.UpdatedAt.After(factCutoff)
		for _, documents := range r.assessmentDocuments {
			for _, document := range documents {
				if document.RelationshipID == relationshipID && document.Status == AssessmentDocumentValidated && document.ExpiresOn != nil && !calendarDay(document.ExpiresOn.UTC()).After(leadDate) {
					due = true
				}
			}
		}
		if due {
			eligible = append(eligible, relationshipID)
		}
	}
	sort.Strings(eligible)
	if len(eligible) > policy.BatchSize {
		eligible = eligible[:policy.BatchSize]
	}
	receipt := RefreshBatchReceipt{RelationshipsExamined: len(eligible)}
	for _, relationshipID := range eligible {
		relationship := r.relationships[relationshipID]
		vendor := r.vendors[relationship.VendorID]
		candidate := RefreshCandidate{Scope: Scope{TenantID: relationship.TenantID, LegalEntityID: relationship.LegalEntityID}, RelationshipID: relationshipID, ObservedVersions: map[string]int64{}}
		reasons := []string{}
		if !vendor.UpdatedAt.After(factCutoff) {
			candidate.TargetKeys = append(candidate.TargetKeys, refreshIdentityTargetKeys...)
			for _, key := range refreshIdentityTargetKeys {
				candidate.ObservedVersions[key] = vendor.Version
			}
			reasons = append(reasons, "HELD_VENDOR_FACTS_CONFIRMATION_DUE")
		}
		for assessmentID, documents := range r.assessmentDocuments {
			for artifactID, document := range documents {
				if document.RelationshipID != relationshipID || document.Status != AssessmentDocumentValidated || document.ExpiresOn == nil || calendarDay(document.ExpiresOn.UTC()).After(leadDate) {
					continue
				}
				targetKey := "VENDOR.DOCUMENT." + strings.ToUpper(strings.TrimSpace(document.DocumentType))
				candidate.TargetKeys = append(candidate.TargetKeys, targetKey)
				candidate.ObservedVersions[targetKey] = document.Version
				reasons = append(reasons, "VENDOR_DOCUMENT_EXPIRY_DUE")
				if !calendarDay(document.ExpiresOn.UTC()).After(today) {
					document.Status = AssessmentDocumentExpired
					document.Version++
					document.UpdatedAt = now
					documents[artifactID] = document
					r.assessmentDocuments[assessmentID] = documents
					receipt.DocumentsExpired++
					if assessment, exists := r.assessments[assessmentID]; exists {
						assessment.Version++
						assessment.UpdatedAt = now
						r.assessments[assessmentID] = assessment
						r.appendMemoryAssessmentAudit(assessment, "", "AssessmentDocumentExpired")
					}
				}
			}
		}
		candidate.TargetKeys = uniqueSortedStrings(candidate.TargetKeys)
		candidate.Reason = strings.Join(uniqueSortedStrings(reasons), "+")
		if len(candidate.TargetKeys) == 0 || r.hasCoveringRefreshAttention(candidate) {
			continue
		}
		attentionID, err := id.NewUUIDv7()
		if err != nil {
			return receipt, err
		}
		attention := RefreshAttention{ID: attentionID, Scope: candidate.Scope, RelationshipID: relationshipID, OwnerPrincipalID: relationship.BusinessOwnerPrincipalID, TargetKeys: append([]string(nil), candidate.TargetKeys...), Reason: candidate.Reason, ObservedVersions: cloneVersionMap(candidate.ObservedVersions), DedupeKey: refreshDedupeKey(candidate), State: RefreshAttentionOpen, CreatedAt: now, UpdatedAt: now, Version: 1}
		r.refreshAttentions[attention.DedupeKey] = attention
		event := RefreshAttentionEvent{AttentionID: attention.ID, RelationshipID: relationshipID, EventType: "VendorRefreshAttentionCreated", OccurredAt: now}
		r.refreshEvents = append(r.refreshEvents, event)
		r.refreshOutbox = append(r.refreshOutbox, event)
		receipt.AttentionsCreated++
	}
	return receipt, nil
}

func (r *MemoryAssessmentRepository) hasCoveringRefreshAttention(candidate RefreshCandidate) bool {
	for _, attention := range r.refreshAttentions {
		if attention.State != RefreshAttentionOpen || attention.TenantID != candidate.TenantID || attention.LegalEntityID != candidate.LegalEntityID || attention.RelationshipID != candidate.RelationshipID {
			continue
		}
		covers := true
		for _, key := range candidate.TargetKeys {
			if attention.ObservedVersions[key] != candidate.ObservedVersions[key] {
				covers = false
				break
			}
		}
		if covers {
			return true
		}
	}
	return false
}

func calendarDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
func uniqueSortedStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
func cloneVersionMap(values map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
