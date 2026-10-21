package controller

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

func start(t *testing.T) *Controller {
	t.Helper()
	c := New(domain.DefaultState())
	ctx, cancel := context.WithCancel(context.Background())
	go c.Run(ctx)
	t.Cleanup(cancel)
	return c
}

func next(t *testing.T, s *Subscription) domain.Snapshot {
	t.Helper()
	select {
	case snap := <-s.C():
		return snap
	case <-time.After(time.Second):
		t.Fatal("no snapshot received")
		return domain.Snapshot{}
	}
}

func TestSubscribeReceivesCurrentSnapshot(t *testing.T) {
	c := start(t)
	got := next(t, c.Subscribe())
	if got.State != domain.DefaultState() || got.Levels != domain.Silence {
		t.Errorf("initial snapshot = %+v", got)
	}
}

func TestCommandsArePublished(t *testing.T) {
	c := start(t)
	sub := c.Subscribe()
	next(t, sub)

	c.Submit(domain.ToggleMute{})
	if got := next(t, sub); got.State.Master.Mute {
		t.Error("mute still on after ToggleMute")
	}
}

func TestNoOpCommandPublishesNothing(t *testing.T) {
	c := start(t)
	sub := c.Subscribe()
	next(t, sub)

	c.Submit(domain.SetMute{On: true}) // already muted
	c.Submit(domain.SetVolume{DB: -30})
	if got := next(t, sub); got.State.Master.VolumeDB != -30 {
		t.Errorf("expected the volume change as the next snapshot, got %+v", got.State.Master)
	}
}

func TestSlowSubscriberGetsLatest(t *testing.T) {
	c := start(t)
	sub := c.Subscribe()
	for db := -100; db <= -1; db++ {
		c.Submit(domain.SetVolume{DB: db})
	}
	deadline := time.After(time.Second)
	for {
		select {
		case snap := <-sub.C():
			if snap.State.Master.VolumeDB == -1 {
				return
			}
		case <-deadline:
			t.Fatal("never saw the final volume")
		}
	}
}

// A subscriber that never reads must not stall the controller or others.
func TestStalledSubscriberDoesNotBlock(t *testing.T) {
	c := start(t)
	_ = c.Subscribe() // never read
	live := c.Subscribe()

	var wg sync.WaitGroup
	for w := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 1000 {
				c.Submit(domain.SetLevels{Levels: domain.Levels{LeftDB: float64(-i), RightDB: float64(w)}})
			}
		}()
	}
	wg.Wait()
	c.Submit(domain.SetVolume{DB: -7})

	deadline := time.After(2 * time.Second)
	for {
		select {
		case snap := <-live.C():
			if snap.State.Master.VolumeDB == -7 {
				return
			}
		case <-deadline:
			t.Fatal("controller stalled")
		}
	}
}

func TestSubmitAfterStopReturns(t *testing.T) {
	c := New(domain.DefaultState())
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() { c.Run(ctx); close(stopped) }()
	cancel()
	<-stopped

	done := make(chan struct{})
	go func() {
		for range commandBuffer + 10 {
			c.Submit(domain.ToggleMute{})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Submit blocked after the controller stopped")
	}
}

func TestStopAppliesQueuedCommands(t *testing.T) {
	c := New(domain.DefaultState())
	c.Submit(domain.SetDimDB{DB: 35}) // queued before Run, like a save right before quitting
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.Run(ctx)
	if got := c.Snapshot().State.Master.DimDB; got != 35 {
		t.Errorf("DimDB = %d, want 35", got)
	}
}

func TestUnsubscribe(t *testing.T) {
	c := start(t)
	sub := c.Subscribe()
	<-sub.C()
	c.Unsubscribe(sub)
	c.Submit(domain.ToggleMute{})
	live := c.Subscribe() // the change is applied once this sees it
	for next(t, live).State.Master.Mute {
	}
	select {
	case s := <-sub.C():
		t.Errorf("unsubscribed subscription received %+v", s.State.Master)
	default:
	}
}
