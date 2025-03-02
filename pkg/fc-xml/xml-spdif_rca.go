package focusritexml

type SpdifRca struct {
	ID               int           `xml:"id,attr"`
	SupportsTalkback bool          `xml:"supports-talkback,attr"`
	Hidden           bool          `xml:"hidden,attr"`
	Name             string        `xml:"name,attr"`
	StereoName       string        `xml:"stereo-name,attr"`
	Available        ElementBool   `xml:"available"`
	Meter            ElementFloat  `xml:"meter"`
	Nickname         ElementString `xml:"nickname"`
	Mute             ElementBool   `xml:"mute"`
	Stereo           ElementBool   `xml:"stereo"`
	Source           ElementInt    `xml:"source"`
}
