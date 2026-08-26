package thirdparty

import "sync"

// RelationshipLinkCoordinator serializes in-memory link termination and work
// creation. PostgreSQL repositories additionally lock the typed link row so
// the same invariant holds across processes.
type RelationshipLinkCoordinator struct{ mu sync.Mutex }

func (c *RelationshipLinkCoordinator) Lock() {
	if c != nil {
		c.mu.Lock()
	}
}

func (c *RelationshipLinkCoordinator) Unlock() {
	if c != nil {
		c.mu.Unlock()
	}
}
