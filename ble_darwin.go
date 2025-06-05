//go:build darwin
// +build darwin

package main

import (
	"fmt"
	"os"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/darwin"
)

func initBleDevice() ble.Device {
	d, err := darwin.NewDevice()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init device: %v\n", err)
		os.Exit(1)
	}
	ble.SetDefaultDevice(d)
	return d
}
