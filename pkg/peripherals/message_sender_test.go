package peripherals

import (
	"bytes"
	"gocpu/pkg/cpu"
	"io"
	"os"
	"strings"
	"testing"
)

func TestMessagePeripheral(t *testing.T) {
	// Setup
	c := cpu.NewCPU()
	p := NewMessageSender(c, 0)
	c.MountPeripheral(0, p)

	// Action
	// Manually inject "system" into memory at 0x1000
	target := "system"
	for i := 0; i < len(target); i++ {
		c.Memory[0x1000+i] = target[i]
	}
	c.Memory[0x1000+len(target)] = 0 // Null terminator

	// Inject fake bytes at 0x2000
	c.Memory[0x2000] = 0xDE
	c.Memory[0x2001] = 0xAD
	c.Memory[0x2002] = 0xBE
	c.Memory[0x2003] = 0xEF

	// Set registers
	p.Write16(0x02, 0x1000)
	p.Write16(0x04, 0x2000)
	p.Write16(0x06, 4)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Trigger send
	p.Write16(0x00, 1)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Assertion
	expected := "[Message HW] To: system | Body: deadbeef\n"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain %q, got %q", expected, output)
	}
}
