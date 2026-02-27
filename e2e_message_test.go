package main

import (
	"bytes"
	"gocpu/pkg/compiler"
	"gocpu/pkg/cpu"
	"gocpu/pkg/peripherals"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSendMessageIntegration(t *testing.T) {
	// 1. Read C source
	sourceBytes, err := os.ReadFile("_capps/send_message.c")
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}
	source := string(sourceBytes)

	// 2. Compile
	_, mc, err := compiler.Compile(source, "_capps")
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// 3. Setup CPU and Peripheral
	vm := cpu.NewCPU()
	p := peripherals.NewMessageSender(vm, 0)
	vm.MountPeripheral(0, p)

	// 4. Load Code
	if len(mc) > len(vm.Memory) {
		t.Fatalf("Program too large for memory")
	}
	copy(vm.Memory[:], mc)

	// 5. Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 6. Run
	vm.RunUntilDone()

	// 7. Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// 8. Assertions
	expectedMessage := "[Message HW] To: Central Command | Body: Ground control to major Tom\n"
	if !strings.Contains(output, expectedMessage) {
		t.Errorf("Expected peripheral output %q, got %q", expectedMessage, output)
	}

	expectedPrint := "Message sent!\n"
	if !strings.Contains(output, expectedPrint) {
		t.Errorf("Expected console output %q, got %q", expectedPrint, output)
	}
}
