package asm

import (
	"gocpu/pkg/cpu"
	"reflect"
	"testing"
)

// encodeWords converts a slice of uint16 to little-endian bytes.
// This helps update test cases from the old []uint16 format.
func encodeWords(words ...uint16) []byte {
	out := make([]byte, len(words)*2)
	for i, w := range words {
		out[i*2] = byte(w & 0xFF)
		out[i*2+1] = byte(w >> 8)
	}
	return out
}

func TestHelperFunctions(t *testing.T) {
	// Test isIdentifier
	tests := []struct {
		input string
		want  bool
	}{
		{"abc", true},
		{"_abc", true},
		{"abc1", true},
		{"1abc", false},
		{"", false},
		{"ab-c", false},
	}
	for _, tc := range tests {
		if got := isIdentifier(tc.input); got != tc.want {
			t.Errorf("isIdentifier(%q) = %v; want %v", tc.input, got, tc.want)
		}
	}

	// Test normalizeLabel
	if got := normalizeLabel("label"); got != "LABEL" {
		t.Errorf("normalizeLabel(\"label\") = %q; want \"LABEL\"", got)
	}

	// Test instructionLength (now in bytes)
	lenTests := []struct {
		mnemonic string
		wantLen  uint16
		wantOk   bool
	}{
		{"HLT", 2, true},
		{"NOP", 2, true},
		{"LDI", 4, true},
		{"JMP", 4, true},
		{"INVALID", 0, false},
	}
	for _, tc := range lenTests {
		gotLen, gotOk := instructionLength(tc.mnemonic)
		if gotLen != tc.wantLen || gotOk != tc.wantOk {
			t.Errorf("instructionLength(%q) = %d, %v; want %d, %v", tc.mnemonic, gotLen, gotOk, tc.wantLen, tc.wantOk)
		}
	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		line    string
		want    parsedLine
		wantErr bool
	}{
		{
			"LDI R0, 5",
			parsedLine{lineNo: 1, mnemonic: "LDI", operands: []string{"R0", "5"}},
			false,
		},
		{
			"  MOV R0, R1  ; comment",
			parsedLine{lineNo: 1, mnemonic: "MOV", operands: []string{"R0", "R1"}},
			false,
		},
		{
			"START: NOP",
			parsedLine{lineNo: 1, labels: []string{"START"}, mnemonic: "NOP", operands: nil},
			false,
		},
		{
			"LABEL1: LABEL2: HLT",
			parsedLine{lineNo: 1, labels: []string{"LABEL1", "LABEL2"}, mnemonic: "HLT", operands: nil},
			false,
		},
		{
			".ORG 0x100",
			parsedLine{lineNo: 1, mnemonic: ".ORG", operands: []string{"0x100"}},
			false,
		},
		{
			".STRING \"hello\"",
			parsedLine{lineNo: 1, mnemonic: ".STRING", operands: []string{"hello"}},
			false,
		},
		{
			".PSTRING \"hi\"",
			parsedLine{lineNo: 1, mnemonic: ".PSTRING", operands: []string{"hi"}},
			false,
		},
		// Invalid cases
		{
			"1LABEL: NOP",
			parsedLine{lineNo: 1},
			true,
		},
		{
			".STRING \"unterminated",
			parsedLine{lineNo: 1},
			true,
		},
		{
			".STRING missing_quote",
			parsedLine{lineNo: 1},
			true,
		},
	}

	for _, tc := range tests {
		got, err := parseLine(tc.line, 1)
		if (err != nil) != tc.wantErr {
			t.Errorf("parseLine(%q) error = %v, wantErr %v", tc.line, err, tc.wantErr)
			continue
		}
		if !tc.wantErr {
			if got.lineNo != tc.want.lineNo {
				t.Errorf("parseLine(%q) lineNo = %d, want %d", tc.line, got.lineNo, tc.want.lineNo)
			}
			if got.mnemonic != tc.want.mnemonic {
				t.Errorf("parseLine(%q) mnemonic = %q, want %q", tc.line, got.mnemonic, tc.want.mnemonic)
			}
			if !reflect.DeepEqual(got.labels, tc.want.labels) && !(len(got.labels) == 0 && len(tc.want.labels) == 0) {
				t.Errorf("parseLine(%q) labels = %v, want %v", tc.line, got.labels, tc.want.labels)
			}
			if !reflect.DeepEqual(got.operands, tc.want.operands) && !(len(got.operands) == 0 && len(tc.want.operands) == 0) {
				t.Errorf("parseLine(%q) operands = %v, want %v", tc.line, got.operands, tc.want.operands)
			}
		}
	}
}

