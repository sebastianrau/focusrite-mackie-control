package osc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sebastianrau/focusrite-mackie-control/internal/controller"
	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

// The full loop with the real controller: selecting speaker B over OSC
// switches the exclusive speaker A off, and both changes come back.
func TestRoundTripWithController(t *testing.T) {
	ctrl := controller.New(domain.DefaultState())
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go ctrl.Run(ctx)

	a := NewAdapter(Config{}, ctrl.Submit, ctrl.Subscribe().C())
	go a.Run(ctx)
	var srv net.Addr
	select {
	case srv = <-a.Listening():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start")
	}

	c := newClient(t, srv)
	c.send(AddrSubscribe)
	if got := c.await(SpeakerAddr(domain.SpeakerA)); got[0] != int32(1) {
		t.Fatalf("speaker A initially %v", got)
	}

	c.send(SpeakerAddr(domain.SpeakerB), int32(1))
	if got := c.await(SpeakerAddr(domain.SpeakerA)); got[0] != int32(0) {
		t.Errorf("speaker A after selecting B = %v, want 0", got)
	}
	if !ctrl.Snapshot().State.Speakers[domain.SpeakerB].Selected {
		t.Error("speaker B not selected in the controller")
	}
}
