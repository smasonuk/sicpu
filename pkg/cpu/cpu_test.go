package cpu

import (
	"bytes"
	"testing"
)

// w16 writes a uint16 value as 2 LE bytes at addr. Used to load instructions/data.
func w16(c *CPU, addr uint16, val uint16) {
	c.Memory[addr] = byte(val & 0xFF)
	c.Memory[addr+1] = byte(val >> 8)
}

// loadProgram loads a slice of uint16 words into memory starting at address 0.
func loadProgram(c *CPU, words ...uint16) {
	addr := uint16(0)
	for _, w := range words {
		w16(c, addr, w)
		addr += 2
	}
}

func TestInstructionEncoding(t *testing.T) {
	// 0x0C = 0000 1100. SHL is 0x0C.
	// RegA=2 (010), RegB=0 (000).
	// Opcode << 10 = 0000 1100 << 10 = 0011 0000 0000 0000 = 0x3000
	// RegA << 7  = 2 << 7  = 0000 0001 0000 0000 = 0x0100
	// RegB << 4  = 0
	// Total = 0x3100
	encoded := EncodeInstruction(OpSHL, 2, 0, 0)
	if encoded != 0x3100 {
		t.Errorf("EncodeInstruction(OpSHL, 2, 0, 0): expected 0x3100, got 0x%04X", encoded)
	}
}

