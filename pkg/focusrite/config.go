package focusrite

// PresetConfiguration erstellt eine Beispielkonfiguration
func PresetConfiguration() *FocusriteControl {
	fc := NewFocusriteControl("Scarlett 18i20", "1234")

	// Beispielhafte Eingänge und Ausgänge hinzufügen
	fc.AddInput(1, 30.0, false, false, 0.5)
	fc.AddInput(2, 25.0, false, true, 0.7)
	fc.AddOutput(1, -6.0, false)
	fc.AddOutput(2, -10.0, true)

	// Routing hinzufügen
	fc.AddRouting(1, 1)
	fc.AddRouting(2, 2)

	return fc
}
