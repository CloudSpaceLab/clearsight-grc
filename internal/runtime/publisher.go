package runtime

import (
	"context"
	"reflect"
)

type CompositePublisher struct {
	publishers []Publisher
}

func NewCompositePublisher(publishers ...Publisher) *CompositePublisher {
	filtered := make([]Publisher, 0, len(publishers))
	for _, publisher := range publishers {
		if publisher != nil && !isNilPublisher(publisher) {
			filtered = append(filtered, publisher)
		}
	}
	return &CompositePublisher{publishers: filtered}
}

func isNilPublisher(publisher Publisher) bool {
	value := reflect.ValueOf(publisher)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (p *CompositePublisher) Publish(ctx context.Context, event OutboxEvent) error {
	if p == nil {
		return nil
	}
	for _, publisher := range p.publishers {
		if err := publisher.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
