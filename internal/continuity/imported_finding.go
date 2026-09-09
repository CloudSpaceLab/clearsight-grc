package continuity

import (
	"context"
	"time"
)

// ImportedFinding is a complete command bundle, prepared by the import service
// after source, current actor, ownership and relationship validation.
type ImportedFinding struct {
	Matter Matter
	Action Action
	Events []Event
}

func BuildImportedFinding(matter Matter, action Action, actor string, at time.Time) (ImportedFinding, error) {
	matter.Type = MatterVendorDeficiency
	matter.Status = MatterInitialReview
	matter.Version = 1
	matter.Reference = matterReference(matter.ID)
	matter.CreatedAt = at
	matter.UpdatedAt = at
	action.TenantID = matter.TenantID
	action.MatterID = matter.ID
	action.Status = ActionPlanned
	action.Version = 1
	action.CreatedAt = at
	action.UpdatedAt = at
	action.RequiredResponsibility = "PERFORMER"
	created, err := newEvent(matter.TenantID, "MATTER", matter.ID, 1, EventMatterCreated, matter, actorFor(actor), actor, at)
	if err != nil {
		return ImportedFinding{}, err
	}
	added, err := newEvent(matter.TenantID, "MATTER", matter.ID, 2, EventActionAdded, action, actorFor(actor), actor, at)
	if err != nil {
		return ImportedFinding{}, err
	}
	return ImportedFinding{Matter: matter, Action: action, Events: []Event{created, added}}, nil
}

// StoreImportedFindings commits the memory projection and linked records under
// one write lock. The callback validates its complete batch before changing it.
func (r *MemoryRepository) StoreImportedFindings(ctx context.Context, items []ImportedFinding, links func() error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	aggregates := make([]MatterAggregate, len(items))
	for i, item := range items {
		if !r.visibleLegalEntity(ctx, item.Matter.TenantID, item.Matter.LegalEntityID) {
			return ErrNotFound
		}
		if _, ok := r.matters[item.Matter.TenantID][item.Matter.ID]; ok {
			return ErrDuplicate
		}
		aggregate, err := reconstructMatter(item.Events)
		if err != nil {
			return err
		}
		aggregates[i] = aggregate
	}
	if err := links(); err != nil {
		return err
	}
	for i, item := range items {
		tenant := item.Matter.TenantID
		if r.matters[tenant] == nil {
			r.matters[tenant] = map[string]MatterAggregate{}
		}
		if r.matterEvents[tenant] == nil {
			r.matterEvents[tenant] = map[string][]Event{}
		}
		r.matters[tenant][item.Matter.ID] = aggregates[i]
		r.matterEvents[tenant][item.Matter.ID] = append([]Event(nil), item.Events...)
	}
	return nil
}
