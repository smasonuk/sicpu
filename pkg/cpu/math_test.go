package cpu

import (
	"testing"
)

func TestFixedPointMath(t *testing.T) {
	tests := []struct {
		name     string
		inputA   uint16
		inputB   uint16
		op       uint16 // 0=Mul, 1=Div
		expected uint16
	}{
		{"Mul_Simple", 0x0100, 0x0100, 0, 0x0100},   // 1.0 * 1.0 = 1.0
		{"Mul_Fraction", 0x0080, 0x0080, 0, 0x0040}, // 0.5 * 0.5 = 0.25
		{"Mul_Negative", 0xFF00, 0x0200, 0, 0xFE00}, // -1.0 * 2.0 = -2.0
		{"Div_Simple", 0x0400, 0x0200, 1, 0x0200},   // 4.0 / 2.0 = 2.0
		{"Div_Precise", 0x0100, 0x0300, 1, 0x0055},  // 1.0 / 3.0 = 0.333...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := NewCPU()
			vm.handleMMIOWrite16(0xFF23, tt.op)     // Set Op
			vm.handleMMIOWrite16(0xFF20, tt.inputA) // Write A
			vm.handleMMIOWrite16(0xFF21, tt.inputB) // Write B (Triggers)

			res := vm.Read16(0xFF22)
			if res != tt.expected {
				t.Errorf("Expected 0x%04X, got 0x%04X", tt.expected, res)
			}
		})
	}
}

func TestDivisionByZero(t *testing.T) {
	vm := NewCPU()
	vm.handleMMIOWrite16(0xFF23, 1)      // Set Op to Div
	vm.handleMMIOWrite16(0xFF20, 0x0100) // Write A
	vm.handleMMIOWrite16(0xFF21, 0x0000) // Write B (Zero)

	res := vm.Read16(0xFF22)
	if res != 0xFFFF {
		t.Errorf("Expected 0xFFFF (Error), got 0x%04X", res)
	}
}
