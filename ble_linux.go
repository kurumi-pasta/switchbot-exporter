//go:build linux
// +build linux

package main

import (
	"fmt"
	"os"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
)

func initBleDevice() ble.Device {
	d, err := linux.NewDevice()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init device: %v\n", err)
		os.Exit(1)
	}
	ble.SetDefaultDevice(d)
	return d
}