func TestAssemble(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		want    []byte
		wantErr bool
	}{
		{
			"Basic Instructions",
			`
			LDI R0, 10
			ADD R0, R1
			HLT
			`,
			encodeWords(
				cpu.EncodeInstruction(cpu.OpLDI, cpu.RegA, 0, 0), 10,
				cpu.EncodeInstruction(cpu.OpADD, cpu.RegA, cpu.RegB, 0),
				cpu.EncodeInstruction(cpu.OpHLT, 0, 0, 0),
			),
			false,
		},
		{
			"Labels and Jumps",
			// LDI R0, 5  -> 4 bytes (addr 0-3)
			// LOOP:       -> addr 4
			// SUB R0, R1  -> 2 bytes (addr 4-5)
			// JNZ LOOP    -> 4 bytes (addr 6-9), target = 4
			// HLT         -> 2 bytes (addr 10-11)
			`
			LDI R0, 5
			LOOP:
			SUB R0, R1
			JNZ LOOP
			HLT
			`,
			encodeWords(
				cpu.EncodeInstruction(cpu.OpLDI, cpu.RegA, 0, 0), 5,
				cpu.EncodeInstruction(cpu.OpSUB, cpu.RegA, cpu.RegB, 0),
				cpu.EncodeInstruction(cpu.OpJNZ, 0, 0, 0), 4, // LOOP is at byte addr 4
				cpu.EncodeInstruction(cpu.OpHLT, 0, 0, 0),
			),
			false,
		},
		{
			".ORG",
			`
			.ORG 0x0004
			HLT
			`,
			// 4 bytes padding + 2 bytes HLT
			append([]byte{0, 0, 0, 0}, encodeWords(cpu.EncodeInstruction(cpu.OpHLT, 0, 0, 0))...),
			false,
		},
		{
			".STRING",
			`
			.STRING "AB"
			`,
			// 'A'=0x41, 'B'=0x42, null=0x00 -> 3 bytes
			[]byte{0x41, 0x42, 0x00},
			false,
		},
		{
			"Comments",
			`
			; Comment
			LDI R0, 1 // Comment
			`,
			encodeWords(cpu.EncodeInstruction(cpu.OpLDI, cpu.RegA, 0, 0), 1),
			false,
		},
		{
			"Immediate with hex",
			`
			LDI R0, 0x10
			`,
			encodeWords(cpu.EncodeInstruction(cpu.OpLDI, cpu.RegA, 0, 0), 0x10),
			false,
		},
		// Errors
		{
			"Unknown Instruction",
			`FOOBAR R0`,
			nil,
			true,
		},
		{
			"Duplicate Label",
			`
			L: HLT
			L: NOP
			`,
			nil,
			true,
		},
		{
			"Invalid Register",
			`ADD R0, R9`,
			nil,
			true,
		},
		{
			"Invalid Operand Count",
			`ADD R0`,
			nil,
			true,
		},
		{
			"Undefined Label",
			`JMP NOWHERE`,
			nil,
			true,
		},
		{
			".ORG Backward",
			`
			NOP
			.ORG 0
			`,
			nil,
			true,
		},
		{
			"Registers R2 R3",
			`
			MOV R2, R3
			`,
			encodeWords(cpu.EncodeInstruction(cpu.OpMOV, cpu.RegC, cpu.RegD, 0)),
			false,
		},
		{
			"FILL Instruction",
			`
			FILL R1, R3, R0
			`,
			encodeWords(cpu.EncodeInstruction(cpu.OpFILL, cpu.RegB, cpu.RegD, cpu.RegA)),
			false,
		},
		{
			"LDSP STSP",
			`
			LDSP R0
			STSP R2
			`,
			encodeWords(
				cpu.EncodeInstruction(cpu.OpLDSP, cpu.RegA, 0, 0),
				cpu.EncodeInstruction(cpu.OpSTSP, cpu.RegC, 0, 0),
			),
			false,
		},
		{
			"COPY Instruction",
			`
			COPY R0, R1, R2
			`,
			encodeWords(cpu.EncodeInstruction(cpu.OpCOPY, cpu.RegA, cpu.RegB, cpu.RegC)),
			false,
		},
		{
			".PSTRING ABC",
			`
			.PSTRING "ABC"
			`,
			// 'A'=0x41, 'B'=0x42 -> LE word 0x4241 -> bytes [0x41, 0x42]
			// 'C'=0x43 -> LE word 0x0043 -> bytes [0x43, 0x00]
			// null word -> [0x00, 0x00]
			[]byte{0x41, 0x42, 0x43, 0x00, 0x00, 0x00},
			false,
		},
		{
			".PSTRING even length",
			`
			.PSTRING "ABCD"
			`,
			// 'A'|'B'<<8 -> [0x41,0x42], 'C'|'D'<<8 -> [0x43,0x44], null -> [0x00,0x00]
			[]byte{0x41, 0x42, 0x43, 0x44, 0x00, 0x00},
			false,
		},
		{
			".PSTRING empty",
			`
			.PSTRING ""
			`,
			// null word -> [0x00, 0x00]
			[]byte{0x00, 0x00},
			false,
		},
		{
			"SHL two-register",
			`
			SHL R0, R1
			`,
			encodeWords(cpu.EncodeInstruction(cpu.OpSHL, cpu.RegA, cpu.RegB, 0)),
			false,
		},
		{
			"SHR two-register",
			`
			SHR R2, R3
			`,
			encodeWords(cpu.EncodeInstruction(cpu.OpSHR, cpu.RegC, cpu.RegD, 0)),
			false,
		},
		{
			".STRING escape sequences",
			`
			.STRING "A\nB"
			`,
			// 'A'=65, '\n'=10, 'B'=66, null=0 -> 4 bytes
			[]byte{65, 10, 66, 0},
			false,
		},
		{
			".STRING empty",
			`
			.STRING ""
			`,
			[]byte{0},
			false,
		},
		{
			"Label only line",
			`
			START:
			LDI R0, 1
			`,
			encodeWords(cpu.EncodeInstruction(cpu.OpLDI, cpu.RegA, 0, 0), 1),
			false,
		},
		{
			"Program Too Large",
			`
			.ORG 0xFFFF
			LDI R0, 1
			`,
			nil,
			true,
		},
		{
			"LDB STB instructions",
			`
			LDB R0, R1
			STB R2, R3
			`,
			encodeWords(
				cpu.EncodeInstruction(cpu.OpLDB, cpu.RegA, cpu.RegB, 0),
				cpu.EncodeInstruction(cpu.OpSTB, cpu.RegC, cpu.RegD, 0),
			),
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := Assemble(tc.code)
			if (err != nil) != tc.wantErr {
				t.Errorf("Assemble() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Assemble() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStripComments(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"LDI R0, 1", "LDI R0, 1"},
		{"LDI R0, 1 ; comment", "LDI R0, 1 "},
		{"LDI R0, 1 // comment", "LDI R0, 1 "},
		{"// comment", ""},
		{"; comment", ""},
		{"LDI R0, 1 ; first // second", "LDI R0, 1 "},
	}
	for _, tc := range tests {
		if got := stripComments(tc.input); got != tc.want {
			t.Errorf("stripComments(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
