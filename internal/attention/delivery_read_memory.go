package attention

import (
	"context"
	"time"
)

type MemoryDeliveryReader struct{}

func NewMemoryDeliveryReader() *MemoryDeliveryReader { return &MemoryDeliveryReader{} }

func (*MemoryDeliveryReader) Health(_ context.Context, _ string, asOf time.Time, window time.Duration) (DeliveryHealth, error) {
	asOf = asOf.UTC()
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	return DeliveryHealth{
		AsOf: asOf, WindowStart: asOf.Add(-window),
		Classes: []DeliveryClassHealth{}, Failures: []DeliveryFailureSummary{},
	}, nil
}

func (*MemoryDeliveryReader) RecordHistory(_ context.Context, _, _, _, _ string, _ int) (RecordNotificationHistory, error) {
	return RecordNotificationHistory{Items: []RecordNotificationEvent{}, AsOf: time.Now().UTC()}, nil
}

var _ DeliveryReader = (*MemoryDeliveryReader)(nil)
