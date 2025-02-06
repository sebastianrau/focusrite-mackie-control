package faderdb

// Lookup-Tabelle für Fader -> dB.
var faderToDBTable = []struct {
	fader uint16
	db    float64
}{
	{0, -127},
	{1, -126},
	{912, -60},
	{1664, -50},
	{2416, -40},
	{4240, -30},
	{6128, -20},
	{7904, -10},
	{9808, -7},
	{11872, -5},
	{14112, -3},
	{16384, 0},
}

// FaderToDB konvertiert einen Faderwert (0 bis 16384) in Dezibel (dB) mithilfe einer Lookup-Tabelle und Interpolation.
func FaderToDB(faderValue uint16) float64 {
	if faderValue <= 0 {
		return -127.0
	}
	if faderValue >= 16384 {
		return 10 // Maximalwert
	}

	// Suche den Bereich für die Interpolation
	for i := 1; i < len(faderToDBTable); i++ {
		prev := faderToDBTable[i-1]
		curr := faderToDBTable[i]
		//fmt.Printf("Prev: %d, curr %d\n", prev.fader, curr.fader)

		if faderValue >= prev.fader && faderValue <= curr.fader {
			// Lineare Interpolation
			factor := float64(faderValue-prev.fader) / float64(curr.fader-prev.fader)
			return prev.db + factor*(curr.db-prev.db)
		}
	}

	return 10 // Fallback (sollte nie erreicht werden)
}

// DBToFader konvertiert einen Dezibel-Wert (dB) zurück in einen Faderwert (0 bis 16384) mithilfe einer Lookup-Tabelle und Interpolation.
func DBToFader(dbValue float64) uint16 {
	if dbValue <= -80 {
		return 0 // Minimalwert
	}
	if dbValue >= 10 {
		return 16384 // Maximalwert
	}

	// Suche den Bereich für die Interpolation
	for i := 1; i < len(faderToDBTable); i++ {
		prev := faderToDBTable[i-1]
		curr := faderToDBTable[i]
		if dbValue >= prev.db && dbValue <= curr.db {
			// Lineare Interpolation
			factor := (dbValue - prev.db) / (curr.db - prev.db)
			return prev.fader + uint16(factor*float64(curr.fader-prev.fader))
		}
	}

	return 16384 // Fallback (sollte nie erreicht werden)
}
