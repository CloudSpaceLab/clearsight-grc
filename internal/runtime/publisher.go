package runtime

import "context"

type CompositePublisher struct {
	publishers []Publisher
}

func NewCompositePublisher(publishers ...Publisher) *CompositePublisher {
	filtered := make([]Publisher, 0, len(publishers))
	for _, publisher := range publishers {
		if publisher != nil {
			filtered = append(filtered, publisher)
		}
	}
	return &CompositePublisher{publishers: filtered}
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
