//go:build darwin
// +build darwin

package main

/* #cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

int SetActivationPolicy(void) {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    return 0;
}
*/
import "C"

// setActivationPolicy is a macOS-only workaround for hiding the app dock icon
// while keeping the system tray menu available.
func setActivationPolicy() {
	// log is defined in main.go (same package)
	log.Debugln("Setting ActivationPolicy (darwin)")
	C.SetActivationPolicy()
}
