package asm

import (
	"testing"
)

func TestAssembleSourceMap(t *testing.T) {
	code := `
; Line 1: Comment
LDI R0, 10      ; Line 3: Instruction (4 bytes: 2 opcode + 2 immediate)
                ; Line 4: Empty
LABEL:          ; Line 5: Label
ADD R0, R1      ; Line 6: Instruction (2 bytes)
.ORG 0x0010     ; Line 7: ORG (padding to byte addr 0x0010)
HLT             ; Line 8: Instruction (2 bytes at byte addr 0x0010)
.STRING "AB"    ; Line 9: String (3 bytes: 'A', 'B', 0)
`
	// Expected byte layout:
	// Byte addr 0x0000: LDI R0, 10 opcode (from Line 3)
	// Byte addr 0x0001: LDI opcode high byte
	// Byte addr 0x0002: immediate low byte (10)
	// Byte addr 0x0003: immediate high byte (0)
	// Byte addr 0x0004: ADD R0, R1 (from Line 6). Label LABEL on Line 5 points here (byte 4).
	// Byte addr 0x0005: ADD high byte
	// .ORG 0x0010 adds padding from byte 6 to byte 15.
	// Byte addr 0x0010 (16): HLT (from Line 8)
	// Byte addr 0x0011: HLT high byte
	// Byte addr 0x0012 (18): 'A' (from Line 9)
	// Byte addr 0x0013 (19): 'B'
	// Byte addr 0x0014 (20): 0

	// Expected sourceMap:
	// 0x0000 -> 3  (LDI)
	// 0x0004 -> 6  (ADD, also where LABEL points)
	// 0x0010 -> 8  (HLT)
	// 0x0012 -> 9  (.STRING)

	_, sourceMap, err := Assemble(code)
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	tests := []struct {
		addr uint16
		line int
	}{
		{0x0000, 3},
		{0x0004, 6},
		{0x0010, 8},
		{0x0012, 9},
	}

	for _, tc := range tests {
		if got := sourceMap[tc.addr]; got != tc.line {
			t.Errorf("sourceMap[0x%04X] = %d; want %d", tc.addr, got, tc.line)
		}
	}
}
