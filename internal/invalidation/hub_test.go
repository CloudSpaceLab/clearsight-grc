package invalidation

import (
	"fmt"
	"testing"
	"time"
)

func TestHubDeliversOnlyExactActorScope(t *testing.T) {
	hub := NewHub()
	first, cancelFirst := hub.Subscribe(Scope{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "person-a"})
	defer cancelFirst()
	second, cancelSecond := hub.Subscribe(Scope{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "person-b"})
	defer cancelSecond()

	hub.Publish(Event{TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "person-a", Revision: "rev-1"})

	select {
	case event := <-first:
		if event.Revision != "rev-1" {
			t.Fatalf("revision=%q", event.Revision)
		}
	case <-time.After(time.Second):
		t.Fatal("matching actor did not receive invalidation")
	}
	select {
	case event := <-second:
		t.Fatalf("other actor received invalidation: %#v", event)
	default:
	}
}

func TestHubCoalescesSlowSubscriberToLatestRevision(t *testing.T) {
	hub := NewHub()
	events, cancel := hub.Subscribe(Scope{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "person"})
	defer cancel()
	hub.Publish(Event{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "person", Revision: "rev-1"})
	hub.Publish(Event{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "person", Revision: "rev-2"})
	event := <-events
	if event.Revision != "rev-2" {
		t.Fatalf("revision=%q want rev-2", event.Revision)
	}
}


func TestHubHandlesLargeActorBurstWithoutCrossScopeLeak(t *testing.T) {
	hub := NewHub()
	const actors = 500
	channels := make([]<-chan Event, 0, actors)
	cancels := make([]func(), 0, actors)
	for index := 0; index < actors; index++ {
		events, cancel := hub.Subscribe(Scope{
			TenantID: "tenant", LegalEntityID: "entity", PrincipalID: fmt.Sprintf("person-%04d", index),
		})
		channels = append(channels, events)
		cancels = append(cancels, cancel)
	}
	defer func() {
		for _, cancel := range cancels {
			cancel()
		}
	}()

	for revision := 1; revision <= 5; revision++ {
		for index := 0; index < actors; index++ {
			hub.Publish(Event{
				TenantID: "tenant", LegalEntityID: "entity", PrincipalID: fmt.Sprintf("person-%04d", index),
				Revision: fmt.Sprintf("rev-%d", revision),
			})
		}
	}
	for index, events := range channels {
		select {
		case event := <-events:
			if event.Revision != "rev-5" || event.PrincipalID != fmt.Sprintf("person-%04d", index) {
				t.Fatalf("actor %d received %#v", index, event)
			}
		case <-time.After(time.Second):
			t.Fatalf("actor %d did not receive burst invalidation", index)
		}
	}
}
