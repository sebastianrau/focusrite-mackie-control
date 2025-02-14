package monitorcontroller

type RemoteController interface {
	SetControlChannel(controllerChannel chan interface{}) //sets the Channel To Controller for remote control

	HandleDim(bool)                               // sNew Dim State
	HandleMute(bool)                              // New Mute State
	HandleVolume(int)                             // Volume -127 .. 0 dB
	HandleMeter(int, int)                         // Meter Value in DB
	HandleSpeakerSelect(SpeakerID, bool)          // Speaker with given ID new Selection State
	HandleSpeakerName(SpeakerID, string)          // Speaker with given ID new Name Update
	HandleSpeakerUpdate(SpeakerID, *SpeakerState) // Send Speaker Update
	HandleMasterUpdate(*MasterState)              // Send Master Update
	HandleDeviceUpdate(*DeviceInfo)               //Device Arrived / Connected etc.

}

type RcUpdateRequest bool
type RcSetMute bool
type RcSetDim bool
type RcSetVolume int

type RcSpeakerSelect struct {
	Id    SpeakerID
	State bool
}

type RcSetSpeakerName struct {
	Id   SpeakerID
	Name string
}

type DeviceInfo struct {
	DeviceId        int
	Model           string
	SampleRate      string
	SerialNumber    string
	ConnectionState bool
}
