package monitoring

import (
	"context"
	"sort"
	"strings"
	"time"
)

type FormSubjectLinkEvent struct {
	LinkID    string
	Type      string
	ActorID   string
	CreatedAt time.Time
	Reason    string
}

func (r *MemoryRepository) CreateFormSubjectLink(_ context.Context, input FormSubjectLink) (FormSubjectLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	form, ok := r.forms[revisionKey(input.TenantID, input.FormTemplateID, input.FormTemplateVersion)]
	if !ok || form.LegalEntityID != input.LegalEntityID || form.Status != LifecycleActive || !form.IsCurrent {
		return FormSubjectLink{}, ErrNotFound
	}
	for _, link := range r.subjectLinks {
		if link.TenantID == input.TenantID && link.LegalEntityID == input.LegalEntityID &&
			link.SubjectType == input.SubjectType && link.SubjectID == input.SubjectID &&
			link.FormTemplateID == input.FormTemplateID && link.EndedAt == nil {
			return link, nil
		}
	}
	input.FormName, input.FormStatus = form.Name, form.Status
	r.subjectLinks[input.ID] = input
	r.subjectLinkEvents = append(r.subjectLinkEvents, FormSubjectLinkEvent{
		LinkID: input.ID, Type: "LINKED", ActorID: input.LinkedBy, CreatedAt: input.LinkedAt,
	})
	return input, nil
}

func (r *MemoryRepository) ListFormSubjectLinks(_ context.Context, tenantID, legalEntityID, subjectType, subjectID, cursor string, limit int) (FormSubjectLinkPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var values []FormSubjectLink
	for _, link := range r.subjectLinks {
		if link.TenantID != tenantID || link.LegalEntityID != legalEntityID ||
			link.SubjectType != subjectType || link.SubjectID != subjectID || link.EndedAt != nil {
			continue
		}
		if cursor != "" && strings.Compare(link.ID, cursor) >= 0 {
			continue
		}
		if form, ok := r.forms[revisionKey(tenantID, link.FormTemplateID, link.FormTemplateVersion)]; ok {
			link.FormName, link.FormStatus = form.Name, form.Status
		}
		values = append(values, link)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID > values[j].ID })
	page := FormSubjectLinkPage{Items: values}
	if len(values) > limit {
		page.Items = values[:limit]
		page.NextCursor = values[limit-1].ID
	}
	if page.Items == nil {
		page.Items = []FormSubjectLink{}
	}
	return page, nil
}

func (r *MemoryRepository) EndFormSubjectLink(_ context.Context, tenantID, legalEntityID, subjectType, subjectID, linkID string, version int64, actorID string, at time.Time, reason string) (FormSubjectLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	link, ok := r.subjectLinks[linkID]
	if !ok || link.TenantID != tenantID || link.LegalEntityID != legalEntityID ||
		link.SubjectType != subjectType || link.SubjectID != subjectID {
		return FormSubjectLink{}, ErrNotFound
	}
	if link.EndedAt != nil || link.Version != version {
		return FormSubjectLink{}, ErrConflict
	}
	link.EndedAt = &at
	link.Version++
	r.subjectLinks[linkID] = link
	r.subjectLinkEvents = append(r.subjectLinkEvents, FormSubjectLinkEvent{
		LinkID: link.ID, Type: "UNLINKED", ActorID: actorID, CreatedAt: at, Reason: reason,
	})
	return link, nil
}
