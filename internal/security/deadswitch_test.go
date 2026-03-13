package security

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeadManSwitch_HeartbeatKeepsAlive(t *testing.T) {
	triggered := atomic.Bool{}
	dms := NewDeadManSwitch(200*time.Millisecond, func() {
		triggered.Store(true)
	})

	ctx := context.Background()
	dms.Start(ctx)
	defer dms.Stop()

	// Send heartbeats faster than the timeout
	for i := 0; i < 5; i++ {
		time.Sleep(80 * time.Millisecond)
		dms.Heartbeat()
	}

	if !dms.IsAlive() {
		t.Fatal("expected switch to be alive while heartbeats are sent")
	}
	if triggered.Load() {
		t.Fatal("trigger should not have fired while heartbeats are sent")
	}
}

func TestDeadManSwitch_TriggersOnMissedHeartbeat(t *testing.T) {
	triggered := make(chan struct{}, 1)
	dms := NewDeadManSwitch(100*time.Millisecond, func() {
		select {
		case triggered <- struct{}{}:
		default:
		}
	})

	ctx := context.Background()
	dms.Start(ctx)
	defer dms.Stop()

	// Don't send any heartbeats; wait for trigger
	select {
	case <-triggered:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("expected trigger to fire within timeout")
	}

	if dms.IsAlive() {
		t.Fatal("expected switch to be dead after trigger")
	}
}

func TestDeadManSwitch_StopPreventsTriggering(t *testing.T) {
	triggered := atomic.Bool{}
	dms := NewDeadManSwitch(100*time.Millisecond, func() {
		triggered.Store(true)
	})

	ctx := context.Background()
	dms.Start(ctx)
	dms.Stop()

	// Wait longer than the timeout
	time.Sleep(250 * time.Millisecond)

	if triggered.Load() {
		t.Fatal("trigger should not fire after Stop()")
	}
}

func TestDeadManSwitch_ContextCancellation(t *testing.T) {
	triggered := atomic.Bool{}
	dms := NewDeadManSwitch(100*time.Millisecond, func() {
		triggered.Store(true)
	})

	ctx, cancel := context.WithCancel(context.Background())
	dms.Start(ctx)
	defer dms.Stop()

	// Cancel the context before timeout
	cancel()

	time.Sleep(250 * time.Millisecond)

	if triggered.Load() {
		t.Fatal("trigger should not fire after context cancellation")
	}
}

func TestDeadManSwitch_IsAliveInitially(t *testing.T) {
	dms := NewDeadManSwitch(time.Hour, func() {})
	if !dms.IsAlive() {
		t.Fatal("newly created switch should be alive")
	}
}

func TestDeadManSwitch_HeartbeatResetsTimer(t *testing.T) {
	triggered := atomic.Bool{}
	dms := NewDeadManSwitch(150*time.Millisecond, func() {
		triggered.Store(true)
	})

	ctx := context.Background()
	dms.Start(ctx)
	defer dms.Stop()

	// Wait close to the timeout, then heartbeat
	time.Sleep(100 * time.Millisecond)
	dms.Heartbeat()

	// Wait again close to the timeout
	time.Sleep(100 * time.Millisecond)
	dms.Heartbeat()

	// Total elapsed > 200ms but each gap < 150ms, so no trigger
	if triggered.Load() {
		t.Fatal("trigger should not have fired with regular heartbeats")
	}
	if !dms.IsAlive() {
		t.Fatal("switch should be alive with regular heartbeats")
	}
}

func TestDeadManSwitch_MultipleStopCalls(t *testing.T) {
	dms := NewDeadManSwitch(time.Hour, func() {})
	dms.Start(context.Background())
	dms.Stop()
	// Second Stop should not panic or deadlock
	dms.Stop()
}
