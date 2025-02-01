package focusritexml

type Settings struct {
	DelayCompensation  string      `xml:"delay-compensation"`
	PhantomPersistence ElementBool `xml:"phantom-persistence"`
	SpdifMode          SpdifMode   `xml:"spdif-mode"`
	Talkback           Talkback    `xml:"talkback"`
}
