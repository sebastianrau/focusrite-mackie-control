//go:build darwin
// +build darwin

package guifix

/* #cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

int SetActivationPolicy(void) {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    return 0;
}
*/
import "C"
import "github.com/sebastianrau/focusrite-mackie-control/pkg/logger"

var log *logger.CustomLogger = logger.WithPackage("gui-fix")

// setActivationPolicy is a macOS-only workaround for hiding the app dock icon
// while keeping the system tray menu available.
func SetActivationPolicy() {
	// log is defined in main.go (same package)
	log.Debugln("Setting ActivationPolicy (darwin)")
	C.SetActivationPolicy()
}
