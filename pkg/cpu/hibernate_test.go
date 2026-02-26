package cpu

import (
	"encoding/binary"
	"testing"
)

// ── Ticket 1: Core state round-trip ────────────────────────────────────────

func TestCPU_HibernateCoreState(t *testing.T) {
	c1 := NewCPU()
	c1.Regs[0] = 0x1234
	c1.Regs[1] = 0xABCD
	c1.Regs[7] = 0x0007
	c1.PC = 0x0042
	c1.SP = 0xFF00
	c1.Z = true
	c1.N = false
	c1.C = true
	c1.IE = true
	c1.Waiting = false
	c1.Halted = false
	c1.InterruptPending = true
	c1.CallDepth = 3
	c1.PeripheralIntMask = 0x000F
	c1.GraphicsEnabled = true
	c1.TextOverlay = false
	c1.BufferedMode = true
	c1.ColorMode8bpp = true
	c1.TextResolutionMode = 1
	c1.CurrentBank = 2
	c1.DisplayBank = 1
	c1.Palette[5] = 0xF81F
	c1.PaletteIndex = 5

	data, err := c1.HibernateToBytes()
	if err != nil {
		t.Fatalf("HibernateToBytes: %v", err)
	}

	c2 := NewCPU()
	if err := c2.RestoreFromBytes(data); err != nil {
		t.Fatalf("RestoreFromBytes: %v", err)
	}

	if c2.Regs != c1.Regs {
		t.Errorf("Regs mismatch: got %v, want %v", c2.Regs, c1.Regs)
	}
	if c2.PC != c1.PC {
		t.Errorf("PC: got 0x%04X, want 0x%04X", c2.PC, c1.PC)
	}
	if c2.SP != c1.SP {
		t.Errorf("SP: got 0x%04X, want 0x%04X", c2.SP, c1.SP)
	}
	if c2.Z != c1.Z {
		t.Errorf("Z: got %v, want %v", c2.Z, c1.Z)
	}
	if c2.N != c1.N {
		t.Errorf("N: got %v, want %v", c2.N, c1.N)
	}
	if c2.C != c1.C {
		t.Errorf("C: got %v, want %v", c2.C, c1.C)
	}
	if c2.IE != c1.IE {
		t.Errorf("IE: got %v, want %v", c2.IE, c1.IE)
	}
	if c2.Waiting != c1.Waiting {
		t.Errorf("Waiting: got %v, want %v", c2.Waiting, c1.Waiting)
	}
	if c2.Halted != c1.Halted {
		t.Errorf("Halted: got %v, want %v", c2.Halted, c1.Halted)
	}
	if c2.InterruptPending != c1.InterruptPending {
		t.Errorf("InterruptPending: got %v, want %v", c2.InterruptPending, c1.InterruptPending)
	}
	if c2.CallDepth != c1.CallDepth {
		t.Errorf("CallDepth: got %d, want %d", c2.CallDepth, c1.CallDepth)
	}
	if c2.PeripheralIntMask != c1.PeripheralIntMask {
		t.Errorf("PeripheralIntMask: got 0x%04X, want 0x%04X", c2.PeripheralIntMask, c1.PeripheralIntMask)
	}
	if c2.GraphicsEnabled != c1.GraphicsEnabled {
		t.Errorf("GraphicsEnabled: got %v, want %v", c2.GraphicsEnabled, c1.GraphicsEnabled)
	}
	if c2.TextOverlay != c1.TextOverlay {
		t.Errorf("TextOverlay: got %v, want %v", c2.TextOverlay, c1.TextOverlay)
	}
	if c2.BufferedMode != c1.BufferedMode {
		t.Errorf("BufferedMode: got %v, want %v", c2.BufferedMode, c1.BufferedMode)
	}
	if c2.ColorMode8bpp != c1.ColorMode8bpp {
		t.Errorf("ColorMode8bpp: got %v, want %v", c2.ColorMode8bpp, c1.ColorMode8bpp)
	}
	if c2.TextResolutionMode != c1.TextResolutionMode {
		t.Errorf("TextResolutionMode: got %d, want %d", c2.TextResolutionMode, c1.TextResolutionMode)
	}
	if c2.CurrentBank != c1.CurrentBank {
		t.Errorf("CurrentBank: got %d, want %d", c2.CurrentBank, c1.CurrentBank)
	}
	if c2.DisplayBank != c1.DisplayBank {
		t.Errorf("DisplayBank: got %d, want %d", c2.DisplayBank, c1.DisplayBank)
	}
	if c2.Palette != c1.Palette {
		t.Errorf("Palette mismatch")
	}
	if c2.PaletteIndex != c1.PaletteIndex {
		t.Errorf("PaletteIndex: got %d, want %d", c2.PaletteIndex, c1.PaletteIndex)
	}
}

