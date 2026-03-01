package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"gocpu/pkg/compiler"
	"gocpu/pkg/cpu"
	"gocpu/pkg/devices"
	"gocpu/pkg/peripherals"
	"gocpu/pkg/utils"
)

// startDiskSyncer flushes the VFS to disk every interval while stop is open.
func startDiskSyncer(vm *cpu.CPU, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if vm.Disk.Dirty {
				if err := vm.Disk.PersistTo(vm.StoragePath); err == nil {
					vm.Disk.Dirty = false
				}
			}
		case <-stop:
			return
		}
	}
}
func main() {
	filename := os.Args[1]
	showAsm := false
	if len(os.Args) > 2 {
		for _, arg := range os.Args[2:] {
			showAsm = arg == "--show-asm"
		}
	}

	fullPath, baseDir, err := utils.GetPathInfo(filename)
	sourceBytes, err := os.ReadFile(fullPath)
	if err != nil {
		log.Fatalf("Failed to read source file: %v", err)
	}
	demoSource := string(sourceBytes)

	print("Compiling source file:", fullPath, "\n")
	print("Base directory:", baseDir, "\n")
	// print("Source code:\n", demoSource, "\n")

	asm, mc, err := compiler.Compile(demoSource, baseDir)
	if err != nil {
		log.Print(*asm)
		log.Fatalf("Compilation failed: %v", err)
		return
	}
	machineCode := mc

	if showAsm {
		print("Generated Assembly:\n", *asm, "\n")
	}

	// dispatch := func(target string, body []byte) {
	// 	fmt.Printf("[Message HW] To: %s | Body: %x\n", target, body)
	// 	fmt.Printf("[Message HW as string] To: %s | Body: %s\n", target, string(body))

	// }

	// Register peripheral factories for hibernation restore.
	cpu.RegisterPeripheral(peripherals.MessagePeripheralType, func(c *cpu.CPU, slot uint8) cpu.Peripheral {
		return peripherals.NewMessageSender(c, slot, c.DispatchMessage)
	})
	cpu.RegisterPeripheral(peripherals.MessageReceiverType, func(c *cpu.CPU, slot uint8) cpu.Peripheral {
		return peripherals.NewMessageReceiver(c, slot)
	})

	cpu.RegisterMessageDevice(devices.NavigationDeviceType, func() cpu.MessageDevice {
		return devices.NewNavigationDevice()
	})

	vm := cpu.NewCPU("gocpu_vfs")
	vm.SetNonLocalMessages(func(target string, body []byte) {
		fmt.Printf("[Message HW] To: %s | Body: %x\n", target, body)
		fmt.Printf("[Message HW as string] To: %s | Body: %s\n", target, string(body))

	})

	msgReceiver := peripherals.NewMessageReceiver(vm, 1)
	vm.MessagePusher = msgReceiver.PushMessage
	vm.MountPeripheral(1, msgReceiver)

	vm.MountPeripheral(0, peripherals.NewMessageSender(vm, 0, vm.DispatchMessage))

	vm.MountMessageDevice("navigation@local", devices.NewNavigationDevice())

	if len(machineCode) > len(vm.Memory) {
		log.Fatalf("Program too large for memory")
	}
	copy(vm.Memory[:], machineCode)

	// Start background disk syncer (flushes dirty VFS to host every 3 s)
	stopSyncer := make(chan struct{})
	go startDiskSyncer(vm, 3*time.Second, stopSyncer)

	vm.RunUntilDone()

	close(stopSyncer)
	if vm.Disk.Dirty {
		_ = vm.Disk.PersistTo(vm.StoragePath)
	}

}
