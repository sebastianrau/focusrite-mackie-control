package focusritexml

import "encoding/xml"

// DeviceArrival represents the root element.
type DeviceArrival struct {
	XMLName xml.Name `xml:"device-arrival"`
	Device  Device   `xml:"device"`
}

type Preset struct {
	ID   int    `xml:"id,attr"`
	Enum []Enum `xml:"enum"`
}

type Enum struct {
	Value string `xml:"value,attr"`
}

type Firmware struct {
	Version          ElementString `xml:"version"`
	NeedsUpdate      ElementBool   `xml:"needs-update"`
	FirmwareProgress ElementString `xml:"firmware-progress"`
	UpdateFirmware   ElementString `xml:"update-firmware"`
	RestoreFactory   ElementString `xml:"restore-factory"`
}

func (f *Firmware) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[f.Version.ID] = &f.Version
	elementsMap[f.NeedsUpdate.ID] = &f.NeedsUpdate
	elementsMap[f.FirmwareProgress.ID] = &f.FirmwareProgress
	elementsMap[f.UpdateFirmware.ID] = &f.UpdateFirmware
	elementsMap[f.RestoreFactory.ID] = &f.RestoreFactory
}

type Mixer struct {
	Available ElementBool `xml:"available"`
	Inputs    MixerInputs `xml:"inputs"`
	Mixes     []Mix       `xml:"mixes>mix"`
}

func (m *Mixer) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[m.Available.ID] = &m.Available
	m.Inputs.UpdateMap(elementsMap)
	for i := range m.Mixes {
		m.Mixes[i].UpdateMap(elementsMap)
	}
}

type MixerInputs struct {
	AddInput             ElementString `xml:"add-input"`
	AddInputWithoutReset ElementString `xml:"add-input-without-reset"`
	AddStereoInput       ElementString `xml:"add-stereo-input"`
	RemoveInput          ElementString `xml:"remove-input"`
	FreeInputs           ElementString `xml:"free-inputs"`
	InputList            []Input       `xml:"input"`
}

func (mi *MixerInputs) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[mi.AddInput.ID] = &mi.AddInput
	elementsMap[mi.AddInputWithoutReset.ID] = &mi.AddInputWithoutReset
	elementsMap[mi.AddStereoInput.ID] = &mi.AddStereoInput
	elementsMap[mi.RemoveInput.ID] = &mi.RemoveInput
	elementsMap[mi.FreeInputs.ID] = &mi.FreeInputs
	for i := range mi.InputList {
		mi.InputList[i].UpdateMap(elementsMap)
	}
}

type Input struct {
	Source ElementString `xml:"source"`
	Stereo ElementString `xml:"stereo"`
}

func (i *Input) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[i.Source.ID] = &i.Source
	elementsMap[i.Stereo.ID] = &i.Stereo
}

type Mix struct {
	ID         string     `xml:"id,attr"`
	Name       string     `xml:"name,attr"`
	StereoName string     `xml:"stereo-name,attr"`
	Meter      ElementInt `xml:"meter"`
	Inputs     []MixInput `xml:"input"`
}

func (m *Mix) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[m.Meter.ID] = &m.Meter
	for i := range m.Inputs {
		m.Inputs[i].UpdateMap(elementsMap)
	}
}

type MixInput struct {
	Gain ElementInt  `xml:"gain"`
	Pan  ElementInt  `xml:"pan"`
	Mute ElementBool `xml:"mute"`
	Solo ElementBool `xml:"solo"`
}

func (mi *MixInput) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[mi.Gain.ID] = &mi.Gain
	elementsMap[mi.Pan.ID] = &mi.Pan
	elementsMap[mi.Mute.ID] = &mi.Mute
	elementsMap[mi.Solo.ID] = &mi.Solo
}

type Inputs struct {
	Analogues []Analogue `xml:"analogue"`
	Playbacks []Playback `xml:"playback"`
}

func (i *Inputs) UpdateMap(elementsMap map[int]Elements) {
	for j := range i.Analogues {
		i.Analogues[j].UpdateMap(elementsMap)
	}
	for j := range i.Playbacks {
		i.Playbacks[j].UpdateMap(elementsMap)
	}
}

type Analogue struct {
	ID               string        `xml:"id,attr"`
	SupportsTalkback string        `xml:"supports-talkback,attr"`
	Hidden           string        `xml:"hidden,attr"`
	Name             string        `xml:"name,attr"`
	StereoName       string        `xml:"stereo-name,attr"`
	Available        ElementBool   `xml:"available"`
	Meter            ElementInt    `xml:"meter"`
	Nickname         ElementString `xml:"nickname"`
	Stereo           ElementBool   `xml:"stereo"`
	SourceID         ElementInt    `xml:"source"`
	Mode             ElementString `xml:"mode"`
	Air              ElementString `xml:"air"`
	Pad              ElementString `xml:"pad"`
	Mute             ElementBool   `xml:"mute"`
	Gain             ElementInt    `xml:"gain"`
	HardwareControl  ElementString `xml:"hardware-control"`
}

