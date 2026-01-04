//go:build !darwin
// +build !darwin

package guifix

import "github.com/sebastianrau/focusrite-mackie-control/pkg/logger"

var log *logger.CustomLogger = logger.WithPackage("gui-fix")

func SetActivationPolicy() {}
