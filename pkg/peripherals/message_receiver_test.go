package peripherals

import (
	"bytes"
	"encoding/binary"
	"errors"
	"gocpu/pkg/cpu"
	"gocpu/pkg/vfs"
	"testing"
)

func TestMessageReceiver_PushMessage(t *testing.T) {
	c := cpu.NewCPU()
	mr := NewMessageReceiver(c, 0)
	c.MountPeripheral(0, mr)

	msg1 := []byte("Hello")
	sender1 := "Earth"
	err := mr.PushMessage(sender1, msg1)
	if err != nil {
		t.Fatalf("PushMessage failed: %v", err)
	}

	data, err := c.Disk.Read(".msgq.sys")
	if err != nil {
		t.Fatalf("Failed to read queue: %v", err)
	}

	expectedLen := 1 + len(sender1) + 2 + len(msg1)
	if len(data) != expectedLen {
		t.Fatalf("Expected queue length %d, got %d", expectedLen, len(data))
	}
	
	senderLen := int(data[0])
	if senderLen != len(sender1) {
		t.Errorf("Expected sender length %d, got %d", len(sender1), senderLen)
	}

	length := binary.LittleEndian.Uint16(data[1+senderLen : 1+senderLen+2])
	if length != uint16(len(msg1)) {
		t.Errorf("Expected length %d, got %d", len(msg1), length)
	}
	if !bytes.Equal(data[1+senderLen+2:], msg1) {
		t.Errorf("Expected payload %q, got %q", msg1, data[1+senderLen+2:])
	}

	msg2 := []byte("World")
	sender2 := "Mars"
	err = mr.PushMessage(sender2, msg2)
	if err != nil {
		t.Fatalf("PushMessage 2 failed: %v", err)
	}

	data, err = c.Disk.Read(".msgq.sys")
	if err != nil {
		t.Fatalf("Failed to read queue: %v", err)
	}

	expectedLen2 := expectedLen + 1 + len(sender2) + 2 + len(msg2)
	if len(data) != expectedLen2 {
		t.Fatalf("Expected queue length %d, got %d", expectedLen2, len(data))
	}
}

func TestMessageReceiver_Step(t *testing.T) {
	c := cpu.NewCPU()
	mr := NewMessageReceiver(c, 2) // Slot 2
	c.MountPeripheral(2, mr)

	msg := []byte("TEST_PAYLOAD")
	sender := "Venus"
	_ = mr.PushMessage(sender, msg)

	// Step 1: Process message
	mr.Step()

	// Check State
	if mr.state != 1 {
		t.Errorf("Expected state 1 (Waiting), got %d", mr.state)
	}

	// Check Interrupt
	if c.PeripheralIntMask&(1<<2) == 0 {
		t.Error("Expected interrupt bit for slot 2 to be set")
	}

	// Check INBOX.MSG
	inbox, err := c.Disk.Read("INBOX.MSG")
	if err != nil {
		t.Fatalf("INBOX.MSG not created: %v", err)
	}
	if !bytes.Equal(inbox, msg) {
		t.Errorf("INBOX.MSG content mismatch. Want %q, got %q", msg, inbox)
	}

	// Check SENDER.MSG
	senderInbox, err := c.Disk.Read("SENDER.MSG")
	if err != nil {
		t.Fatalf("SENDER.MSG not created: %v", err)
	}
	if string(senderInbox) != sender {
		t.Errorf("SENDER.MSG content mismatch. Want %q, got %q", sender, senderInbox)
	}

	// Check Queue (should STILL EXIST)
	qData, err := c.Disk.Read(".msgq.sys")
	if err != nil {
		t.Fatalf("Queue file missing after dispatch (should only delete on ACK): %v", err)
	}
	if len(qData) == 0 {
		t.Error("Queue file should not be empty yet")
	}

	// Step 2: Ensure it doesn't process again while Waiting
	mr.Step()
	if mr.state != 1 {
		t.Errorf("State changed unexpectedly: %d", mr.state)
	}

	// Step 3: ACK
	mr.Write16(0x00, 1)
	if mr.state != 0 {
		t.Errorf("Expected state 0 (Idle) after ACK, got %d", mr.state)
	}

	// NOW Queue should be empty/deleted
	_, err = c.Disk.Read(".msgq.sys")
	if !errors.Is(err, vfs.ErrFileNotFound) {
		t.Error("Expected .msgq.sys to be deleted (empty) after ACK")
	}
}

func TestMessageReceiver_MultipleMessages(t *testing.T) {
	c := cpu.NewCPU()
	mr := NewMessageReceiver(c, 0)
	c.MountPeripheral(0, mr)

	_ = mr.PushMessage("Earth", []byte("A"))
	_ = mr.PushMessage("Mars", []byte("B"))

	// Process first
	mr.Step()
	inbox, _ := c.Disk.Read("INBOX.MSG")
	if string(inbox) != "A" {
		t.Errorf("Expected 'A', got %q", inbox)
	}
	senderInbox, _ := c.Disk.Read("SENDER.MSG")
	if string(senderInbox) != "Earth" {
		t.Errorf("Expected 'Earth', got %q", senderInbox)
	}

	// Queue should still exist with 'A' and 'B'
	q, _ := c.Disk.Read(".msgq.sys")
	if len(q) == 0 {
		t.Error("Queue shouldn't be empty")
	}

	mr.Write16(0x00, 1) // ACK

	// Queue should now have 'B' only
	// Process second
	mr.Step()
	inbox, _ = c.Disk.Read("INBOX.MSG")
	if string(inbox) != "B" {
		t.Errorf("Expected 'B', got %q", inbox)
	}
	senderInbox, _ = c.Disk.Read("SENDER.MSG")
	if string(senderInbox) != "Mars" {
		t.Errorf("Expected 'Mars', got %q", senderInbox)
	}

	mr.Write16(0x00, 1) // ACK

	// Queue should be gone
	_, err := c.Disk.Read(".msgq.sys")
	if !errors.Is(err, vfs.ErrFileNotFound) {
		t.Error("Queue should be empty")
	}
}