func TestALU(t *testing.T) {
	// ADD
	cpu := NewCPU()
	cpu.Regs[RegA] = 10
	cpu.Regs[RegB] = 20
	loadProgram(cpu,
		EncodeInstruction(OpADD, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 30 {
		t.Errorf("OpADD: expected 30, got %d", cpu.Regs[RegA])
	}
	if cpu.Z {
		t.Errorf("OpADD: expected Z=false")
	}

	// SUB (Zero result)
	cpu = NewCPU()
	cpu.Regs[RegA] = 10
	cpu.Regs[RegB] = 10
	loadProgram(cpu,
		EncodeInstruction(OpSUB, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0 {
		t.Errorf("OpSUB: expected 0, got %d", cpu.Regs[RegA])
	}
	if !cpu.Z {
		t.Errorf("OpSUB: expected Z=true")
	}

	// AND
	cpu = NewCPU()
	cpu.Regs[RegA] = 0x00FF
	cpu.Regs[RegB] = 0x0F0F
	loadProgram(cpu,
		EncodeInstruction(OpAND, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0x000F {
		t.Errorf("OpAND: expected 0x000F, got 0x%04X", cpu.Regs[RegA])
	}

	// OR
	cpu = NewCPU()
	cpu.Regs[RegA] = 0x00F0
	cpu.Regs[RegB] = 0x000F
	loadProgram(cpu,
		EncodeInstruction(OpOR, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0x00FF {
		t.Errorf("OpOR: expected 0x00FF, got 0x%04X", cpu.Regs[RegA])
	}

	// XOR
	cpu = NewCPU()
	cpu.Regs[RegA] = 0xFFFF
	cpu.Regs[RegB] = 0x00FF
	loadProgram(cpu,
		EncodeInstruction(OpXOR, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0xFF00 {
		t.Errorf("OpXOR: expected 0xFF00, got 0x%04X", cpu.Regs[RegA])
	}

	// NOT
	cpu = NewCPU()
	cpu.Regs[RegA] = 0x0000
	loadProgram(cpu,
		EncodeInstruction(OpNOT, RegA, 0, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0xFFFF {
		t.Errorf("OpNOT: expected 0xFFFF, got 0x%04X", cpu.Regs[RegA])
	}

	// SHL
	cpu = NewCPU()
	cpu.Regs[RegA] = 0x0001
	cpu.Regs[RegB] = 1
	loadProgram(cpu,
		EncodeInstruction(OpSHL, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0x0002 {
		t.Errorf("OpSHL: expected 0x0002, got 0x%04X", cpu.Regs[RegA])
	}

	// SHR
	cpu = NewCPU()
	cpu.Regs[RegA] = 0x0002
	cpu.Regs[RegB] = 1
	loadProgram(cpu,
		EncodeInstruction(OpSHR, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0x0001 {
		t.Errorf("OpSHR: expected 0x0001, got 0x%04X", cpu.Regs[RegA])
	}
}

func TestJumps(t *testing.T) {
	// JMP: jump to byte addr 0x000A (10)
	cpu := NewCPU()
	loadProgram(cpu,
		EncodeInstruction(OpJMP, 0, 0, 0), 0x000A, // 0x0000: JMP 0x000A
		EncodeInstruction(OpHLT, 0, 0, 0),          // 0x0004: HLT (not reached)
	)
	w16(cpu, 0x000A, EncodeInstruction(OpHLT, 0, 0, 0)) // 0x000A: HLT
	cpu.Step() // Execute JMP
	cpu.Step() // Execute HLT at 0x000A
	if cpu.PC != 0x000C {
		t.Errorf("OpJMP: expected PC=0x000C, got 0x%04X", cpu.PC)
	}

	// JZ taken: jump to byte addr 0x000A
	cpu = NewCPU()
	cpu.Z = true
	loadProgram(cpu,
		EncodeInstruction(OpJZ, 0, 0, 0), 0x000A,
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	w16(cpu, 0x000A, EncodeInstruction(OpHLT, 0, 0, 0))
	cpu.Step()
	if cpu.PC != 0x000A {
		t.Errorf("OpJZ taken: expected PC=0x000A, got 0x%04X", cpu.PC)
	}

	// JZ not taken
	cpu = NewCPU()
	cpu.Z = false
	loadProgram(cpu,
		EncodeInstruction(OpJZ, 0, 0, 0), 0x000A,
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Step()
	if cpu.PC != 0x0004 {
		t.Errorf("OpJZ not taken: expected PC=0x0004, got 0x%04X", cpu.PC)
	}

	// JNZ taken
	cpu = NewCPU()
	cpu.Z = false
	loadProgram(cpu,
		EncodeInstruction(OpJNZ, 0, 0, 0), 0x000A,
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	w16(cpu, 0x000A, EncodeInstruction(OpHLT, 0, 0, 0))
	cpu.Step()
	if cpu.PC != 0x000A {
		t.Errorf("OpJNZ taken: expected PC=0x000A, got 0x%04X", cpu.PC)
	}

	// JN taken
	cpu = NewCPU()
	cpu.N = true
	loadProgram(cpu,
		EncodeInstruction(OpJN, 0, 0, 0), 0x000A,
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	w16(cpu, 0x000A, EncodeInstruction(OpHLT, 0, 0, 0))
	cpu.Step()
	if cpu.PC != 0x000A {
		t.Errorf("OpJN taken: expected PC=0x000A, got 0x%04X", cpu.PC)
	}
}

func TestStack(t *testing.T) {
	// PUSH: SP starts at 0xB5FE, after push -> 0xB5FC
	cpu := NewCPU()
	cpu.Regs[RegA] = 0x1234
	loadProgram(cpu,
		EncodeInstruction(OpPUSH, RegA, 0, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.SP != 0xB5FC {
		t.Errorf("OpPUSH: expected SP=0xB5FC, got 0x%04X", cpu.SP)
	}
	if cpu.Read16(0xB5FC) != 0x1234 {
		t.Errorf("OpPUSH: expected Memory[0xB5FC]=0x1234, got 0x%04X", cpu.Read16(0xB5FC))
	}

	// POP
	cpu = NewCPU()
	cpu.Write16(0xB5FC, 0x5678)
	cpu.SP = 0xB5FC
	loadProgram(cpu,
		EncodeInstruction(OpPOP, RegA, 0, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.SP != 0xB5FE {
		t.Errorf("OpPOP: expected SP=0xB5FE, got 0x%04X", cpu.SP)
	}
	if cpu.Regs[RegA] != 0x5678 {
		t.Errorf("OpPOP: expected R0=0x5678, got 0x%04X", cpu.Regs[RegA])
	}
}

func TestSubroutine(t *testing.T) {
	// CALL to byte address 0x0020; RET returns to instruction after CALL
	cpu := NewCPU()
	loadProgram(cpu,
		EncodeInstruction(OpCALL, 0, 0, 0), 0x0020, // 0x0000: CALL 0x0020 (4 bytes)
		EncodeInstruction(OpHLT, 0, 0, 0),           // 0x0004: HLT
	)
	w16(cpu, 0x0020, EncodeInstruction(OpRET, 0, 0, 0)) // 0x0020: RET

	cpu.Run()
	if cpu.PC != 0x0006 {
		t.Errorf("Subroutine: expected PC=0x0006 (after HLT), got 0x%04X", cpu.PC)
	}
}

func TestInterrupts(t *testing.T) {
	// Verify PushKey triggers an interrupt
	pushKeyCPU := NewCPU()
	pushKeyCPU.PushKey(65)
	if !pushKeyCPU.InterruptPending {
		t.Errorf("PushKey: expected InterruptPending=true after PushKey")
	}

	cpu := NewCPU()

	// Enable interrupts
	w16(cpu, 0x0000, EncodeInstruction(OpEI, 0, 0, 0))  // 0x0000
	w16(cpu, 0x0002, EncodeInstruction(OpNOP, 0, 0, 0)) // 0x0002
	w16(cpu, 0x0004, EncodeInstruction(OpHLT, 0, 0, 0)) // 0x0004

	// Interrupt handler at 0x0010
	w16(cpu, 0x0010, EncodeInstruction(OpNOP, 0, 0, 0))  // NOP in handler
	w16(cpu, 0x0012, EncodeInstruction(OpRETI, 0, 0, 0)) // Return

	cpu.Step() // Execute EI. PC=0x0002. IE=true.
	if !cpu.IE {
		t.Errorf("OpEI: expected IE=true")
	}

	cpu.TriggerInterrupt()
	if !cpu.InterruptPending {
		t.Errorf("TriggerInterrupt: expected InterruptPending=true")
	}

	// Next Step: interrupt dispatched, then NOP at 0x0010 executed
	cpu.Step()

	if cpu.PC != 0x0012 {
		t.Errorf("Interrupt: expected PC=0x0012 (after NOP), got 0x%04X", cpu.PC)
	}
	if cpu.IE {
		t.Errorf("Interrupt: expected IE=false inside handler")
	}
	// SP should be 0xB5FC (started 0xB5FE, pushed 2 bytes)
	if cpu.SP != 0xB5FC {
		t.Errorf("Interrupt: expected SP=0xB5FC, got 0x%04X", cpu.SP)
	}
	// Pushed PC was 0x0002
	if cpu.Read16(cpu.SP) != 0x0002 {
		t.Errorf("Interrupt: expected pushed PC=0x0002, got 0x%04X", cpu.Read16(cpu.SP))
	}

	cpu.Step() // Execute RETI at 0x0012
	// RETI: PC = Pop() = 0x0002. SP = 0xB5FE. IE = true.
	if cpu.PC != 0x0002 {
		t.Errorf("RETI: expected PC=0x0002, got 0x%04X", cpu.PC)
	}
	if !cpu.IE {
		t.Errorf("RETI: expected IE=true")
	}

	// Check DI
	w16(cpu, 0x0002, EncodeInstruction(OpDI, 0, 0, 0))
	cpu.Run()
	if cpu.IE {
		t.Errorf("OpDI: expected IE=false")
	}
}

func TestIO(t *testing.T) {
	cpu := NewCPU()
	loadProgram(cpu,
		EncodeInstruction(OpLDI, RegA, 0, 0), 0xFF00, // LDI R0, 0xFF00
		EncodeInstruction(OpLDI, RegB, 0, 0), 65,      // LDI R1, 65 ('A')
		EncodeInstruction(OpST, RegA, RegB, 0),         // ST [R0], R1
		EncodeInstruction(OpLDI, RegA, 0, 0), 0xFF01,  // LDI R0, 0xFF01
		EncodeInstruction(OpST, RegA, RegB, 0),         // ST [R0], R1
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
}

func TestReadMem(t *testing.T) {
	cpu := NewCPU()
	cpu.Write16(0x1234, 0x5678)
	val := cpu.ReadMem(0x1234)
	if val != 0x5678 {
		t.Errorf("ReadMem: expected 0x5678, got 0x%04X", val)
	}
}

func TestWriteMem_Standard(t *testing.T) {
	cpu := NewCPU()
	cpu.WriteMem(0x1000, 0x9ABC)
	if cpu.Read16(0x1000) != 0x9ABC {
		t.Errorf("WriteMem_Standard: expected 0x9ABC, got 0x%04X", cpu.Read16(0x1000))
	}
}

func TestWriteMem_MMIO(t *testing.T) {
	cpu := NewCPU()
	var buf bytes.Buffer
	cpu.Output = &buf

	// MMIO write should output char, not store in Memory
	savedLo := cpu.Memory[0xFF00]
	savedHi := cpu.Memory[0xFF01]

	cpu.WriteMem(0xFF00, 65) // 'A'

	if buf.String() != "A" {
		t.Errorf("WriteMem_MMIO: expected output 'A', got '%s'", buf.String())
	}

	// Memory at 0xFF00 should not be modified by MMIO write
	if cpu.Memory[0xFF00] != savedLo || cpu.Memory[0xFF01] != savedHi {
		t.Errorf("WriteMem_MMIO: Memory[0xFF00/01] should be unchanged")
	}
}

func TestTextVRAM_Write(t *testing.T) {
	cpu := NewCPU()
	// Write 16-bit value 0x0041 to address 0xF600 + 5*2 = 0xF60A (word index 5)
	cpu.WriteMem(0xF60A, 0x0041)
	if cpu.TextVRAM[5] != 0x0041 {
		t.Errorf("TextVRAM_Write: expected TextVRAM[5]=0x0041, got 0x%04X", cpu.TextVRAM[5])
	}
	// Main memory at that address should be zero (TextVRAM is separate)
	if cpu.Memory[0xF60A] != 0 {
		t.Errorf("TextVRAM_Write: expected Memory[0xF60A]=0, got 0x%02X", cpu.Memory[0xF60A])
	}
}

func TestTextVRAM_Read(t *testing.T) {
	cpu := NewCPU()
	cpu.TextVRAM[1023] = 0x0042
	// Word index 1023 is at byte address 0xF600 + 1023*2 = 0xF600 + 0x07FE = 0xFDFE
	val := cpu.ReadMem(0xFDFE)
	if val != 0x0042 {
		t.Errorf("TextVRAM_Read: expected 0x0042, got 0x%04X", val)
	}
}

func TestTextVRAM_Bounds(t *testing.T) {
	cpu := NewCPU()
	// Below VRAM range (Base RAM)
	cpu.WriteMem(0xB5FE, 0x1111)
	// Above VRAM range (Memory above MMIO)
	cpu.WriteMem(0xFF30, 0x2222)

	if cpu.Read16(0xB5FE) != 0x1111 {
		t.Errorf("TextVRAM_Bounds: expected 0x1111 at 0xB5FE, got 0x%04X", cpu.Read16(0xB5FE))
	}
	if cpu.Read16(0xFF30) != 0x2222 {
		t.Errorf("TextVRAM_Bounds: expected 0x2222 at 0xFF30, got 0x%04X", cpu.Read16(0xFF30))
	}
	// TextVRAM should be untouched
	for i := 0; i < 1024; i++ {
		if cpu.TextVRAM[i] != 0 {
			t.Errorf("TextVRAM_Bounds: expected TextVRAM[%d]=0, got 0x%04X", i, cpu.TextVRAM[i])
		}
	}
}

func TestBankSwitching_Write(t *testing.T) {
	cpu := NewCPU()
	cpu.WriteMem(0xFF02, 0)
	cpu.WriteMem(0xB600, 0xAAAA)

	cpu.WriteMem(0xFF02, 1)
	cpu.WriteMem(0xB600, 0xBBBB)

	// GraphicsBanks are now [4][16384]byte; low byte of 0xAAAA is 0xAA
	if cpu.GraphicsBanks[0][0] != 0xAA {
		t.Errorf("BankSwitching_Write: expected GraphicsBanks[0][0]=0xAA, got 0x%02X", cpu.GraphicsBanks[0][0])
	}
	if cpu.GraphicsBanks[0][1] != 0xAA {
		t.Errorf("BankSwitching_Write: expected GraphicsBanks[0][1]=0xAA, got 0x%02X", cpu.GraphicsBanks[0][1])
	}
	if cpu.GraphicsBanks[1][0] != 0xBB {
		t.Errorf("BankSwitching_Write: expected GraphicsBanks[1][0]=0xBB, got 0x%02X", cpu.GraphicsBanks[1][0])
	}
	if cpu.Memory[0xB600] != 0 {
		t.Errorf("BankSwitching_Write: expected Memory[0xB600]=0, got 0x%02X", cpu.Memory[0xB600])
	}
}

func TestBankSwitching_Read(t *testing.T) {
	cpu := NewCPU()
	cpu.WriteMem(0xFF02, 1)
	// Write 0xCCCC into bank 1 at offset 16382/16383 (last two bytes)
	cpu.GraphicsBanks[1][16382] = 0xCC
	cpu.GraphicsBanks[1][16383] = 0xCC

	val := cpu.ReadMem(0xF5FE) // 0xB600 + 16382 = 0xF5FE
	if val != 0xCCCC {
		t.Errorf("BankSwitching_Read: expected 0xCCCC, got 0x%04X", val)
	}
}

func TestBankRegister_Masking(t *testing.T) {
	cpu := NewCPU()
	cpu.WriteMem(0xFF02, 0xFFFF)
	if cpu.CurrentBank != 3 {
		t.Errorf("BankRegister_Masking: expected CurrentBank=3, got %d", cpu.CurrentBank)
	}
}

func TestVRAMConfigRegister_Write(t *testing.T) {
	cpu := NewCPU()

	cpu.WriteMem(0xFF03, 1)
	if cpu.TextResolutionMode != 1 {
		t.Errorf("VRAMConfigRegister_Write: expected TextResolutionMode=1, got %d", cpu.TextResolutionMode)
	}

	cpu.WriteMem(0xFF03, 0xFFFF)
	if cpu.TextResolutionMode != 1 {
		t.Errorf("VRAMConfigRegister_Write: expected TextResolutionMode=1 after masking, got %d", cpu.TextResolutionMode)
	}

	cpu.WriteMem(0xFF03, 0)
	if cpu.TextResolutionMode != 0 {
		t.Errorf("VRAMConfigRegister_Write: expected TextResolutionMode=0, got %d", cpu.TextResolutionMode)
	}
}

func TestKeyboardBuffer_Read(t *testing.T) {
	cpu := NewCPU()
	cpu.PushKey(65) // 'A'
	cpu.PushKey(66) // 'B'

	if val := cpu.ReadMem(0xFF04); val != 65 {
		t.Errorf("KeyboardBuffer_Read: expected 65, got %d", val)
	}
	if val := cpu.ReadMem(0xFF04); val != 66 {
		t.Errorf("KeyboardBuffer_Read: expected 66, got %d", val)
	}
	if val := cpu.ReadMem(0xFF04); val != 0 {
		t.Errorf("KeyboardBuffer_Read empty: expected 0, got %d", val)
	}
}

func TestVRAMConfigRegister_Read(t *testing.T) {
	cpu := NewCPU()
	cpu.TextResolutionMode = 1

	val := cpu.ReadMem(0xFF03)
	if val != 1 {
		t.Errorf("VRAMConfigRegister_Read: expected 1, got %d", val)
	}
}

func TestOpMUL(t *testing.T) {
	cpu := NewCPU()
	cpu.Regs[RegA] = 10
	cpu.Regs[RegB] = 5
	loadProgram(cpu,
		EncodeInstruction(OpMUL, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 50 {
		t.Errorf("OpMUL: expected 50, got %d", cpu.Regs[RegA])
	}
}

func TestOpDIV(t *testing.T) {
	cpu := NewCPU()
	cpu.Regs[RegA] = 50
	cpu.Regs[RegB] = 5
	loadProgram(cpu,
		EncodeInstruction(OpDIV, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 10 {
		t.Errorf("OpDIV: expected 10, got %d", cpu.Regs[RegA])
	}

	// Test division by zero
	cpu = NewCPU()
	cpu.Regs[RegA] = 50
	cpu.Regs[RegB] = 0
	loadProgram(cpu,
		EncodeInstruction(OpDIV, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0 {
		t.Errorf("OpDIV by zero: expected 0, got %d", cpu.Regs[RegA])
	}
	if !cpu.Z {
		t.Errorf("OpDIV by zero: expected Z=true")
	}
}

func TestBitmapEnable_DefaultOff(t *testing.T) {
	c := NewCPU()
	if c.GraphicsEnabled {
		t.Error("expected GraphicsEnabled=false on a fresh CPU")
	}
	if !c.TextOverlay {
		t.Error("expected TextOverlay=true on a fresh CPU")
	}
}

func TestBitmapEnable(t *testing.T) {
	c := NewCPU()
	c.WriteMem(0xFF05, 0x02)
	if !c.GraphicsEnabled {
		t.Error("expected GraphicsEnabled=true after writing 0x02 to 0xFF05")
	}
	if c.TextOverlay {
		t.Error("expected TextOverlay=false after writing 0x02 to 0xFF05")
	}
}

func TestBitmapEnable_ReadBack(t *testing.T) {
	c := NewCPU()
	c.WriteMem(0xFF05, 0x03)
	got := c.ReadMem(0xFF05)
	if got != 0x03 {
		t.Errorf("ReadMem(0xFF05) after writing 0x03: expected 0x03, got 0x%04X", got)
	}
}

func TestBufferedMode(t *testing.T) {
	c := NewCPU()
	if c.BufferedMode {
		t.Error("BufferedMode should be false by default")
	}
	c.WriteMem(0xFF05, 0x04)
	if !c.BufferedMode {
		t.Error("BufferedMode should be true after writing 0x04 to 0xFF05")
	}
	c.WriteMem(0xFF05, 0x00)
	if c.BufferedMode {
		t.Error("BufferedMode should be false after writing 0x00 to 0xFF05")
	}
}

func TestVideoFlip(t *testing.T) {
	c := NewCPU()

	// Write unique bytes to GraphicsBanks[2][0..1]
	c.CurrentBank = 2
	c.GraphicsBanks[2][0] = 0xEF
	c.GraphicsBanks[2][1] = 0xBE

	// Write to TextVRAM
	c.TextVRAM[0] = 0xCAFE

	if c.GraphicsBanksFront[2][0] != 0 {
		t.Errorf("Expected Front buffer to be 0, got 0x%X", c.GraphicsBanksFront[2][0])
	}
	if c.TextVRAM_Front[0] != 0 {
		t.Errorf("Expected Front TextVRAM to be 0, got 0x%X", c.TextVRAM_Front[0])
	}

	c.WriteMem(0xFF06, 2)

	if c.DisplayBank != 2 {
		t.Errorf("Expected DisplayBank=2, got %d", c.DisplayBank)
	}
	if c.GraphicsBanksFront[2][0] != 0xEF {
		t.Errorf("Expected Front buffer[0] = 0xEF, got 0x%X", c.GraphicsBanksFront[2][0])
	}
	if c.GraphicsBanksFront[2][1] != 0xBE {
		t.Errorf("Expected Front buffer[1] = 0xBE, got 0x%X", c.GraphicsBanksFront[2][1])
	}
	if c.TextVRAM_Front[0] != 0xCAFE {
		t.Errorf("Expected Front TextVRAM to contain 0xCAFE, got 0x%X", c.TextVRAM_Front[0])
	}
}

func TestOpFILL(t *testing.T) {
	c := NewCPU()
	// FILL R0, R1, R2
	// R0 = Start Address = 0x1000
	// R1 = Count = 10 (16-bit words)
	// R2 = Value = 0xAA55

	c.Regs[RegA] = 0x1000
	c.Regs[RegB] = 10
	c.Regs[RegC] = 0xAA55

	instr := EncodeInstruction(OpFILL, RegA, RegB, RegC)
	loadProgram(c,
		instr,
		EncodeInstruction(OpHLT, 0, 0, 0),
	)

	c.Run()

	for i := uint16(0); i < 10; i++ {
		if val := c.Read16(0x1000 + i*2); val != 0xAA55 {
			t.Errorf("FILL: Memory[0x%04X] = 0x%04X; want 0xAA55", 0x1000+i*2, val)
		}
	}

	// Next word should be untouched
	if val := c.Read16(0x1000 + 10*2); val != 0 {
		t.Errorf("FILL: Memory[0x%04X] = 0x%04X; want 0", 0x1000+10*2, val)
	}
}

func TestOpCOPY(t *testing.T) {
	// Case 1: Non-overlapping copy
	c := NewCPU()
	c.Regs[RegA] = 0x1000
	c.Regs[RegB] = 0x2000
	c.Regs[RegC] = 4

	// Init source: 4 uint16 words at 0x1000
	c.Write16(0x1000, 1)
	c.Write16(0x1002, 2)
	c.Write16(0x1004, 3)
	c.Write16(0x1006, 4)

	instr := EncodeInstruction(OpCOPY, RegA, RegB, RegC)
	loadProgram(c,
		instr,
		EncodeInstruction(OpHLT, 0, 0, 0),
	)

	c.Run()

	for i := uint16(0); i < 4; i++ {
		if val := c.Read16(0x2000 + i*2); val != uint16(i+1) {
			t.Errorf("Non-overlap: Memory[0x%04X] = %d; want %d", 0x2000+i*2, val, i+1)
		}
	}

	// Case 2: Overlapping copy (Src < Dst)
	// Src=0x1000, Dst=0x1002, Count=3 words
	// words: [1,2,3,4] -> backwards copy -> [1,1,2,3]
	c = NewCPU()
	c.Regs[RegA] = 0x1000
	c.Regs[RegB] = 0x1002
	c.Regs[RegC] = 3

	c.Write16(0x1000, 1)
	c.Write16(0x1002, 2)
	c.Write16(0x1004, 3)
	c.Write16(0x1006, 4)

	instr = EncodeInstruction(OpCOPY, RegA, RegB, RegC)
	loadProgram(c,
		instr,
		EncodeInstruction(OpHLT, 0, 0, 0),
	)

	c.Run()

	expected := []uint16{1, 1, 2, 3}
	for i, exp := range expected {
		if val := c.Read16(0x1000 + uint16(i)*2); val != exp {
			t.Errorf("Overlap Src<Dst: Memory[0x%04X] = %d; want %d", 0x1000+uint16(i)*2, val, exp)
		}
	}

	// Case 3: Overlapping copy (Dst < Src)
	// Src=0x1002, Dst=0x1000, Count=3 words
	// [1,2,3,4] -> [2,3,4,4]
	c = NewCPU()
	c.Regs[RegA] = 0x1002
	c.Regs[RegB] = 0x1000
	c.Regs[RegC] = 3

	c.Write16(0x1000, 1)
	c.Write16(0x1002, 2)
	c.Write16(0x1004, 3)
	c.Write16(0x1006, 4)

	instr = EncodeInstruction(OpCOPY, RegA, RegB, RegC)
	loadProgram(c,
		instr,
		EncodeInstruction(OpHLT, 0, 0, 0),
	)

	c.Run()

	expected = []uint16{2, 3, 4, 4}
	for i, exp := range expected {
		if val := c.Read16(0x1000 + uint16(i)*2); val != exp {
			t.Errorf("Overlap Dst<Src: Memory[0x%04X] = %d; want %d", 0x1000+uint16(i)*2, val, exp)
		}
	}
}

func TestALU_EdgeCases(t *testing.T) {
	// ADD Overflow
	cpu := NewCPU()
	cpu.Regs[RegA] = 0xFFFF
	cpu.Regs[RegB] = 1
	loadProgram(cpu,
		EncodeInstruction(OpADD, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0 {
		t.Errorf("OpADD overflow: expected 0, got 0x%04X", cpu.Regs[RegA])
	}
	if !cpu.Z {
		t.Errorf("OpADD overflow: expected Z=true")
	}

	// SUB Underflow
	cpu = NewCPU()
	cpu.Regs[RegA] = 5
	cpu.Regs[RegB] = 10
	loadProgram(cpu,
		EncodeInstruction(OpSUB, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0xFFFB {
		t.Errorf("OpSUB underflow: expected 0xFFFB, got 0x%04X", cpu.Regs[RegA])
	}
	if !cpu.N {
		t.Errorf("OpSUB underflow: expected N=true")
	}

	// MUL Overflow
	cpu = NewCPU()
	cpu.Regs[RegA] = 0x100
	cpu.Regs[RegB] = 0x100
	loadProgram(cpu,
		EncodeInstruction(OpMUL, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0 {
		t.Errorf("OpMUL overflow: expected 0, got 0x%04X", cpu.Regs[RegA])
	}
	if !cpu.Z {
		t.Errorf("OpMUL overflow: expected Z=true")
	}

	// SHL Overflow
	cpu = NewCPU()
	cpu.Regs[RegA] = 0x8000
	cpu.Regs[RegB] = 1
	loadProgram(cpu,
		EncodeInstruction(OpSHL, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0 {
		t.Errorf("OpSHL overflow: expected 0, got 0x%04X", cpu.Regs[RegA])
	}
	if !cpu.Z {
		t.Errorf("OpSHL overflow: expected Z=true")
	}

	// SHR of 1
	cpu = NewCPU()
	cpu.Regs[RegA] = 1
	cpu.Regs[RegB] = 1
	loadProgram(cpu,
		EncodeInstruction(OpSHR, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0 {
		t.Errorf("OpSHR of 1: expected 0, got 0x%04X", cpu.Regs[RegA])
	}
	if !cpu.Z {
		t.Errorf("OpSHR of 1: expected Z=true")
	}

	// NOT of 0xFFFF
	cpu = NewCPU()
	cpu.Regs[RegA] = 0xFFFF
	loadProgram(cpu,
		EncodeInstruction(OpNOT, RegA, 0, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0 {
		t.Errorf("OpNOT of 0xFFFF: expected 0, got 0x%04X", cpu.Regs[RegA])
	}
	if !cpu.Z {
		t.Errorf("OpNOT of 0xFFFF: expected Z=true")
	}

	// N flag when bit 15 is set
	cpu = NewCPU()
	cpu.Regs[RegA] = 0x7FFF
	cpu.Regs[RegB] = 1
	loadProgram(cpu,
		EncodeInstruction(OpADD, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0x8000 {
		t.Errorf("OpADD N flag test: expected 0x8000, got 0x%04X", cpu.Regs[RegA])
	}
	if !cpu.N {
		t.Errorf("OpADD N flag test: expected N=true")
	}
}

func TestWFI_Wakeup(t *testing.T) {
	cpu := NewCPU()
	w16(cpu, 0x0000, EncodeInstruction(OpEI, 0, 0, 0))  // 0x0000
	w16(cpu, 0x0002, EncodeInstruction(OpWFI, 0, 0, 0)) // 0x0002
	w16(cpu, 0x0004, EncodeInstruction(OpNOP, 0, 0, 0)) // 0x0004 (after WFI)
	w16(cpu, 0x0006, EncodeInstruction(OpHLT, 0, 0, 0)) // 0x0006

	// Interrupt handler at 0x0010
	w16(cpu, 0x0010, EncodeInstruction(OpNOP, 0, 0, 0))  // NOP
	w16(cpu, 0x0012, EncodeInstruction(OpRETI, 0, 0, 0)) // RETI

	cpu.Step() // Execute EI
	if !cpu.IE {
		t.Errorf("OpEI: expected IE=true")
	}

	cpu.Step() // Execute WFI
	if !cpu.Waiting {
		t.Errorf("OpWFI: expected Waiting=true")
	}

	// Step while waiting - PC should not advance
	pcBefore := cpu.PC
	cpu.Step()
	if cpu.PC != pcBefore {
		t.Errorf("Waiting: expected PC to remain 0x%04X, got 0x%04X", pcBefore, cpu.PC)
	}

	// Trigger Interrupt
	cpu.TriggerInterrupt()
	cpu.Step()

	if cpu.Waiting {
		t.Errorf("Interrupt: expected Waiting=false")
	}
	if cpu.PC != 0x0012 {
		t.Errorf("Interrupt: expected PC=0x0012 (in handler after NOP), got 0x%04X", cpu.PC)
	}

	// Execute RETI
	cpu.Step()
	if cpu.PC != 0x0004 {
		t.Errorf("RETI: expected PC=0x0004 (after WFI), got 0x%04X", cpu.PC)
	}
}

func TestStackPointerOps(t *testing.T) {
	cpu := NewCPU()

	// LDSP R0 - SP defaults to 0xB5FE
	loadProgram(cpu,
		EncodeInstruction(OpLDSP, RegA, 0, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.Regs[RegA] != 0xB5FE {
		t.Errorf("OpLDSP: expected R0=0xB5FE, got 0x%04X", cpu.Regs[RegA])
	}

	// STSP R0
	cpu = NewCPU()
	cpu.Regs[RegA] = 0x8000
	loadProgram(cpu,
		EncodeInstruction(OpSTSP, RegA, 0, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()
	if cpu.SP != 0x8000 {
		t.Errorf("OpSTSP: expected SP=0x8000, got 0x%04X", cpu.SP)
	}

	// Round-trip: PUSH, LDSP, STSP, POP
	// SP starts 0xB5FE. After PUSH -> 0xB5FC.
	cpu = NewCPU()
	cpu.Regs[RegA] = 0x1234
	loadProgram(cpu,
		EncodeInstruction(OpPUSH, RegA, 0, 0), // Push 0x1234. SP -> 0xB5FC
		EncodeInstruction(OpLDSP, RegB, 0, 0), // R1 = SP (0xB5FC)
		EncodeInstruction(OpSTSP, RegB, 0, 0), // SP = R1 (0xB5FC)
		EncodeInstruction(OpPOP, RegC, 0, 0),  // Pop into R2. Should be 0x1234.
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()

	if cpu.SP != 0xB5FE {
		t.Errorf("RoundTrip: expected SP=0xB5FE, got 0x%04X", cpu.SP)
	}
	if cpu.Regs[RegC] != 0x1234 {
		t.Errorf("RoundTrip: expected R2=0x1234, got 0x%04X", cpu.Regs[RegC])
	}
}

func TestVirtualDisk_MMIO(t *testing.T) {
	c := NewCPU()

	// 1. Write File: "TEST" containing 2 bytes [0x34, 0x12] at 0x2000
	// Filename "TEST\0" at 0x1000
	c.Memory[0x1000] = 'T'
	c.Memory[0x1001] = 'E'
	c.Memory[0x1002] = 'S'
	c.Memory[0x1003] = 'T'
	c.Memory[0x1004] = 0

	// Data: 2 bytes at 0x2000
	c.Memory[0x2000] = 0x34
	c.Memory[0x2001] = 0x12

	// Set MMIO registers (16-bit LE)
	c.Write16(0xFF11, 0x1000) // NamePtr
	c.Write16(0xFF12, 0x2000) // BufPtr
	c.Write16(0xFF13, 2)      // Length = 2 bytes

	// Trigger Write Command (2)
	c.WriteMem(0xFF10, 2)

	// Check Status
	if c.Read16(0xFF14) != 0 {
		t.Errorf("Write File: Expected Status=0, got %d", c.Read16(0xFF14))
	}

	// Verify file in Disk (2 bytes)
	data, err := c.Disk.Read("TEST")
	if err != nil {
		t.Fatalf("Write File: Failed to read from disk: %v", err)
	}
	if len(data) != 2 || data[0] != 0x34 || data[1] != 0x12 {
		t.Errorf("Write File: Content mismatch. Got %v", data)
	}

	// 2. Read File: "TEST" into 0x3000
	c.Write16(0xFF12, 0x3000) // BufPtr

	// Trigger Read Command (1)
	c.WriteMem(0xFF10, 1)

	if c.Read16(0xFF14) != 0 {
		t.Errorf("Read File: Expected Status=0, got %d", c.Read16(0xFF14))
	}

	// Verify memory content
	if c.Memory[0x3000] != 0x34 || c.Memory[0x3001] != 0x12 {
		t.Errorf("Read File: Memory content mismatch. 0x3000=0x%02X, 0x3001=0x%02X", c.Memory[0x3000], c.Memory[0x3001])
	}

	// 3. Get Size
	c.Write16(0xFF13, 0)
	c.WriteMem(0xFF10, 3)

	if c.Read16(0xFF14) != 0 {
		t.Errorf("Get Size: Expected Status=0, got %d", c.Read16(0xFF14))
	}
	if c.Read16(0xFF13) != 2 {
		t.Errorf("Get Size: Expected Size=2 bytes, got %d", c.Read16(0xFF13))
	}

	// 4. Error: File Not Found
	c.Memory[0x1000] = 'B' // "BEST"
	c.WriteMem(0xFF10, 1)
	if c.Read16(0xFF14) != 1 {
		t.Errorf("Error Not Found: Expected Status=1, got %d", c.Read16(0xFF14))
	}

	// 5. Error: Out of Bounds (Read into 0xFFFF with 2 bytes)
	c.Memory[0x1000] = 'T' // "TEST"
	c.Write16(0xFF12, 0xFFFF)
	c.WriteMem(0xFF10, 1)
	if c.Read16(0xFF14) != 4 {
		t.Errorf("Error Out of Bounds: Expected Status=4, got %d", c.Read16(0xFF14))
	}

	// 6. Error: Invalid Name (Too Long - 20 'A's without null)
	for i := 0; i < 20; i++ {
		c.Memory[0x4000+uint16(i)] = 'A'
	}
	c.Write16(0xFF11, 0x4000)
	c.WriteMem(0xFF10, 3)
	if c.Read16(0xFF14) != 3 {
		t.Errorf("Error Invalid Name: Expected Status=3, got %d", c.Read16(0xFF14))
	}

	// 7. Max Length Filename (16 chars) - Valid format
	namePtr := uint16(0x5000)
	for i := 0; i < 12; i++ {
		c.Memory[namePtr+uint16(i)] = 'A'
	}
	c.Memory[namePtr+12] = '.'
	c.Memory[namePtr+13] = 'T'
	c.Memory[namePtr+14] = 'X'
	c.Memory[namePtr+15] = 'T'
	c.Memory[namePtr+16] = 0

	c.Write16(0xFF11, namePtr)
	c.WriteMem(0xFF10, 3)
	if c.Read16(0xFF14) == 3 {
		t.Errorf("Max Length Name: Expected Status!=3, got 3")
	}

	// 8. Too Long Filename (17 'A's)
	for i := 0; i < 17; i++ {
		c.Memory[namePtr+uint16(i)] = 'A'
	}
	c.Memory[namePtr+17] = 0
	c.WriteMem(0xFF10, 3)
	if c.Read16(0xFF14) != 3 {
		t.Errorf("Too Long Name: Expected Status=3, got %d", c.Read16(0xFF14))
	}
}

func TestLDB_STB(t *testing.T) {
	c := NewCPU()
	// Store single byte 0x42 at address 0x1000
	c.Memory[0x1000] = 0x42
	c.Regs[RegB] = 0x1000
	loadProgram(c,
		EncodeInstruction(OpLDB, RegA, RegB, 0), // LDB R0, [R1]: R0 = Memory[0x1000] = 0x42
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	c.Run()
	if c.Regs[RegA] != 0x42 {
		t.Errorf("LDB: expected R0=0x42, got 0x%04X", c.Regs[RegA])
	}

	// STB: store low byte of register to memory
	c = NewCPU()
	c.Regs[RegA] = 0x1000
	c.Regs[RegB] = 0xABCD // Only 0xCD should be stored
	loadProgram(c,
		EncodeInstruction(OpSTB, RegA, RegB, 0), // STB [R0], R1
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	c.Run()
	if c.Memory[0x1000] != 0xCD {
		t.Errorf("STB: expected Memory[0x1000]=0xCD, got 0x%02X", c.Memory[0x1000])
	}
	// Next byte should be untouched
	if c.Memory[0x1001] != 0 {
		t.Errorf("STB: expected Memory[0x1001]=0x00, got 0x%02X", c.Memory[0x1001])
	}
}

func TestNewRegisters(t *testing.T) {
	cpu := NewCPU()
	// Store 100 in R4
	cpu.Regs[4] = 100
	// Store 200 in R5
	cpu.Regs[5] = 200

	// ADD R4, R5 (Result in R4)
	loadProgram(cpu,
		EncodeInstruction(OpADD, 4, 5, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	cpu.Run()

	if cpu.Regs[4] != 300 {
		t.Errorf("Expected R4 to be 300, got %d", cpu.Regs[4])
	}
}

func TestSignedOps(t *testing.T) {
	// 1. Test OpIDIV (Signed Division)
	c := NewCPU()
	// -10 / 2 = -5
	// -10 in 16-bit = 0xFFF6
	// 2 in 16-bit = 0x0002
	// -5 in 16-bit = 0xFFFB
	c.Regs[RegA] = 0xFFF6
	c.Regs[RegB] = 2
	loadProgram(c,
		EncodeInstruction(OpIDIV, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	c.Run()
	if c.Regs[RegA] != 0xFFFB {
		t.Errorf("OpIDIV (-10 / 2): expected 0xFFFB (-5), got 0x%04X", c.Regs[RegA])
	}
	if !c.N {
		t.Errorf("OpIDIV: expected N=true for negative result")
	}

	// 2. Test OpADD Carry
	// 0xFFFF + 1 = 0 (Carry set)
	c = NewCPU()
	c.Regs[RegA] = 0xFFFF
	c.Regs[RegB] = 1
	loadProgram(c,
		EncodeInstruction(OpADD, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	c.Run()
	if c.Regs[RegA] != 0 {
		t.Errorf("OpADD overflow: expected 0, got 0x%04X", c.Regs[RegA])
	}
	if !c.C {
		t.Errorf("OpADD overflow: expected C=true")
	}

	// 3. Test OpSUB Borrow (Carry)
	// 1 - 2 = -1 (Borrow set)
	// Unsigned: 1 < 2 -> Borrow
	c = NewCPU()
	c.Regs[RegA] = 1
	c.Regs[RegB] = 2
	loadProgram(c,
		EncodeInstruction(OpSUB, RegA, RegB, 0),
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	c.Run()
	if c.Regs[RegA] != 0xFFFF {
		t.Errorf("OpSUB borrow: expected 0xFFFF (-1), got 0x%04X", c.Regs[RegA])
	}
	if !c.C {
		t.Errorf("OpSUB borrow: expected C=true")
	}

	// 4. Test OpJC (Jump if Carry)
	// Calculate 1 - 2 (Sets C), then JC to target
	c = NewCPU()
	c.Regs[RegA] = 1
	c.Regs[RegB] = 2
	loadProgram(c,
		EncodeInstruction(OpSUB, RegA, RegB, 0), // Sets C
		EncodeInstruction(OpJC, 0, 0, 0), 0x000A, // Jump to 0x000A if C
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	w16(c, 0x000A, EncodeInstruction(OpHLT, 0, 0, 0))
	c.Run()
	if c.PC != 0x000C { // 0x000A + 2 (HLT)
		t.Errorf("OpJC taken: expected PC=0x000C, got 0x%04X", c.PC)
	}

	// 5. Test OpJC Not Taken
	// Calculate 2 - 1 (No Borrow), then JC (should not jump)
	c = NewCPU()
	c.Regs[RegA] = 2
	c.Regs[RegB] = 1
	loadProgram(c,
		EncodeInstruction(OpSUB, RegA, RegB, 0), // C=false
		EncodeInstruction(OpJC, 0, 0, 0), 0x000A,
		EncodeInstruction(OpHLT, 0, 0, 0),
	)
	c.Step() // SUB
	c.Step() // JC
	if c.PC != 0x0008 { // 0 + 2 (SUB) + 4 (JC) = 6. Wait.
		// SUB: 2 bytes. Addr 0.
		// JC: 4 bytes. Addr 2.
		// HLT: 2 bytes. Addr 6.
		// So if not taken, PC should be 6.
		// Wait, my mental addressing is off.
		// SUB: 0x0000. PC -> 0x0002.
		// JC: 0x0002. PC -> 0x0006.
		// HLT: 0x0006.
	}
	// Let's rely on c.Run() stopping at HLT.
	c.Run()
	if c.PC != 0x0008 { // HLT at 0x0006 executed -> PC=0x0008
		t.Errorf("OpJC not taken: expected PC=0x0008, got 0x%04X", c.PC)
	}
}

func TestMemoryMapBoundaries(t *testing.T) {
	c := NewCPU()

	// 1. Graphics VRAM (0xB600 - 0xF5FF)
	c.WriteByte(0xB600, 0x42)
	if c.GraphicsBanks[c.CurrentBank][0] != 0x42 {
		t.Errorf("Graphics VRAM: expected 0x42 at index 0, got 0x%02X", c.GraphicsBanks[c.CurrentBank][0])
	}

	// 2. Text VRAM (0xF600 - 0xFDFF)
	c.Write16(0xF600, 0x1234)
	if c.TextVRAM[0] != 0x1234 {
		t.Errorf("Text VRAM: expected 0x1234 at index 0, got 0x%04X", c.TextVRAM[0])
	}

	// 3. Base RAM (0xB5FF)
	c.WriteByte(0xB5FF, 0x77)
	if c.Memory[0xB5FF] != 0x77 {
		t.Errorf("Base RAM: expected 0x77 at 0xB5FF, got 0x%02X", c.Memory[0xB5FF])
	}
}

type dummyPeripheral struct {
	lastVal uint16
}
func (p *dummyPeripheral) Read16(offset uint16) uint16 { return p.lastVal }
func (p *dummyPeripheral) Write16(offset uint16, val uint16) { p.lastVal = val }
func (p *dummyPeripheral) Step() {}
func (p *dummyPeripheral) Type() string { return "Dummy" }

func TestPeripheralRouting(t *testing.T) {
	c := NewCPU()
	p := &dummyPeripheral{}
	c.MountPeripheral(0, p)

	// Expansion Bus starts at 0xFE00. Slot 0 is 0xFE00-0xFE0F.
	c.Write16(0xFE00, 0x1337)
	if p.lastVal != 0x1337 {
		t.Errorf("Peripheral Routing: expected 0x1337, got 0x%04X", p.lastVal)
	}
	
	val := c.Read16(0xFE00)
	if val != 0x1337 {
		t.Errorf("Peripheral Reading: expected 0x1337, got 0x%04X", val)
	}
}
