package runtime

import (
	"context"
	"errors"
	"testing"
	"time"
)

type orderedPublisher struct {
	name  string
	calls *[]string
	err   error
}

type optionalPublisher struct{}

func (*optionalPublisher) Publish(context.Context, OutboxEvent) error {
	return errors.New("typed nil publisher was called")
}

func (p orderedPublisher) Publish(context.Context, OutboxEvent) error {
	*p.calls = append(*p.calls, p.name)
	return p.err
}

func TestCompositePublisherStopsBeforeMarkingLaterSinksDelivered(t *testing.T) {
	calls := []string{}
	publisher := NewCompositePublisher(
		orderedPublisher{name: "internal", calls: &calls},
		orderedPublisher{name: "external", calls: &calls, err: errors.New("unavailable")},
		orderedPublisher{name: "later", calls: &calls},
	)
	if err := publisher.Publish(context.Background(), OutboxEvent{ID: "event-1"}); err == nil {
		t.Fatal("expected publisher failure")
	}
	if len(calls) != 2 || calls[0] != "internal" || calls[1] != "external" {
		t.Fatalf("unexpected publisher order: %#v", calls)
	}
}

func TestCompositePublisherSkipsTypedNilOptionalSink(t *testing.T) {
	var optional *optionalPublisher
	calls := []string{}
	publisher := NewCompositePublisher(optional, orderedPublisher{name: "stored", calls: &calls})

	if err := publisher.Publish(context.Background(), OutboxEvent{ID: "event-1"}); err != nil {
		t.Fatalf("publish with disabled optional sink: %v", err)
	}
	if len(calls) != 1 || calls[0] != "stored" {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestInboxProcessedReflectsReceipt(t *testing.T) {
	repo := NewMemoryRepository()
	processed, err := repo.InboxProcessed(context.Background(), "t", "c", "e")
	if err != nil || processed {
		t.Fatalf("unexpected initial inbox state: %v processed=%v", err, processed)
	}
	if _, err := repo.RecordInbox(context.Background(), "t", "c", "e", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	processed, err = repo.InboxProcessed(context.Background(), "t", "c", "e")
	if err != nil || !processed {
		t.Fatalf("receipt not visible: %v processed=%v", err, processed)
	}
}