// ── Ticket 2: Memory & VRAM binary serialisation ───────────────────────────

func TestCPU_HibernateMemory(t *testing.T) {
	c1 := NewCPU()

	// Write a recognisable pattern into main memory
	c1.Memory[0x0100] = 0xDE
	c1.Memory[0x0101] = 0xAD
	c1.Memory[0x0102] = 0xBE
	c1.Memory[0x0103] = 0xEF

	// Write into GraphicsBanks[1] (we need to temporarily switch CurrentBank)
	savedBank := c1.CurrentBank
	c1.CurrentBank = 1
	c1.GraphicsBanks[1][0x0010] = 0xCA
	c1.GraphicsBanks[1][0x0011] = 0xFE
	c1.CurrentBank = savedBank

	// Write into TextVRAM
	c1.TextVRAM[42] = 0x1234
	c1.TextVRAM[43] = 0x5678
	c1.TextVRAM_Front[10] = 0xABCD

	data, err := c1.HibernateToBytes()
	if err != nil {
		t.Fatalf("HibernateToBytes: %v", err)
	}

	c2 := NewCPU()
	if err := c2.RestoreFromBytes(data); err != nil {
		t.Fatalf("RestoreFromBytes: %v", err)
	}

	// Verify main memory pattern
	if c2.Memory[0x0100] != 0xDE || c2.Memory[0x0101] != 0xAD ||
		c2.Memory[0x0102] != 0xBE || c2.Memory[0x0103] != 0xEF {
		t.Errorf("Memory pattern mismatch: got %02X %02X %02X %02X",
			c2.Memory[0x0100], c2.Memory[0x0101], c2.Memory[0x0102], c2.Memory[0x0103])
	}

	// Verify GraphicsBanks[1]
	if c2.GraphicsBanks[1][0x0010] != 0xCA || c2.GraphicsBanks[1][0x0011] != 0xFE {
		t.Errorf("GraphicsBanks[1] mismatch: got %02X %02X",
			c2.GraphicsBanks[1][0x0010], c2.GraphicsBanks[1][0x0011])
	}

	// Verify TextVRAM
	if c2.TextVRAM[42] != 0x1234 {
		t.Errorf("TextVRAM[42]: got 0x%04X, want 0x1234", c2.TextVRAM[42])
	}
	if c2.TextVRAM[43] != 0x5678 {
		t.Errorf("TextVRAM[43]: got 0x%04X, want 0x5678", c2.TextVRAM[43])
	}
	if c2.TextVRAM_Front[10] != 0xABCD {
		t.Errorf("TextVRAM_Front[10]: got 0x%04X, want 0xABCD", c2.TextVRAM_Front[10])
	}
}

// ── Ticket 3: VFS state export & import ────────────────────────────────────

func TestCPU_HibernateVFS(t *testing.T) {
	c1 := NewCPU()

	if err := c1.Disk.Write("test.txt", []byte("hello")); err != nil {
		t.Fatalf("VFS Write: %v", err)
	}

	created, _, err := c1.Disk.GetMeta("test.txt")
	if err != nil {
		t.Fatalf("GetMeta: %v", err)
	}

	data, err := c1.HibernateToBytes()
	if err != nil {
		t.Fatalf("HibernateToBytes: %v", err)
	}

	c2 := NewCPU()
	if err := c2.RestoreFromBytes(data); err != nil {
		t.Fatalf("RestoreFromBytes: %v", err)
	}

	content, err := c2.Disk.Read("test.txt")
	if err != nil {
		t.Fatalf("Read after restore: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("content: got %q, want %q", string(content), "hello")
	}

	restoredCreated, _, err := c2.Disk.GetMeta("test.txt")
	if err != nil {
		t.Fatalf("GetMeta after restore: %v", err)
	}
	if !restoredCreated.Equal(created) {
		t.Errorf("Created timestamp mismatch: got %v, want %v", restoredCreated, created)
	}

	size, err := c2.Disk.Size("test.txt")
	if err != nil {
		t.Fatalf("Size after restore: %v", err)
	}
	if size != 5 {
		t.Errorf("Size: got %d, want 5", size)
	}

	if !c2.Disk.Dirty {
		t.Error("Disk.Dirty should be true after restore")
	}
}

// ── Ticket 4: Peripheral registry & state restoration ─────────────────────

// mockStatefulPeripheral is a minimal Peripheral + StatefulPeripheral used in tests.
type mockStatefulPeripheral struct {
	value uint16
}

func (m *mockStatefulPeripheral) Read16(_ uint16) uint16  { return m.value }
func (m *mockStatefulPeripheral) Write16(_ uint16, v uint16) { m.value = v }
func (m *mockStatefulPeripheral) Step()                    {}
func (m *mockStatefulPeripheral) Type() string             { return "MockStateful" }

func (m *mockStatefulPeripheral) SaveState() []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, m.value)
	return b
}

