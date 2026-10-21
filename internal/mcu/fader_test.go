package mcu

import (
	"testing"

	"github.com/sebastianrau/gomcu"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

func TestFaderEnds(t *testing.T) {
	for _, logScale := range []bool{false, true} {
		if got := faderToDB(0, logScale); got != domain.MinVolumeDB {
			t.Errorf("log=%t: faderToDB(0) = %d", logScale, got)
		}
		if got := faderToDB(faderMax, logScale); got != domain.MaxVolumeDB {
			t.Errorf("log=%t: faderToDB(max) = %d", logScale, got)
		}
		if got := faderToDB(16383, logScale); got != domain.MaxVolumeDB {
			t.Errorf("log=%t: faderToDB(16383) = %d", logScale, got)
		}
		if got := dbToFader(domain.MinVolumeDB-10, logScale); got != 0 {
			t.Errorf("log=%t: dbToFader(below min) = %d", logScale, got)
		}
		if got := dbToFader(6, logScale); got != faderMax {
			t.Errorf("log=%t: dbToFader(+6) = %d", logScale, got)
		}
	}
}

// Every dB value must survive a round trip through the fader, otherwise
// the fader would jump after the controller echoes the volume.
func TestFaderRoundTrip(t *testing.T) {
	for _, logScale := range []bool{false, true} {
		prev := -1
		for db := domain.MinVolumeDB; db <= domain.MaxVolumeDB; db++ {
			pos := dbToFader(db, logScale)
			if int(pos) <= prev {
				t.Fatalf("log=%t: dbToFader not increasing at %d dB", logScale, db)
			}
			prev = int(pos)
			if got := faderToDB(pos, logScale); got != db {
				t.Errorf("log=%t: %d dB -> %d -> %d dB", logScale, db, pos, got)
			}
		}
	}
}

func TestLogScaleMarks(t *testing.T) {
	for _, m := range []struct {
		pos uint16
		db  int
	}{{912, -60}, {6128, -20}, {14112, -3}} {
		if got := faderToDB(m.pos, true); got != m.db {
			t.Errorf("faderToDB(%d) = %d, want %d", m.pos, got, m.db)
		}
	}
}

func TestMeterLevel(t *testing.T) {
	tests := []struct {
		db   float64
		want gomcu.MeterLevel
	}{
		{3, gomcu.MoreThan0}, {0, gomcu.MoreThan0}, {-1, gomcu.MoreThan2}, {-9, gomcu.MoreThan10},
		{-25, gomcu.MoreThan30}, {-59, gomcu.MoreThan60}, {-60, gomcu.LessThan60}, {-127, gomcu.LessThan60},
	}
	for _, tt := range tests {
		if got := meterLevel(tt.db); got != tt.want {
			t.Errorf("meterLevel(%v) = %v, want %v", tt.db, got, tt.want)
		}
	}
}
