package backend

import (
	"sync"
)

// ReadinessTracker tracks whether the backend is ready to serve requests.
// It is considered ready once all registered components have reported ready.
type ReadinessTracker struct {
	mu       sync.Mutex
	total    int
	reported int
	ready    bool
	ch       chan struct{}
}

// NewReadinessTracker creates a tracker expecting n components to report ready.
func NewReadinessTracker(n int) *ReadinessTracker {
	return &ReadinessTracker{
		total: n,
		ch:    make(chan struct{}),
	}
}

// ReportReady is called by a component when it has completed its first sync.
// Once all registered components have reported, the tracker transitions to ready.
func (r *ReadinessTracker) ReportReady() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ready {
		return
	}
	r.reported++
	if r.reported >= r.total {
		r.ready = true
		close(r.ch)
	}
}

// IsReady returns true if all components have reported ready.
func (r *ReadinessTracker) IsReady() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ready
}

// Ready returns a channel that is closed when the tracker becomes ready.
func (r *ReadinessTracker) Ready() <-chan struct{} {
	return r.ch
}
