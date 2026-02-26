package cpu_test

import (
	"testing"
	"gocpu/pkg/cpu"
	"gocpu/pkg/peripherals"
)

type MockPeripheral struct {
	Read16Handler  func(offset uint16) uint16
	Write16Handler func(offset, val uint16)
	StepHandler    func()
}

func (m *MockPeripheral) Read16(offset uint16) uint16 {
	if m.Read16Handler != nil {
		return m.Read16Handler(offset)
	}
	return 0
}

func (m *MockPeripheral) Write16(offset, val uint16) {
	if m.Write16Handler != nil {
		m.Write16Handler(offset, val)
	}
}

func (m *MockPeripheral) Step() {
	if m.StepHandler != nil {
		m.StepHandler()
	}
}

func TestMountPeripheral_Valid(t *testing.T) {
	c := cpu.NewCPU()
	p := &MockPeripheral{}
	c.MountPeripheral(5, p)

	if c.Peripherals[5] != p {
		t.Errorf("Expected peripheral to be mounted at slot 5")
	}
}

func TestMountPeripheral_OutOfBounds(t *testing.T) {
	c := cpu.NewCPU()
	p := &MockPeripheral{}

	// Should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MountPeripheral panicked: %v", r)
			}
		}()
		c.MountPeripheral(16, p)
	}()

	// Check array is unchanged (all nil)
	for i := 0; i < 16; i++ {
		if c.Peripherals[i] != nil {
			t.Errorf("Expected slot %d to be nil", i)
		}
	}
}

func TestTriggerPeripheralInterrupt(t *testing.T) {
	c := cpu.NewCPU()
	c.TriggerPeripheralInterrupt(3)

	if c.PeripheralIntMask != 0x0008 {
		t.Errorf("Expected PeripheralIntMask to be 0x0008, got 0x%04X", c.PeripheralIntMask)
	}
	if !c.InterruptPending {
		t.Errorf("Expected InterruptPending to be true")
	}
}

func TestMMIO_Read16_Mapped(t *testing.T) {
	c := cpu.NewCPU()
	p := &MockPeripheral{
		Read16Handler: func(offset uint16) uint16 {
			if offset == 0x04 {
				return 0xABCD
			}
			return 0
		},
	}
	c.MountPeripheral(2, p) // 0xFC20

	val := c.Read16(0xFC24)
	if val != 0xABCD {
		t.Errorf("Expected 0xABCD, got 0x%04X", val)
	}
}

func TestMMIO_Write16_Mapped(t *testing.T) {
	c := cpu.NewCPU()
	var calledOffset, calledVal uint16
	p := &MockPeripheral{
		Write16Handler: func(offset, val uint16) {
			calledOffset = offset
			calledVal = val
		},
	}
	c.MountPeripheral(0, p) // 0xFC00

	c.Write16(0xFC08, 0x1234)

	if calledOffset != 0x08 {
		t.Errorf("Expected offset 0x08, got 0x%02X", calledOffset)
	}
	if calledVal != 0x1234 {
		t.Errorf("Expected val 0x1234, got 0x%04X", calledVal)
	}
}

func TestMMIO_Unmapped(t *testing.T) {
	c := cpu.NewCPU()
	// Slot 5 (0xFC50) is unmapped
	val := c.Read16(0xFC50)
	if val != 0 {
		t.Errorf("Expected 0 for unmapped read, got 0x%04X", val)
	}

	// Write should not panic
	c.Write16(0xFC50, 0xFFFF)
}

func TestMMIO_ByteAccess(t *testing.T) {
	c := cpu.NewCPU()
	var lastWriteVal uint16
	p := &MockPeripheral{
		Read16Handler: func(offset uint16) uint16 {
			// Return current value for read-modify-write
			// Assuming we simulate a register that holds value.
			// For simplicity, just return 0 initially, or store state.
			return lastWriteVal
		},
		Write16Handler: func(offset, val uint16) {
			lastWriteVal = val
		},
	}
	c.MountPeripheral(0, p) // 0xFC00

	// Write byte 0xAA to 0xFC00 (low byte)
	// Read16 will return 0 (initial lastWriteVal)
	// Write16 will be called with (0 & 0xFF00) | 0xAA = 0x00AA
	c.WriteByte(0xFC00, 0xAA)

	if lastWriteVal != 0x00AA {
		t.Errorf("Expected 0x00AA after low byte write, got 0x%04X", lastWriteVal)
	}

	// Write byte 0xBB to 0xFC01 (high byte)
	// Read16 returns 0x00AA
	// Write16 called with (0x00AA & 0x00FF) | (0xBB << 8) = 0xBBAA
	c.WriteByte(0xFC01, 0xBB)

	if lastWriteVal != 0xBBAA {
		t.Errorf("Expected 0xBBAA after high byte write, got 0x%04X", lastWriteVal)
	}
}

func TestInterruptRegister_Read(t *testing.T) {
	c := cpu.NewCPU()
	c.PeripheralIntMask = 0x0005 // Slots 0 and 2

	val := c.Read16(0xFF09)
	if val != 0x0005 {
		t.Errorf("Expected 0x0005, got 0x%04X", val)
	}
}

func TestInterruptRegister_Acknowledge(t *testing.T) {
	c := cpu.NewCPU()
	c.PeripheralIntMask = 0x000F

	c.Write16(0xFF09, 0x0002) // Clear bit 1

	if c.PeripheralIntMask != 0x000D {
		t.Errorf("Expected 0x000D, got 0x%04X", c.PeripheralIntMask)
	}
}

func TestPeripheral_StepCalled(t *testing.T) {
	c := cpu.NewCPU()

	// Write JMP 0 instruction at address 0 to prevent HALT
	// OpJMP = 0x0E. Instruction = 0x0E << 10 = 0x3800
	// LE: 0x00 0x38
	c.Memory[0] = 0x00
	c.Memory[1] = 0x38
	// Target address 0x0000
	c.Memory[2] = 0x00
	c.Memory[3] = 0x00

	steps := 0
	p := &MockPeripheral{
		StepHandler: func() {
			steps++
		},
	}
	c.MountPeripheral(0, p)

	for i := 0; i < 10; i++ {
		c.Step()
	}

	if steps != 10 {
		t.Errorf("Expected 10 steps, got %d", steps)
	}
}

func TestDMATransfer_Success(t *testing.T) {
	c := cpu.NewCPU()
	dma := peripherals.NewDMATester(c, 1)
	c.MountPeripheral(1, dma)

	// Target Address: 0x1000
	targetAddr := uint16(0x1000)
	length := uint16(16)

	// Write Target Address to 0xFC12 (Slot 1, Offset 2)
	c.Write16(0xFC12, targetAddr)

	// Write Length to 0xFC14 (Slot 1, Offset 4)
	c.Write16(0xFC14, length)

	// Trigger DMA: Write 1 to 0xFC10 (Slot 1, Offset 0)
	c.Write16(0xFC10, 1)

	// Assert Memory content
	for i := uint16(0); i < length; i++ {
		val := c.ReadByte(targetAddr + i)
		if val != 0xAA {
			t.Errorf("Expected memory at 0x%04X to be 0xAA, got 0x%02X", targetAddr+i, val)
		}
	}

	// Assert Interrupt
	if c.PeripheralIntMask != 0x0002 {
		t.Errorf("Expected PeripheralIntMask to be 0x0002, got 0x%04X", c.PeripheralIntMask)
	}
	if !c.InterruptPending {
		t.Errorf("Expected InterruptPending to be true")
	}
}
