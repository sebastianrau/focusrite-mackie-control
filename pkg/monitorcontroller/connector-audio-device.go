package monitorcontroller

type AudioDevice interface {
	SetControlChannel(controllerChannel chan interface{}) //sets the Channel To Controller for remote control

	HandleDim(bool)                      // sNew Dim State
	HandleMute(bool)                     // New Mute State
	HandleVolume(int)                    // Volume -127 .. 0 dB
	HandleMeter(int)                     // Meter Value in DB
	HandleSpeakerSelect(SpeakerID, bool) // Speaker with given ID new selection State
	HandleSpeakerName(SpeakerID, string) // Speaker with given ID new Name Update
	HandleSpeakerUpdate(SpeakerID, *SpeakerState)
	HandleMasterUpdate(*MasterState)
}

type AdUpdateRequest struct{}
type AdSetMute bool
type AdSetDim bool
type AdSetVolume int

type AdSetLevel struct {
	Left  int
	Right int
}

type AdSpeakerSelect struct {
	Id    SpeakerID
	State bool
}

type AdSetSpeakerName struct {
	Id   SpeakerID
	Name string
}

type AdSetDeviceStatus struct {
	DeviceId        int
	Model           string
	SerialNumber    string
	SampleRate      string
	ConnectionState bool
}
