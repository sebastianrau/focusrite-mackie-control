package debuggerconnector

import (
	focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
)

type DebuggerConnector struct {
	log *logger.CustomLogger
}

func NewDebuggerConnector() *DebuggerConnector {
	dc := &DebuggerConnector{
		log: logger.WithPackage("DebugConnector"),
	}
	return dc
}

func (dc *DebuggerConnector) SetControlChannel(controllerChannel chan interface{}) {
	if controllerChannel != nil {
		dc.log.Debugf("Control Channel arrived was set")
	} else {
		dc.log.Debugf("Control Channel has been set to NIL")
	}
}

// configured Device Arrived
func (dc *DebuggerConnector) HandleDeviceArrival(dev *focusritexml.Device) {
	dc.log.Debugf("New FC Device arrived: ID: %d, %s (%s)", dev.ID, dev.Model, dev.SerialNumber)
}

// configured Device removed
func (dc *DebuggerConnector) HandleDeviceRemoval() {
	dc.log.Debugln("FC Device removed, no Device to control")
}

// sNew Dim State
func (dc *DebuggerConnector) HandleDim(state bool) {
	dc.log.Debugf("Dim: %t", state)
}

// New Mute State
func (dc *DebuggerConnector) HandleMute(state bool) {
	dc.log.Debugf("Mute: %t", state)
}

// Volume -127 .. 0 dB
func (dc *DebuggerConnector) HandleVolume(volume int) {
	dc.log.Debugf("Volume: %d", volume)
}

// Meter Value in DB
func (dc *DebuggerConnector) HandleMeter(volume int) {
	//dc.log.Debugf("Meter: %d", volume)
}

// Speaker with given ID new selection State
func (dc *DebuggerConnector) HandleSpeakerSelect(id monitorcontroller.SpeakerID, state bool) {
	dc.log.Debugf("Speaker %d: %t", id, state)
}

func (dc *DebuggerConnector) HandleSpeakerUpdate(id monitorcontroller.SpeakerID, spk *monitorcontroller.SpeakerState) {
	dc.log.Debugf("Speaker Update id %d: %s selected: %t", id, spk.Name, spk.Selected)
	dc.log.Debugf("%v", spk)
}
func (dc *DebuggerConnector) HandleMasterUpdate(master *monitorcontroller.MasterState) {
	dc.log.Debugf("master Update: Mute %t, Dim %t, Volume :%d dB",
		master.Mute,
		master.Dim,
		master.VolumeDB,
	)
	dc.log.Debugf("%v", master)
}
