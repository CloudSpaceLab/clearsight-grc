package invalidation

import (
	"strings"
	"sync"
)

type Scope struct {
	TenantID      string
	LegalEntityID string
	PrincipalID   string
}

type Event struct {
	TenantID      string `json:"tenant_id,omitempty"`
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	PrincipalID   string `json:"principal_id,omitempty"`
	Revision      string `json:"revision"`
}

type Stream interface {
	Subscribe(Scope) (<-chan Event, func())
}

type Hub struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[string]map[uint64]chan Event
}

func NewHub() *Hub {
	return &Hub{subscribers: map[string]map[uint64]chan Event{}}
}

func (h *Hub) Subscribe(scope Scope) (<-chan Event, func()) {
	scope = normalizeScope(scope)
	channel := make(chan Event, 1)
	if h == nil || !validScope(scope) {
		close(channel)
		return channel, func() {}
	}
	key := scopeKey(scope)
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	if h.subscribers[key] == nil {
		h.subscribers[key] = map[uint64]chan Event{}
	}
	h.subscribers[key][id] = channel
	h.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			if group := h.subscribers[key]; group != nil {
				if current, ok := group[id]; ok {
					delete(group, id)
					close(current)
				}
				if len(group) == 0 {
					delete(h.subscribers, key)
				}
			}
			h.mu.Unlock()
		})
	}
	return channel, cancel
}

func (h *Hub) Publish(event Event) {
	event.TenantID = strings.TrimSpace(event.TenantID)
	event.LegalEntityID = strings.TrimSpace(event.LegalEntityID)
	event.PrincipalID = strings.TrimSpace(event.PrincipalID)
	event.Revision = strings.TrimSpace(event.Revision)
	if h == nil || event.Revision == "" || !validScope(Scope{
		TenantID: event.TenantID, LegalEntityID: event.LegalEntityID, PrincipalID: event.PrincipalID,
	}) {
		return
	}
	key := scopeKey(Scope{TenantID: event.TenantID, LegalEntityID: event.LegalEntityID, PrincipalID: event.PrincipalID})
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, channel := range h.subscribers[key] {
		select {
		case channel <- event:
		default:
			select {
			case <-channel:
			default:
			}
			select {
			case channel <- event:
			default:
			}
		}
	}
}

func normalizeScope(scope Scope) Scope {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	scope.PrincipalID = strings.TrimSpace(scope.PrincipalID)
	return scope
}

func validScope(scope Scope) bool {
	return scope.TenantID != "" && scope.LegalEntityID != "" && scope.PrincipalID != ""
}

func scopeKey(scope Scope) string {
	return scope.TenantID + "\x00" + scope.LegalEntityID + "\x00" + scope.PrincipalID
}

var _ Stream = (*Hub)(nil)
