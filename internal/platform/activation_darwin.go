//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

void setAccessoryActivationPolicy(void) {
	[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}
*/
import "C"

// HideFromDock turns the app into a menu bar accessory: no Dock icon, only the
// system tray menu. It must run on the main thread after the app has started.
func HideFromDock() {
	C.setAccessoryActivationPolicy()
}