func (a *Analogue) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[a.Available.ID] = &a.Available
	elementsMap[a.Meter.ID] = &a.Meter
	elementsMap[a.Nickname.ID] = &a.Nickname
	elementsMap[a.Stereo.ID] = &a.Stereo
	elementsMap[a.SourceID.ID] = &a.SourceID
	elementsMap[a.Mode.ID] = &a.Mode
	elementsMap[a.Air.ID] = &a.Air
	elementsMap[a.Pad.ID] = &a.Pad
	elementsMap[a.Mute.ID] = &a.Mute
	elementsMap[a.Gain.ID] = &a.Gain
	elementsMap[a.HardwareControl.ID] = &a.HardwareControl
}

type Playback struct {
	ID               string        `xml:"id,attr"`
	SupportsTalkback string        `xml:"supports-talkback,attr"`
	Hidden           string        `xml:"hidden,attr"`
	Name             string        `xml:"name,attr"`
	StereoName       string        `xml:"stereo-name,attr"`
	Available        ElementBool   `xml:"available"`
	Meter            ElementInt    `xml:"meter"`
	Nickname         ElementString `xml:"nickname"`
}

func (p *Playback) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[p.Available.ID] = &p.Available
	elementsMap[p.Meter.ID] = &p.Meter
	elementsMap[p.Nickname.ID] = &p.Nickname
}

type Outputs struct {
	Analogues []Analogue `xml:"analogue"`
	Loopbacks []Loopback `xml:"loopback"`
}

func (o *Outputs) UpdateMap(elementsMap map[int]Elements) {
	for i := range o.Analogues {
		o.Analogues[i].UpdateMap(elementsMap)
	}
	for i := range o.Loopbacks {
		o.Loopbacks[i].UpdateMap(elementsMap)
	}
}

type Loopback struct {
	Name       string        `xml:"name,attr"`
	StereoName string        `xml:"stereo-name,attr"`
	Available  ElementString `xml:"available"`
	Meter      ElementInt    `xml:"meter"`
	AssignMix  ElementString `xml:"assign-mix"`
	AssignTBM  ElementString `xml:"assign-talkback-mix"`
	Mute       ElementBool   `xml:"mute"`
	Source     ElementString `xml:"source"`
	Stereo     ElementBool   `xml:"stereo"`
	Nickname   ElementString `xml:"nickname"`
}

func (l *Loopback) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[l.Available.ID] = &l.Available
	elementsMap[l.Meter.ID] = &l.Meter
	elementsMap[l.AssignMix.ID] = &l.AssignMix
	elementsMap[l.AssignTBM.ID] = &l.AssignTBM
	elementsMap[l.Mute.ID] = &l.Mute
	elementsMap[l.Source.ID] = &l.Source
	elementsMap[l.Stereo.ID] = &l.Stereo
	elementsMap[l.Nickname.ID] = &l.Nickname
}

type Monitoring struct {
	MonitorGroupPairs string `xml:",chardata"`
}

type Clocking struct {
	Locked      ElementBool   `xml:"locked"`
	ClockSource ElementString `xml:"clock-source"`
	SampleRate  ElementString `xml:"sample-rate"`
	ClockMaster ElementString `xml:"clock-master"`
}

func (c *Clocking) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[c.Locked.ID] = &c.Locked
	elementsMap[c.ClockMaster.ID] = &c.ClockMaster
	elementsMap[c.SampleRate.ID] = &c.SampleRate
	elementsMap[c.ClockSource.ID] = &c.ClockSource

}

type Settings struct {
	PhantomPersistence ElementBool `xml:"phantom-persistence"`
	DelayCompensation  string      `xml:"delay-compensation"`
}

func (s *Settings) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[s.PhantomPersistence.ID] = &s.PhantomPersistence
}

type QuickStart struct {
	URL     string        `xml:"url,attr"`
	MsdMode ElementString `xml:"msd-mode"`
}

func (qs *QuickStart) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[qs.MsdMode.ID] = &qs.MsdMode
}

type HaloSettings struct {
	AvailableColours []Enum        `xml:"available-colours>enum"`
	GoodMeterColour  ElementString `xml:"good-meter-colour"`
	PreClipColour    ElementString `xml:"pre-clip-meter-colour"`
	ClippingColour   ElementString `xml:"clipping-meter-colour"`
	EnablePreview    ElementBool   `xml:"enable-preview-mode"`
	Halos            ElementString `xml:"halos"`
}

func (hs *HaloSettings) UpdateMap(elementsMap map[int]Elements) {
	elementsMap[hs.GoodMeterColour.ID] = &hs.GoodMeterColour
	elementsMap[hs.PreClipColour.ID] = &hs.PreClipColour
	elementsMap[hs.ClippingColour.ID] = &hs.ClippingColour
	elementsMap[hs.EnablePreview.ID] = &hs.EnablePreview
	elementsMap[hs.Halos.ID] = &hs.Halos
}
