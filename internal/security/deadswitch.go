package security

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// DeadManSwitch monitors system liveness via a heartbeat mechanism.
// If Heartbeat is not called within the configured timeout, the switch
// fires onTrigger (e.g. close all positions, cancel all orders).
type DeadManSwitch struct {
	timeout   time.Duration
	onTrigger func()

	mu       sync.Mutex
	lastBeat time.Time
	alive    atomic.Bool
	cancel   context.CancelFunc
	stopped  chan struct{}
}

// NewDeadManSwitch creates a new switch that fires onTrigger if no heartbeat
// is received within timeout.
func NewDeadManSwitch(timeout time.Duration, onTrigger func()) *DeadManSwitch {
	d := &DeadManSwitch{
		timeout:   timeout,
		onTrigger: onTrigger,
		lastBeat:  time.Now(),
	}
	d.alive.Store(true)
	return d
}

// Heartbeat resets the dead-man timer.
func (d *DeadManSwitch) Heartbeat() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastBeat = time.Now()
	d.alive.Store(true)
}

// IsAlive returns true if the system is still considered alive (heartbeat
// was received within the timeout window).
func (d *DeadManSwitch) IsAlive() bool {
	return d.alive.Load()
}

// Start begins monitoring in a background goroutine.  It respects the
// supplied context and also stops when Stop is called.
func (d *DeadManSwitch) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	d.mu.Lock()
	d.cancel = cancel
	d.stopped = make(chan struct{})
	d.lastBeat = time.Now()
	d.alive.Store(true)
	d.mu.Unlock()

	go d.monitor(ctx)
}

// Stop terminates the monitoring goroutine and blocks until it exits.
func (d *DeadManSwitch) Stop() {
	d.mu.Lock()
	cancel := d.cancel
	stopped := d.stopped
	d.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if stopped != nil {
		<-stopped
	}
}

// monitor is the background loop.
func (d *DeadManSwitch) monitor(ctx context.Context) {
	defer func() {
		d.mu.Lock()
		ch := d.stopped
		d.mu.Unlock()
		if ch != nil {
			close(ch)
		}
	}()

	// Check at half the timeout interval for responsiveness.
	tick := time.NewTicker(d.timeout / 2)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			d.mu.Lock()
			elapsed := time.Since(d.lastBeat)
			d.mu.Unlock()

			if elapsed >= d.timeout {
				d.alive.Store(false)
				if d.onTrigger != nil {
					d.onTrigger()
				}
				return
			}
		}
	}
}
