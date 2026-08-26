package thirdparty

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type MemoryVendorWorkRepository struct {
	mu        sync.RWMutex
	work      map[string]VendorWorkRequest
	captures  map[string][]VendorWorkCaptureLink
	reactions map[string]bool
}

func NewMemoryVendorWorkRepository() *MemoryVendorWorkRepository {
	return &MemoryVendorWorkRepository{work: map[string]VendorWorkRequest{}, captures: map[string][]VendorWorkCaptureLink{}, reactions: map[string]bool{}}
}

func (r *MemoryVendorWorkRepository) CreateVendorWork(_ context.Context, value VendorWorkRequest) (VendorWorkRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.work[value.ID]; exists {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	for _, current := range r.work {
		if current.TenantID == value.TenantID && current.LegalEntityID == value.LegalEntityID && current.RelationshipLinkID == value.RelationshipLinkID && current.State != VendorWorkAccepted && current.State != VendorWorkCancelled {
			return VendorWorkRequest{}, ErrVersionConflict
		}
	}
	r.work[value.ID] = value
	return value, nil
}

func (r *MemoryVendorWorkRepository) GetVendorWork(_ context.Context, scope Scope, id string) (VendorWorkRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.work[id]
	if !ok || value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID {
		return VendorWorkRequest{}, ErrNotFound
	}
	return value, nil
}

func (r *MemoryVendorWorkRepository) FindActiveVendorWork(_ context.Context, scope Scope, linkID string) (VendorWorkRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.work {
		if value.TenantID == scope.TenantID && value.LegalEntityID == scope.LegalEntityID && value.RelationshipLinkID == linkID && value.State != VendorWorkAccepted && value.State != VendorWorkCancelled {
			return value, nil
		}
	}
	return VendorWorkRequest{}, ErrNotFound
}

func (r *MemoryVendorWorkRepository) AttachVendorWorkCapture(_ context.Context, scope Scope, id string, expected int64, link VendorWorkCaptureLink, now time.Time) (VendorWorkRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.work[id]
	if !ok || value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID {
		return VendorWorkRequest{}, ErrNotFound
	}
	if value.Version != expected {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	if value.State != VendorWorkPreparing && value.State != VendorWorkUnderReview {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	for _, current := range r.captures[id] {
		if current.Sequence == link.Sequence || current.RequestID == link.RequestID {
			return VendorWorkRequest{}, ErrVersionConflict
		}
	}
	value.CurrentRequestID, value.CurrentInvitationID, value.CurrentCaptureSequence, value.SubmissionID = link.RequestID, "", link.Sequence, ""
	value.DeliveryState, value.Recovery = VendorWorkDeliveryNotSent, ""
	value.Version++
	value.UpdatedAt = now.UTC()
	r.work[id] = value
	r.captures[id] = append(r.captures[id], link)
	return value, nil
}

func (r *MemoryVendorWorkRepository) MarkVendorWorkSent(_ context.Context, scope Scope, id string, expected int64, invitationID string, delivery VendorWorkDeliveryState, recovery string, now time.Time) (VendorWorkRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.work[id]
	if !ok || value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID {
		return VendorWorkRequest{}, ErrNotFound
	}
	if value.Version != expected {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	if value.State != VendorWorkPreparing && value.State != VendorWorkAwaitingVendor && value.State != VendorWorkChangesRequested {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	if invitationID != "" {
		value.CurrentInvitationID = invitationID
		if value.State == VendorWorkPreparing {
			value.State = VendorWorkAwaitingVendor
		}
		links := r.captures[id]
		for index := range links {
			if links[index].RequestID == value.CurrentRequestID {
				links[index].InvitationID = invitationID
			}
		}
		r.captures[id] = links
	}
	value.DeliveryState, value.Recovery, value.UpdatedAt = delivery, recovery, now.UTC()
	value.Version++
	r.work[id] = value
	return value, nil
}

func (r *MemoryVendorWorkRepository) MarkVendorWorkPreparationRequired(_ context.Context, scope Scope, id string, expected int64, recovery string, now time.Time) (VendorWorkRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.work[id]
	if !ok || value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID {
		return VendorWorkRequest{}, ErrNotFound
	}
	if value.Version != expected {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	if value.State != VendorWorkPreparing || value.CurrentRequestID != "" {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	value.DeliveryState, value.Recovery, value.Version, value.UpdatedAt = VendorWorkDeliveryRetryRequired, recovery, value.Version+1, now.UTC()
	r.work[id] = value
	return value, nil
}

func (r *MemoryVendorWorkRepository) RecordVendorWorkSubmission(_ context.Context, input VendorWorkSubmissionInput, now time.Time) (VendorWorkRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := input.TenantID + "\x00" + input.CausationID
	if r.reactions[key] {
		value, ok := r.work[input.WorkRequestID]
		if !ok || value.TenantID != input.TenantID {
			return VendorWorkRequest{}, ErrNotFound
		}
		return value, nil
	}
	value, ok := r.work[input.WorkRequestID]
	if !ok || value.TenantID != input.TenantID || value.CurrentRequestID != input.RequestID {
		return VendorWorkRequest{}, ErrNotFound
	}
	if value.State != VendorWorkAwaitingVendor && value.State != VendorWorkChangesRequested {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	value.State, value.SubmissionID, value.DeliveryState, value.Recovery = VendorWorkResponseReceived, input.SubmissionID, VendorWorkDeliveryDelivered, ""
	value.ResponseReceivedAt = timePointer(now.UTC())
	value.UpdatedAt = now.UTC()
	value.Version++
	links := r.captures[value.ID]
	for index := range links {
		if links[index].RequestID == input.RequestID {
			links[index].SubmissionID = input.SubmissionID
		}
	}
	r.captures[value.ID] = links
	r.work[value.ID] = value
	r.reactions[key] = true
	return value, nil
}

func (r *MemoryVendorWorkRepository) TransitionVendorWork(_ context.Context, scope Scope, id string, expected int64, target VendorWorkState, actor, detail string, now time.Time) (VendorWorkRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.work[id]
	if !ok || value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID {
		return VendorWorkRequest{}, ErrNotFound
	}
	if value.Version != expected {
		return VendorWorkRequest{}, ErrVersionConflict
	}
	valid := false
	switch target {
	case VendorWorkUnderReview:
		valid = value.State == VendorWorkResponseReceived
		if valid {
			value.ReviewerPrincipalID = actor
			value.ReviewStartedAt = timePointer(now.UTC())
		}
	case VendorWorkChangesRequested:
		valid = value.State == VendorWorkUnderReview
		if valid {
			value.ReviewRationale = detail
		}
	case VendorWorkAccepted:
		valid = value.State == VendorWorkUnderReview
		if valid {
			value.ReviewerPrincipalID, value.ReviewRationale, value.AcceptedAt = actor, detail, timePointer(now.UTC())
		}
	case VendorWorkCancelled:
		valid = value.State != VendorWorkAccepted && value.State != VendorWorkCancelled
		if valid {
			value.CancellationReason, value.CancelledAt = detail, timePointer(now.UTC())
		}
	}
	if !valid {
		return VendorWorkRequest{}, ErrInvalidAssessmentTransition
	}
	value.State, value.UpdatedAt = target, now.UTC()
	value.Version++
	r.work[id] = value
	return value, nil
}

func (r *MemoryVendorWorkRepository) ListVendorWork(_ context.Context, scope Scope, input VendorWorkListInput) (VendorWorkPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := []VendorWorkRequest{}
	for _, value := range r.work {
		if value.TenantID != scope.TenantID || value.LegalEntityID != scope.LegalEntityID || (input.RelationshipID != "" && value.RelationshipID != input.RelationshipID) || (input.TargetType != "" && value.TargetType != input.TargetType) || (input.TargetID != "" && value.TargetID != input.TargetID) {
			continue
		}
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].UpdatedAt.Equal(values[j].UpdatedAt) {
			return strings.Compare(values[i].ID, values[j].ID) > 0
		}
		return values[i].UpdatedAt.After(values[j].UpdatedAt)
	})
	page := VendorWorkPage{Items: values}
	if len(page.Items) > input.Limit {
		page.Items = page.Items[:input.Limit]
	}
	return page, nil
}

func (r *MemoryVendorWorkRepository) ResolveVendorWorkCapture(_ context.Context, tenant string, origin evidence.RequestOrigin, requestID string) (VendorWorkSubmissionTarget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.work[origin.ID]
	if !ok || value.TenantID != tenant {
		return VendorWorkSubmissionTarget{}, ErrNotFound
	}
	for _, link := range r.captures[value.ID] {
		if link.RequestID == requestID && link.OriginVersion == origin.Version {
			return VendorWorkSubmissionTarget{Scope: Scope{TenantID: value.TenantID, LegalEntityID: value.LegalEntityID}, WorkRequestID: value.ID, WorkVersion: value.Version, RequestID: requestID}, nil
		}
	}
	return VendorWorkSubmissionTarget{}, ErrNotFound
}

func (r *MemoryVendorWorkRepository) HasActiveVendorWork(_ context.Context, scope Scope, linkID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.work {
		if value.TenantID == scope.TenantID && value.LegalEntityID == scope.LegalEntityID && value.RelationshipLinkID == linkID && value.State != VendorWorkAccepted && value.State != VendorWorkCancelled {
			return true, nil
		}
	}
	return false, nil
}

func timePointer(value time.Time) *time.Time { copy := value; return &copy }

var _ VendorWorkRepository = (*MemoryVendorWorkRepository)(nil)
