// Package controller owns the live monitor state. It applies commands from
// all adapters in order on a single goroutine and publishes every change as a
// snapshot to its subscribers.
package controller

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/logging"
)

var log = logging.New("controller")

const commandBuffer = 256

type Controller struct {
	cmds chan domain.Command
	done chan struct{}

	mu     sync.Mutex
	latest domain.Snapshot
	subs   []*Subscription
}

// New creates a controller starting from the given state. Call Run to start
// processing commands.
func New(initial domain.State) *Controller {
	return &Controller{
		cmds:   make(chan domain.Command, commandBuffer),
		done:   make(chan struct{}),
		latest: domain.Snapshot{State: initial, Levels: domain.Silence},
	}
}

// Run applies commands until ctx is cancelled.
func (c *Controller) Run(ctx context.Context) {
	defer close(c.done)
	for {
		select {
		case <-ctx.Done():
			// Apply what was submitted before the stop, e.g. settings saved
			// right before quitting.
			for {
				select {
				case cmd := <-c.cmds:
					c.apply(cmd)
				default:
					return
				}
			}
		case cmd := <-c.cmds:
			c.apply(cmd)
		}
	}
}

func (c *Controller) apply(cmd domain.Command) {
	c.mu.Lock()
	defer c.mu.Unlock()
	next := domain.Apply(c.latest, cmd)
	if next == c.latest {
		return
	}
	if _, meter := cmd.(domain.SetLevels); !meter {
		log.Debug("state changed", "command", fmt.Sprintf("%T%+v", cmd, cmd))
	}
	c.latest = next
	for _, s := range c.subs {
		s.offer(next)
	}
}

// Submit queues a command. It blocks only while the queue is full and returns
// immediately once the controller has stopped.
func (c *Controller) Submit(cmd domain.Command) {
	select {
	case c.cmds <- cmd:
	case <-c.done:
	}
}

// Snapshot returns the current snapshot.
func (c *Controller) Snapshot() domain.Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.latest
}

// Subscribe returns a subscription that immediately holds the current
// snapshot and then receives every change.
func (c *Controller) Subscribe() *Subscription {
	s := &Subscription{ch: make(chan domain.Snapshot, 1)}
	c.mu.Lock()
	defer c.mu.Unlock()
	s.offer(c.latest)
	c.subs = append(c.subs, s)
	return s
}

// Unsubscribe stops delivery to s.
func (c *Controller) Unsubscribe(s *Subscription) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subs = slices.DeleteFunc(c.subs, func(x *Subscription) bool { return x == s })
}

// Subscription delivers snapshots. A slow reader never blocks the controller:
// it skips intermediate snapshots and always receives the latest one.
type Subscription struct {
	ch chan domain.Snapshot
}

func (s *Subscription) C() <-chan domain.Snapshot { return s.ch }

// offer replaces any unread snapshot with snap. Only called with the
// controller lock held, so there is a single sender.
func (s *Subscription) offer(snap domain.Snapshot) {
	select {
	case <-s.ch:
	default:
	}
	s.ch <- snap
}
