package peripherals

import (
	"encoding/binary"
	"fmt"
	"gocpu/pkg/cpu"
)

const MessagePeripheralType = "MessagePeripheral"

type DispatchFunc func(target string, body []byte)

type MessageSender struct {
	c        *cpu.CPU
	slot     uint8
	toAddr   uint16
	bodyAddr uint16
	bodyLen  uint16
	dispatch DispatchFunc
}

func NewMessageSender(c *cpu.CPU, slot uint8, dispatch DispatchFunc) *MessageSender {
	return &MessageSender{
		c:        c,
		slot:     slot,
		dispatch: dispatch,
	}
}

func (m *MessageSender) Type() string { return MessagePeripheralType }

func (m *MessageSender) Read16(offset uint16) uint16 {
	if offset >= 0x08 && offset <= 0x0E {
		return cpu.EncodePeripheralName("MSGSNDR", offset)
	}
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

func (m *MessageSender) Write16(offset uint16, val uint16) {
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

func (m *MessageSender) Step() {
	// Synchronous peripheral, no step needed
}

// SaveState serialises toAddr, bodyAddr, and bodyLen as 6 little-endian bytes.
func (m *MessageSender) SaveState() []byte {
	buf := make([]byte, 6)
	binary.LittleEndian.PutUint16(buf[0:], m.toAddr)
	binary.LittleEndian.PutUint16(buf[2:], m.bodyAddr)
	binary.LittleEndian.PutUint16(buf[4:], m.bodyLen)
	return buf
}

// LoadState restores toAddr, bodyAddr, and bodyLen from the 6-byte payload.
func (m *MessageSender) LoadState(data []byte) error {
	if len(data) < 6 {
		return fmt.Errorf("MessagePeripheral.LoadState: need 6 bytes, got %d", len(data))
	}
	m.toAddr = binary.LittleEndian.Uint16(data[0:])
	m.bodyAddr = binary.LittleEndian.Uint16(data[2:])
	m.bodyLen = binary.LittleEndian.Uint16(data[4:])
	return nil
}

func (m *MessageSender) sendMessage() {
	target, err := m.c.ReadStringFromRAM(m.toAddr)
	if err != nil {
		fmt.Printf("[Message HW] Error reading target: %v\n", err)
		return
	}

	body := make([]byte, m.bodyLen)
	for i := uint16(0); i < m.bodyLen; i++ {
		body[i] = m.c.ReadByte(m.bodyAddr + i)
	}

	if m.dispatch != nil {
		m.dispatch(target, body)
	} else {
		fmt.Printf("[Message HW] To: %s | Body: %x\n", target, body)
	}
}
