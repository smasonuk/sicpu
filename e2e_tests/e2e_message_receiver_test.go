package main

import (
	"bytes"
	"gocpu/pkg/compiler"
	"gocpu/pkg/cpu"
	"gocpu/pkg/peripherals"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMessageReceiver(t *testing.T) {
	srcPath := "../_capps/message_daemon.c"
	sourceBytes, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}

	_, machineCode, err := compiler.Compile(string(sourceBytes), "_capps")
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	c := cpu.NewCPU()
	mr := peripherals.NewMessageReceiver(c, 0)
	c.MountPeripheral(0, mr)

	if len(machineCode) > len(c.Memory) {
		t.Fatalf("Program too large: %d bytes", len(machineCode))
	}
	copy(c.Memory[:], machineCode)

	var outputBuf bytes.Buffer
	c.Output = &outputBuf

	stopChan := make(chan struct{})
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		for {
			select {
			case <-stopChan:
				return
			default:
				c.Step()
				if c.Halted {
					return
				}
				// Yield slightly to prevent tight loop starving scheduler
				// But we want fast execution.
				// runtime.Gosched()?
				// Or sleep 0?
			}
		}
	}()

	// Wait for init
	deadline := time.Now().Add(5 * time.Second)
	initialized := false
	for time.Now().Before(deadline) {
		if strings.Contains(outputBuf.String(), "Interrupts enabled") {
			initialized = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !initialized {
		close(stopChan)
		<-doneChan
		t.Fatalf("Daemon failed to initialize. Output:\n%s", outputBuf.String())
	}

	// Inject
	msg := "HELLO_WORLD"
	sender := "Earth"
	if err := mr.PushMessage(sender, []byte(msg)); err != nil {
		t.Fatalf("PushMessage failed: %v", err)
	}

	// Wait for process
	processed := false
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(outputBuf.String(), "Message Received from Earth: HELLO_WORLD") {
			processed = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	close(stopChan)
	<-doneChan

	if !processed {
		t.Errorf("Timeout waiting for message. Output:\n%s", outputBuf.String())
	}
}
