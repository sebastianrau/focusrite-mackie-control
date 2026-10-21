//go:build !darwin

package platform

// HideFromDock is a no-op outside macOS.
func HideFromDock() {}
