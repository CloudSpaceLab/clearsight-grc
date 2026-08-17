package aigateway

import (
	"sync"
	"time"
)

type circuitBreaker struct {
	mu               sync.Mutex
	failureThreshold int64
	openDuration     time.Duration
	failures         int64
	openUntil        time.Time
	probeInFlight    bool
}

func newCircuitBreaker(config CircuitBreakerConfig) *circuitBreaker {
	return &circuitBreaker{failureThreshold: config.FailureThreshold, openDuration: time.Duration(config.OpenDurationMS) * time.Millisecond}
}

func (c *circuitBreaker) allow(now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.openUntil.IsZero() {
		return true
	}
	if now.Before(c.openUntil) || c.probeInFlight {
		return false
	}
	c.probeInFlight = true
	return true
}

func (c *circuitBreaker) success() {
	c.mu.Lock()
	c.failures = 0
	c.openUntil = time.Time{}
	c.probeInFlight = false
	c.mu.Unlock()
}

func (c *circuitBreaker) failure(now time.Time) {
	c.mu.Lock()
	c.failures++
	if c.probeInFlight || c.failures >= c.failureThreshold {
		c.openUntil = now.Add(c.openDuration)
		c.probeInFlight = false
	}
	c.mu.Unlock()
}

func (c *circuitBreaker) neutral() {
	c.mu.Lock()
	// A caller rejection, client cancellation or downstream write failure says
	// nothing about provider health. Release a half-open probe without erasing
	// prior provider faults; only a completed exchange closes the circuit.
	c.probeInFlight = false
	c.mu.Unlock()
}

func (c *circuitBreaker) available(now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.openUntil.IsZero() || (!now.Before(c.openUntil) && !c.probeInFlight)
}
