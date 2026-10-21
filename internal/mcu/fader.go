package mcu

import (
	"math"

	"github.com/sebastianrau/gomcu"

	"github.com/sebastianrau/focusrite-mackie-control/internal/domain"
)

const faderMax = gomcu.FaderMax

// logTable maps fader positions to dB for the logarithmic scale. It follows
// the markings of a typical console fader: most travel covers -40..0 dB.
var logTable = []struct {
	pos uint16
	db  float64
}{
	{0, domain.MinVolumeDB},
	{1, domain.MinVolumeDB + 1},
	{912, -60},
	{1664, -50},
	{2416, -40},
	{4240, -30},
	{6128, -20},
	{7904, -10},
	{9808, -7},
	{11872, -5},
	{14112, -3},
	{faderMax, domain.MaxVolumeDB},
}

// faderToDB converts a fader position to a volume in dB.
func faderToDB(pos uint16, logScale bool) int {
	pos = min(pos, faderMax)
	if !logScale {
		return int(math.Round(float64(pos)*domain.MinVolumeDB*-1/faderMax)) + domain.MinVolumeDB
	}
	for i := 1; i < len(logTable); i++ {
		lo, hi := logTable[i-1], logTable[i]
		if pos <= hi.pos {
			f := float64(pos-lo.pos) / float64(hi.pos-lo.pos)
			return int(math.Round(lo.db + f*(hi.db-lo.db)))
		}
	}
	return domain.MaxVolumeDB
}

// dbToFader converts a volume in dB to a fader position.
func dbToFader(db int, logScale bool) uint16 {
	v := float64(min(max(db, domain.MinVolumeDB), domain.MaxVolumeDB))
	if !logScale {
		return uint16(math.Round((v - domain.MinVolumeDB) * faderMax / (domain.MinVolumeDB * -1)))
	}
	for i := 1; i < len(logTable); i++ {
		lo, hi := logTable[i-1], logTable[i]
		if v <= hi.db {
			f := (v - lo.db) / (hi.db - lo.db)
			return lo.pos + uint16(math.Round(f*float64(hi.pos-lo.pos)))
		}
	}
	return faderMax
}

// meterLevel converts a level in dBFS to the closest MCU meter segment.
func meterLevel(db float64) gomcu.MeterLevel {
	steps := []struct {
		above float64
		level gomcu.MeterLevel
	}{
		{0, gomcu.MoreThan0}, {-2, gomcu.MoreThan2}, {-4, gomcu.MoreThan4}, {-6, gomcu.MoreThan6},
		{-8, gomcu.MoreThan8}, {-10, gomcu.MoreThan10}, {-14, gomcu.MoreThan14}, {-20, gomcu.MoreThan20},
		{-30, gomcu.MoreThan30}, {-40, gomcu.MoreThan40}, {-50, gomcu.MoreThan50}, {-60, gomcu.MoreThan60},
	}
	if db >= 0 {
		return gomcu.MoreThan0
	}
	for _, s := range steps[1:] {
		if db > s.above {
			return s.level
		}
	}
	return gomcu.LessThan60
}
