package focusrite

import (
	"context"
	"slices"
	"strconv"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
	"github.com/sebastianrau/focusrite-mackie-control/internal/focusrite/protocol"
)

// Sender delivers sets to Focusrite Control. *Client implements it.
type Sender interface {
	Send(protocol.Set)
}

// Adapter keeps the configured interface in sync with the controller state.
// All fields are owned by the Run goroutine.
type Adapter struct {
	cfg    Config
	events <-chan Event
	sender Sender
	submit func(domain.Command)
	snaps  <-chan domain.Snapshot

	// OnAdopt is called with the generated configuration when the adapter
	// adopts an interface because none was configured.
	OnAdopt func(Config)

	catalog chan []DeviceInfo

	serverConnected bool
	devices         map[int]*protocol.Device
	active          *protocol.Device
	// known holds the last value per element ID that was sent to or reported
	// by the active device. Only differing values are sent.
	known map[int]string
	// inflight holds values sent per element ID that the device has not
	// echoed back yet, so a late echo is not mistaken for a user change.
	inflight map[int][]string
	// syncing is set from arrival until the device's first set, which
	// carries its current values. Nothing is sent before that, so the
	// device's own state (hardware mute and dim, nicknames) is not
	// overwritten by a push queued earlier.
	syncing bool
	snap    domain.Snapshot
	ready   bool
}

func NewAdapter(cfg Config, events <-chan Event, sender Sender, submit func(domain.Command), snaps <-chan domain.Snapshot) *Adapter {
	return &Adapter{
		cfg:     cfg,
		events:  events,
		sender:  sender,
		submit:  submit,
		snaps:   snaps,
		catalog: make(chan []DeviceInfo, 1),
		devices: make(map[int]*protocol.Device),
	}
}

// Devices delivers the list of attached interfaces whenever it changes. Only
// the latest list is kept for a slow reader.
func (a *Adapter) Devices() <-chan []DeviceInfo { return a.catalog }

func (a *Adapter) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-a.events:
			if !ok {
				return
			}
			a.handle(e)
		case s := <-a.snaps:
			a.snap, a.ready = s, true
			a.push()
		}
	}
}

func (a *Adapter) handle(e Event) {
	switch e := e.(type) {
	case ServerConnected:
		a.serverConnected = e.Connected
		if !e.Connected {
			clear(a.devices)
			a.active = nil
			a.publishCatalog()
		}
		a.reportDevice()

	case Approved:
		if e.Authorised {
			log.Info("approved by Focusrite Control")
		} else {
			log.Warn("not approved by Focusrite Control yet: allow \"" + clientHostname + "\" in Focusrite Control's remote device settings")
		}

	case DeviceArrived:
		d := e.Device
		a.devices[d.ID] = d
		a.publishCatalog()
		if a.cfg.SerialNumber == "" {
			a.cfg = AutoConfig(Describe(d))
			log.Info("no interface configured, adopting", "model", d.Model, "serial", d.SerialNumber)
			if a.OnAdopt != nil {
				a.OnAdopt(a.cfg)
			}
		}
		if d.SerialNumber == a.cfg.SerialNumber {
			a.active = d
			a.known = make(map[int]string)
			a.inflight = make(map[int][]string)
			a.syncing = true
			a.reportDevice()
		}

	case DeviceRemoved:
		delete(a.devices, e.ID)
		a.publishCatalog()
		if a.active != nil && a.active.ID == e.ID {
			a.active = nil
			a.reportDevice()
		}

	case DeviceUpdated:
		d, ok := a.devices[e.Set.DevID]
		if !ok {
			return
		}
		if unknown, err := d.ApplySet(e.Set); err != nil || len(unknown) > 0 {
			log.Debug("set not fully applied", "device", d.ID, "unknown", unknown, "err", err)
		}
		if d == a.active {
			a.fromDevice(e.Set)
			a.syncing = false
			a.reportDevice() // sample rate may have changed; no-op otherwise
			a.push()         // e.g. new output gains after a hardware dim
		}
	}
}

// fromDevice turns changes made on the interface or in Focusrite Control
// into commands. They are applied to the local snapshot at once, so a push
// before the controller's next snapshot does not undo them.
func (a *Adapter) fromDevice(set protocol.Set) {
	levels := a.snap.Levels
	meterSrc, metering := a.snap.State.MeterSource()
	var meter Output
	if metering {
		meter = a.cfg.Outputs[meterSrc]
	}

	for _, it := range set.Items {
		if a.isEcho(it) {
			continue
		}
		a.known[it.ID] = it.Value
		switch it.ID {
		case a.cfg.MuteID:
			if on, err := strconv.ParseBool(it.Value); err == nil {
				a.apply(domain.SetMute{On: on})
			}
		case a.cfg.DimID:
			if on, err := strconv.ParseBool(it.Value); err == nil {
				a.apply(domain.SetDim{On: on})
			}
		}
		for _, id := range domain.Speakers {
			if out := a.cfg.Outputs[id]; out.NameID != 0 && out.NameID == it.ID {
				a.apply(domain.RenameSpeaker{ID: id, Name: it.Value})
			}
		}
		if metering {
			switch it.ID {
			case meter.MeterLeftID:
				levels.LeftDB = parseLevel(it.Value)
			case meter.MeterRightID:
				levels.RightDB = parseLevel(it.Value)
			}
		}
	}
	if metering && levels != a.snap.Levels {
		a.apply(domain.SetLevels{Levels: levels})
	}
}

func (a *Adapter) apply(cmd domain.Command) {
	a.submit(cmd)
	a.snap = domain.Apply(a.snap, cmd)
}

// maxInflight bounds the remembered unechoed values per element. Coalesced
// sends mean intermediate values are never echoed.
const maxInflight = 8

// isEcho reports whether the device is confirming a value this adapter sent.
func (a *Adapter) isEcho(it protocol.Item) bool {
	pending := a.inflight[it.ID]
	i := slices.Index(pending, it.Value)
	if i < 0 {
		delete(a.inflight, it.ID)
		return false
	}
	if rest := pending[i+1:]; len(rest) > 0 {
		a.inflight[it.ID] = rest
	} else {
		delete(a.inflight, it.ID)
	}
	return true
}

func parseLevel(v string) float64 {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return domain.SilenceDB
	}
	return max(f, domain.SilenceDB)
}

// push sends the element values that differ from what the device has.
func (a *Adapter) push() {
	if a.active == nil || !a.ready || a.syncing {
		return
	}
	var changed []protocol.Item
	for _, it := range DesiredItems(a.cfg, a.snap.State) {
		if v, ok := a.known[it.ID]; ok && v == it.Value {
			continue
		}
		a.known[it.ID] = it.Value
		pending := append(a.inflight[it.ID], it.Value)
		a.inflight[it.ID] = pending[max(0, len(pending)-maxInflight):]
		changed = append(changed, it)
	}
	if len(changed) > 0 {
		a.sender.Send(protocol.Set{DevID: a.active.ID, Items: changed})
	}
}

func (a *Adapter) reportDevice() {
	d := domain.Device{ServerConnected: a.serverConnected}
	if a.active != nil {
		d.Present = true
		d.Model = a.active.Model
		d.SerialNumber = a.active.SerialNumber
		d.SampleRate = a.active.Clocking.SampleRate.Value
	}
	a.submit(domain.SetDevice{Device: d})
}

func (a *Adapter) publishCatalog() {
	list := make([]DeviceInfo, 0, len(a.devices))
	for _, d := range a.devices {
		list = append(list, Describe(d))
	}
	sortDevices(list)
	select {
	case <-a.catalog:
	default:
	}
	a.catalog <- list
}
