package main

import (
	"gocpu/pkg/cpu"
	"gocpu/pkg/devices"
	"gocpu/pkg/peripherals"
	"testing"
)

func TestMainWiringIntegration(t *testing.T) {
	// Register components exactly as main does
	cpu.RegisterPeripheral(peripherals.MessagePeripheralType, func(c *cpu.CPU, slot uint8) cpu.Peripheral {
		return peripherals.NewMessageSender(c, slot, c.DispatchMessage)
	})
	cpu.RegisterPeripheral(peripherals.MessageReceiverType, func(c *cpu.CPU, slot uint8) cpu.Peripheral {
		return peripherals.NewMessageReceiver(c, slot)
	})
	cpu.RegisterMessageDevice(devices.NavigationDeviceType, func() cpu.MessageDevice {
		return devices.NewNavigationDevice()
	})

	vm := cpu.NewCPU("")

	msgReceiver := peripherals.NewMessageReceiver(vm, 2)
	vm.MessagePusher = msgReceiver.PushMessage
	vm.MountPeripheral(2, msgReceiver)

	msgSender := peripherals.NewMessageSender(vm, 0, vm.DispatchMessage)
	vm.MountPeripheral(0, msgSender)

	navDev := devices.NewNavigationDevice()
	vm.MountMessageDevice("navigation@local", navDev)

	// Simulate dispatching a message from a peripheral to the navigation device
	// This tests the DispatchMessage routing and device handling
	vm.DispatchMessage("navigation@local", []byte("move_forward"))

	// Check if the device updated its state
	if navDev.X != 10.0 {
		t.Errorf("Expected X=10.0 on navigation device, got %f", navDev.X)
	}
}
