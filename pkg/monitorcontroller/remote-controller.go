package monitorcontroller

import focusritexml "github.com/sebastianrau/focusrite-mackie-control/pkg/focusrite-xml"

type RemoteController interface {
	SetControlChannel(controllerChannel chan interface{}) //sets the Channel To Controller for remote control

	HandleDeviceArrival(*focusritexml.Device) // configured Device Arrived
	HandleDeviceRemoval()                     //configured Device removed
	HandleDim(bool)                           // sNew Dim State
	HandleMute(bool)                          // New Mute State
	HandleVolume(int)                         // Volume -127 .. 0 dB
	HandleMeter(int)                          // Meter Value in DB
	HandleSpeakerSelect(SpeakerID, bool)      // Speaker with given ID new selection State
	HandleSpeakerUpdate(SpeakerID, *SpeakerConfig)
	HandleMasterUpdate(*MasterConfig)
}

type RcUpdateRequest bool

type RcSetMute bool
type RcSetDim bool
type RcSetVolume int

type RcSpeakerSelect struct {
	id    SpeakerID
	state bool
}

/* TODO add more remote contorl functions
type RcSetSpeakerName struct {
	id   SpeakerID
	name string
}
*/
