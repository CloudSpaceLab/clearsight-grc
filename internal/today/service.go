package today

import (
	"context"
	"sort"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type Loader func(context.Context, identity.Actor) ([]AttentionItem, error)

type Service struct {
	items  []AttentionItem
	loader Loader
}

func NewService(items []AttentionItem) *Service {
	return &Service{items: append([]AttentionItem(nil), items...)}
}

func NewDynamicService(loader Loader) *Service {
	return &Service{loader: loader}
}

func (s *Service) List() []AttentionItem {
	items := append([]AttentionItem{}, s.items...)
	sortAttention(items)
	return items
}

func (s *Service) ListFor(ctx context.Context, actor identity.Actor) ([]AttentionItem, error) {
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
