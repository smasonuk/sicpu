package peripherals

import (
	"encoding/binary"
	"errors"
	"fmt"
	"gocpu/pkg/cpu"
	"gocpu/pkg/vfs"
	"sync"
)

type MessageReceiver struct {
	c     *cpu.CPU
	slot  uint8
	state uint16 // 0=Idle, 1=Waiting
	mu    sync.Mutex
}

func NewMessageReceiver(c *cpu.CPU, slot uint8) *MessageReceiver {
	return &MessageReceiver{
		c:    c,
		slot: slot,
	}
}

func (cam *MessageReceiver) Type() string { return "MessageReceiver" }

func (m *MessageReceiver) Read16(offset uint16) uint16 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if offset >= 0x08 && offset <= 0x0E {
		return cpu.EncodePeripheralName("MSGRECV", offset)
	}
	switch offset {
	case 0x00:
		return m.state
	}
	return 0
}

func (m *MessageReceiver) Write16(offset uint16, val uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch offset {
	case 0x00:
		if val == 1 {
			m.state = 0 // ACK

			// Pop message from queue
			data, err := m.c.Disk.Read(".msgq.sys")
			if err != nil {
				if !errors.Is(err, vfs.ErrFileNotFound) {
					fmt.Printf("[MSGRECV] Error reading queue for ACK: %v\n", err)
				}
				return
			}

			if len(data) < 2 {
				// Corrupt, delete
				_ = m.c.Disk.Delete(".msgq.sys")
				return
			}

			msgLen := binary.LittleEndian.Uint16(data[0:2])
			totalLen := 2 + int(msgLen)

			if len(data) < totalLen {
				// Incomplete, corrupt
				_ = m.c.Disk.Delete(".msgq.sys")
				return
			}

			// Update queue: rewrite with remaining data
			remaining := data[totalLen:]
			if len(remaining) == 0 {
				_ = m.c.Disk.Delete(".msgq.sys")
			} else {
				// Deep copy
				newQueue := make([]byte, len(remaining))
				copy(newQueue, remaining)
				_ = m.c.Disk.Write(".msgq.sys", newQueue)
			}
		}
	}
}

func (m *MessageReceiver) Step() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != 0 {
		return
	}

	// Check for messages in .msgq.sys
	data, err := m.c.Disk.Read(".msgq.sys")
	if err != nil {
		if !errors.Is(err, vfs.ErrFileNotFound) {
			fmt.Printf("[MSGRECV] Error reading queue: %v\n", err)
		}
		return
	}

	if len(data) < 2 {
		// Corrupt or empty queue file, delete it
		_ = m.c.Disk.Delete(".msgq.sys")
		return
	}

	msgLen := binary.LittleEndian.Uint16(data[0:2])
	totalLen := 2 + int(msgLen)

	if len(data) < totalLen {
		fmt.Printf("[MSGRECV] Incomplete message in queue\n")
		return
	}

	payload := make([]byte, msgLen)
	copy(payload, data[2:totalLen])

	// Write payload to INBOX.MSG
	err = m.c.Disk.Write("INBOX.MSG", payload)
	if err != nil {
		fmt.Printf("[MSGRECV] Failed to write INBOX.MSG: %v\n", err)
		return
	}

	// DO NOT update queue yet. Wait for ACK.

	m.state = 1
	m.c.TriggerPeripheralInterrupt(m.slot)
}

func (m *MessageReceiver) PushMessage(msg []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var currentData []byte

	// Read existing queue if it exists
	if data, err := m.c.Disk.Read(".msgq.sys"); err == nil {
		currentData = data
	}

	// Append new message
	// Format: [Len: uint16][Body]
	newMsg := make([]byte, 2+len(msg))
	binary.LittleEndian.PutUint16(newMsg[0:2], uint16(len(msg)))
	copy(newMsg[2:], msg)

	finalData := append(currentData, newMsg...)

	return m.c.Disk.Write(".msgq.sys", finalData)
}
