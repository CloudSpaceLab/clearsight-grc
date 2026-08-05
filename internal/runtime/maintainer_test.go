package runtime

import (
	"context"
	"testing"
	"time"
)

type countingMaintainer struct{ count int }

func (m *countingMaintainer) Maintain(context.Context, time.Time, int) (int, error) {
	m.count++
	return 1, nil
}

func TestTickRunsMaintainers(t *testing.T) {
	repo := NewMemoryRepository()
	maintainer := &countingMaintainer{}
	service := NewService(repo, nil, &countingPublisher{}, "worker")
	service.AddMaintainer(maintainer)
	service.now = func() time.Time { return time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC) }
	if err := service.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if maintainer.count != 1 {
		t.Fatalf("expected one maintenance pass, got %d", maintainer.count)
	}
}
