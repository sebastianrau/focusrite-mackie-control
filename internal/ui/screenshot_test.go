package ui

import (
	"crypto/sha256"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

// TestScreenshots writes the README images when SCREENSHOTS names a folder:
//
//	make screenshots
func TestScreenshots(t *testing.T) {
	dir := os.Getenv("SCREENSHOTS")
	if dir == "" {
		t.Skip("set SCREENSHOTS to a folder to write the README images")
	}

	a, _ := newTestApp(t)
	s := domain.Snapshot{State: domain.DefaultState(), Device: connected()}
	s.State.Master.Mute, s.State.Master.VolumeDB = false, -24
	for i, name := range []string{"Mains", "Nearfields", "Mono Cube", "Headphones", "Sub"} {
		s.State.Speakers[i].Name = name
	}
	s.State.Speakers[domain.SpeakerA].Selected = false
	s.State.Speakers[domain.SpeakerB].Selected = true
	a.render(s)
	a.meter.left, a.meter.right = -9, -11
	a.meter.Refresh()
	capture(t, a.window.Content(), fyne.NewSize(320, 412), filepath.Join(dir, "main-window.png"))

	capture(t, trayStrip(), fyne.NewSize(330, 110), filepath.Join(dir, "menu-bar-icons.png"))
}

// trayStrip shows the menu bar icon states on a dark and a light menu bar.
func trayStrip() fyne.CanvasObject {
	icon := func(cap, fill string) fyne.CanvasObject {
		svg := strings.ReplaceAll(faderSVG+cap+`</svg>`, "#000000", fill)
		// Fyne caches rendered images by resource name, so every variant needs its own.
		name := fmt.Sprintf("icon-%x.svg", sha256.Sum256([]byte(svg)))
		img := canvas.NewImageFromResource(fyne.NewStaticResource(name, []byte(svg)))
		img.SetMinSize(fyne.NewSize(22, 22))
		return img
	}
	bar := func(bg uint32, fg string) fyne.CanvasObject {
		r := canvas.NewRectangle(hex(bg))
		r.CornerRadius = 6
		row := container.NewGridWithColumns(3)
		for _, c := range []string{capNormal, capDim, capMute} {
			row.Add(container.NewCenter(icon(c, fg)))
		}
		return container.NewStack(r, container.New(layout.NewCustomPaddedLayout(6, 6, 0, 0), row))
	}
	labels := container.NewGridWithColumns(3)
	for _, l := range []string{"Normal", "Dim", "Mute"} {
		txt := canvas.NewText(l, colTextDim)
		txt.TextSize = 12
		txt.Alignment = fyne.TextAlignCenter
		labels.Add(txt)
	}
	bg := canvas.NewRectangle(colBackground)
	return container.NewStack(bg, container.New(layout.NewCustomPaddedLayout(10, 10, 10, 10),
		container.NewVBox(labels, bar(0x2b2b2d, "#ffffff"), bar(0xececec, "#1d1d1f"))))
}

func capture(t *testing.T, content fyne.CanvasObject, size fyne.Size, path string) {
	t.Helper()
	w := test.NewTempWindow(t, content)
	w.Canvas().(software.WindowlessCanvas).SetScale(2)
	w.Resize(size)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}
