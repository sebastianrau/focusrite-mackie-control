package guiconfig

import (
	"fmt"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/logger"
	"github.com/sebastianrau/focusrite-mackie-control/pkg/monitorcontroller"
)

var log *logger.CustomLogger = logger.WithPackage("gui-config")

type ControllerConfigGui struct {
	newConfig *monitorcontroller.ControllerSate

	dimSlider *widget.Slider
	dimLabel  *widget.Label

	speakerADisable   *widget.Check
	speakerAExclusive *widget.Check

	speakerBDisable   *widget.Check
	speakerBExclusive *widget.Check

	speakerCDisable   *widget.Check
	speakerCExclusive *widget.Check

	speakerDDisable   *widget.Check
	speakerDExclusive *widget.Check

	speakerSubDisable *widget.Check

	Container *widget.AccordionItem
}

func NewControllerConfig(cfg *monitorcontroller.ControllerSate) *ControllerConfigGui {
	cg := &ControllerConfigGui{
		newConfig: cfg,
	}

	cg.dimLabel = widget.NewLabel("")

	cg.dimSlider = widget.NewSlider(0, 60)
	cg.dimSlider.Step = 5
	cg.dimSlider.OnChanged = func(f float64) {
		log.Debugf("new dim: %f", f)
		cg.newConfig.Master.DimOffset = int(f)
		cg.Update()
	}

	cg.speakerADisable = widget.NewCheck("Disabled", func(b bool) {
		cg.newConfig.Speaker[monitorcontroller.SpeakerA].Disabled = b
	})

	cg.speakerAExclusive = widget.NewCheck("Exclusive", func(b bool) {
		cg.newConfig.Speaker[monitorcontroller.SpeakerA].Exclusive = b
	})

	cg.speakerBDisable = widget.NewCheck("Disabled", func(b bool) {
		cg.newConfig.Speaker[monitorcontroller.SpeakerB].Disabled = b
	})

	cg.speakerBExclusive = widget.NewCheck("Exclusive", func(b bool) {
		cg.newConfig.Speaker[monitorcontroller.SpeakerB].Exclusive = b
	})

	cg.speakerCDisable = widget.NewCheck("Disabled", func(b bool) {
		cg.newConfig.Speaker[monitorcontroller.SpeakerC].Disabled = b
	})

	cg.speakerCExclusive = widget.NewCheck("Exclusive", func(b bool) {
		cg.newConfig.Speaker[monitorcontroller.SpeakerC].Exclusive = b
	})

	cg.speakerDDisable = widget.NewCheck("Disabled", func(b bool) {
		cg.newConfig.Speaker[monitorcontroller.SpeakerD].Disabled = b
	})

	cg.speakerDExclusive = widget.NewCheck("Exclusive", func(b bool) {
		cg.newConfig.Speaker[monitorcontroller.SpeakerD].Exclusive = b
	})

	cg.speakerSubDisable = widget.NewCheck("Disabled", func(b bool) {
		cg.newConfig.Speaker[monitorcontroller.Sub].Disabled = b
	})

	cg.Container = widget.NewAccordionItem("Monitor Controller",
		container.New(layout.NewFormLayout(),
			cg.dimLabel, cg.dimSlider,
			widget.NewLabel("Speaker A:"), container.NewHBox(cg.speakerADisable, cg.speakerAExclusive),
			widget.NewLabel("Speaker B:"), container.NewHBox(cg.speakerBDisable, cg.speakerBExclusive),
			widget.NewLabel("Speaker C:"), container.NewHBox(cg.speakerCDisable, cg.speakerCExclusive),
			widget.NewLabel("Speaker D:"), container.NewHBox(cg.speakerDDisable, cg.speakerDExclusive),
			widget.NewLabel("Subwoofer:"), container.NewHBox(cg.speakerSubDisable),
		),
	)

	cg.Update()

	return cg
}

func (cg *ControllerConfigGui) Update() {
	cg.dimLabel.SetText(fmt.Sprintf("Dim: %d dB", cg.newConfig.Master.DimOffset))
	if int(cg.dimSlider.Value) != cg.newConfig.Master.DimOffset {
		cg.dimSlider.SetValue(float64(cg.newConfig.Master.DimOffset))
	}

	cg.updateSpeaker()
}

func (cg *ControllerConfigGui) updateSpeaker() {
	cg.speakerADisable.SetChecked(cg.newConfig.Speaker[monitorcontroller.SpeakerA].Disabled)
	cg.speakerAExclusive.SetChecked(cg.newConfig.Speaker[monitorcontroller.SpeakerA].Exclusive)

	cg.speakerADisable.SetChecked(cg.newConfig.Speaker[monitorcontroller.SpeakerA].Disabled)
	cg.speakerAExclusive.SetChecked(cg.newConfig.Speaker[monitorcontroller.SpeakerA].Exclusive)

	cg.speakerBDisable.SetChecked(cg.newConfig.Speaker[monitorcontroller.SpeakerB].Disabled)
	cg.speakerBExclusive.SetChecked(cg.newConfig.Speaker[monitorcontroller.SpeakerB].Exclusive)

	cg.speakerCDisable.SetChecked(cg.newConfig.Speaker[monitorcontroller.SpeakerC].Disabled)
	cg.speakerCExclusive.SetChecked(cg.newConfig.Speaker[monitorcontroller.SpeakerC].Exclusive)

	cg.speakerDDisable.SetChecked(cg.newConfig.Speaker[monitorcontroller.SpeakerD].Disabled)
	cg.speakerDExclusive.SetChecked(cg.newConfig.Speaker[monitorcontroller.SpeakerD].Exclusive)

	cg.speakerSubDisable.SetChecked(cg.newConfig.Speaker[monitorcontroller.Sub].Disabled)

}