func (m *mockStatefulPeripheral) LoadState(data []byte) error {
	if len(data) < 2 {
		return nil
	}
	m.value = binary.LittleEndian.Uint16(data)
	return nil
}

func TestCPU_HibernatePeripherals(t *testing.T) {
	// Register the mock factory.
	RegisterPeripheral("MockStateful", func(c *CPU, slot uint8) Peripheral {
		return &mockStatefulPeripheral{}
	})

	c1 := NewCPU()
	mock := &mockStatefulPeripheral{value: 0xBEEF}
	c1.MountPeripheral(2, mock)

	data, err := c1.HibernateToBytes()
	if err != nil {
		t.Fatalf("HibernateToBytes: %v", err)
	}

	c2 := NewCPU()
	if err := c2.RestoreFromBytes(data); err != nil {
		t.Fatalf("RestoreFromBytes: %v", err)
	}

	if c2.Peripherals[2] == nil {
		t.Fatal("Peripheral slot 2 is nil after restore")
	}

	restored, ok := c2.Peripherals[2].(*mockStatefulPeripheral)
	if !ok {
		t.Fatalf("Peripheral slot 2 is %T, want *mockStatefulPeripheral", c2.Peripherals[2])
	}

	if restored.value != 0xBEEF {
		t.Errorf("peripheral value: got 0x%04X, want 0xBEEF", restored.value)
	}
}

// ── Ticket 5: End-to-end execution resume ─────────────────────────────────

func TestCPU_HibernateAndResume(t *testing.T) {
	// Program: infinite counter — R3 increments by 1 each loop iteration.
	//   addr 0: LDI R0, <immediate>   (2 words = 4 bytes)
	//   addr 4: ADD R3, R0            (1 word  = 2 bytes)
	//   addr 6: JMP <immediate 4>     (2 words = 4 bytes)
	c1 := NewCPU()
	loadProgram(c1,
		EncodeInstruction(OpLDI, 0, 0, 0), 1, // LDI R0, 1
		EncodeInstruction(OpADD, 3, 0, 0),     // ADD R3, R0
		EncodeInstruction(OpJMP, 0, 0, 0), 4,  // JMP → addr 4
	)

	// Run 50 steps on the original CPU.
	for i := 0; i < 50; i++ {
		c1.Step()
	}

	// Hibernate at step 50.
	hibernated, err := c1.HibernateToBytes()
	if err != nil {
		t.Fatalf("HibernateToBytes: %v", err)
	}

	// Run 50 more steps on the original.
	for i := 0; i < 50; i++ {
		c1.Step()
	}

	// Restore into a fresh CPU and run the same 50 steps.
	c2 := NewCPU()
	if err := c2.RestoreFromBytes(hibernated); err != nil {
		t.Fatalf("RestoreFromBytes: %v", err)
	}
	for i := 0; i < 50; i++ {
		c2.Step()
	}

	// Both CPUs must be in identical states after their respective 100 steps.
	if c1.Regs != c2.Regs {
		t.Errorf("Regs mismatch after resume: c1=%v c2=%v", c1.Regs, c2.Regs)
	}
	if c1.PC != c2.PC {
		t.Errorf("PC mismatch: c1=0x%04X c2=0x%04X", c1.PC, c2.PC)
	}
}
