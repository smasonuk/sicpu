package peripherals

import (
	"bytes"
	"gocpu/pkg/cpu"
	"testing"
)

func TestMessagePeripheral(t *testing.T) {
	// Setup
	c := cpu.NewCPU()

	var gotTarget string
	var gotBody []byte
	dispatch := func(target string, body []byte) {
		gotTarget = target
		gotBody = body
	}

	p := NewMessageSender(c, 0, dispatch)
	c.MountPeripheral(0, p)

	// Inject "system" into memory at 0x1000
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

	// Trigger send
	p.Write16(0x00, 1)

	// Assert dispatch was called with correct data
	if gotTarget != "system" {
		t.Errorf("expected target %q, got %q", "system", gotTarget)
	}

	expectedBody := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	if !bytes.Equal(gotBody, expectedBody) {
		t.Errorf("expected body %x, got %x", expectedBody, gotBody)
	}
}
