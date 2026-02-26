package peripherals

import (
	"fmt"
	"gocpu/pkg/cpu"
)

type MessagePeripheral struct {
	c    *cpu.CPU
	slot uint8

	toAddr   uint16
	bodyAddr uint16
	bodyLen  uint16
}

func NewMessagePeripheral(c *cpu.CPU, slot uint8) *MessagePeripheral {
	return &MessagePeripheral{
		c:    c,
		slot: slot,
	}
}

func (m *MessagePeripheral) Read16(offset uint16) uint16 {
	switch offset {
	case 0x00:
		return 0
	case 0x02:
		return m.toAddr
	case 0x04:
		return m.bodyAddr
	case 0x06:
		return m.bodyLen
	}
	return 0
}

func (m *MessagePeripheral) Write16(offset uint16, val uint16) {
	switch offset {
	case 0x00:
		if val == 1 {
			m.sendMessage()
		}
	case 0x02:
		m.toAddr = val
	case 0x04:
		m.bodyAddr = val
	case 0x06:
		m.bodyLen = val
	}
}

func (m *MessagePeripheral) Step() {
	// Synchronous peripheral, no step needed
}

func (m *MessagePeripheral) sendMessage() {
	target, err := m.c.ReadStringFromRAM(m.toAddr)
	if err != nil {
		fmt.Printf("[Message HW] Error reading target: %v\n", err)
		return
	}

	body := make([]byte, m.bodyLen)
	for i := uint16(0); i < m.bodyLen; i++ {
		body[i] = m.c.ReadByte(m.bodyAddr + i)
		fmt.Printf("[Message HW] Read byte %d: 0x%02X, ascii: '%c', decimal: %d\n", i, body[i], body[i], body[i])
	}

	// fmt.Printf("[Message HW] To: %s | Body: %x\n", target, body)

	bodyStr := string(body)

	fmt.Printf("[Message HW] To: %s | Body: %s\n", target, bodyStr)
}
