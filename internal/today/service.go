package today

import (
	"context"
	"sort"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type Loader func(context.Context, identity.Actor) ([]AttentionItem, error)

// ActorWorkSnapshot retains the verified actor's bounded work feed and its
// workflow-only completeness signal. It is not a total across other sources.
type ActorWorkSnapshot struct {
	Items           []AttentionItem
	WorkflowHasMore *bool
}

type SnapshotLoader func(context.Context, identity.Actor) (ActorWorkSnapshot, error)

type Service struct {
	items          []AttentionItem
	loader         Loader
	snapshotLoader SnapshotLoader
}

func NewService(items []AttentionItem) *Service {
	return &Service{items: append([]AttentionItem(nil), items...)}
}

func NewDynamicService(loader Loader) *Service {
	return &Service{loader: loader}
}

func NewSnapshotService(loader SnapshotLoader) *Service {
	return &Service{snapshotLoader: loader}
}

func (s *Service) List() []AttentionItem {
	items := append([]AttentionItem{}, s.items...)
	sortAttention(items)
	return items
}

func (s *Service) ListFor(ctx context.Context, actor identity.Actor) ([]AttentionItem, error) {
	if s != nil && s.snapshotLoader != nil {
		value, err := s.SnapshotFor(ctx, actor)
		return value.Items, err
	}
	if s != nil && s.loader != nil {
		items, err := s.loader(ctx, actor)
		if err != nil {
			return nil, err
		}
		items = append([]AttentionItem(nil), items...)
		sortAttention(items)
		return items, nil
	}
	if s == nil {
		return []AttentionItem{}, nil
	}
	return s.List(), nil
}

func (s *Service) SnapshotFor(ctx context.Context, actor identity.Actor) (ActorWorkSnapshot, error) {
	if s != nil && s.snapshotLoader != nil {
		value, err := s.snapshotLoader(ctx, actor)
		if err != nil {
			return ActorWorkSnapshot{}, err
		}
		value.Items = append([]AttentionItem{}, value.Items...)
		sortAttention(value.Items)
		return value, nil
	}
	items, err := s.ListFor(ctx, actor)
	if err != nil {
		return ActorWorkSnapshot{}, err
	}
	return ActorWorkSnapshot{Items: items}, nil
}

func sortAttention(items []AttentionItem) {
	sort.Slice(items, func(i, j int) bool {
		leftNoDeadline := items[i].DueAt.IsZero()
		rightNoDeadline := items[j].DueAt.IsZero()
		if leftNoDeadline != rightNoDeadline {
			return !leftNoDeadline
		}
		if items[i].DueAt.Equal(items[j].DueAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].DueAt.Before(items[j].DueAt)
	})
}
