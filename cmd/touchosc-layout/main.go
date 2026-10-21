// Command touchosc-layout writes the TouchOSC demo layout for the monitor
// controller: go run ./cmd/touchosc-layout examples/touchosc/monitor-controller.tosc
package main

import (
	"fmt"
	"os"

	"github.com/sebastianrau/focusrite-mackie-control/internal/osc/touchosc"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: touchosc-layout <file.tosc>")
		os.Exit(2)
	}
	b, err := touchosc.Layout()
	if err == nil {
		err = os.WriteFile(os.Args[1], b, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "touchosc-layout:", err)
		os.Exit(1)
	}
}
