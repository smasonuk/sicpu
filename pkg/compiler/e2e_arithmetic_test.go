package compiler

import (
	"fmt"
	"testing"
)

func TestArithmetic_E2E(t *testing.T) {
	tests := []struct {
		expr     string
		expected int
	}{
		{"6 * 7", 42},
		{"100 / 10", 10},
		{"10 % 3", 1},
	}
	for _, tt := range tests {
		src := fmt.Sprintf("int main() { return %s; }", tt.expr)
		regs := runCode(t, src)
		if int(regs[0]) != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.expr, tt.expected, regs[0])
		}
	}
}

func TestBitwise_E2E(t *testing.T) {
	tests := []struct {
		expr     string
		expected int
	}{
		{"0xFF & 0x0F", 15},
		{"0xF0 | 0x0F", 255},
		{"~0", 0xFFFF},
	}
	for _, tt := range tests {
		src := fmt.Sprintf("int main() { return %s; }", tt.expr)
		regs := runCode(t, src)
		// Compare as uint16 to handle ~0 correctly
		if regs[0] != uint16(tt.expected) {
			t.Errorf("%s: expected 0x%X, got 0x%X", tt.expr, uint16(tt.expected), regs[0])
		}
	}
}

func TestShifts_E2E(t *testing.T) {
	tests := []struct {
		expr     string
		expected int
	}{
		{"1 << 4", 16},
		{"256 >> 4", 16},
	}
	for _, tt := range tests {
		src := fmt.Sprintf("int main() { return %s; }", tt.expr)
		regs := runCode(t, src)
		if int(regs[0]) != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.expr, tt.expected, regs[0])
		}
	}
}

func TestComparison_E2E(t *testing.T) {
	tests := []struct {
		expr     string
		expected int
	}{
		{"5 < 10", 1},
		{"10 < 5", 0},
		{"5 > 3", 1},
		{"1 != 2", 1},
		{"1 != 1", 0},
	}
	for _, tt := range tests {
		src := fmt.Sprintf("int main() { return %s; }", tt.expr)
		regs := runCode(t, src)
		if int(regs[0]) != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.expr, tt.expected, regs[0])
		}
	}
}

func TestControlFlow_E2E(t *testing.T) {
	src := `
	int main() {
		int s = 0;
		for (int i = 0; i < 5; i++) {
			s += i;
		}
		return s;
	}
	`
	regs := runCode(t, src)
	if regs[0] != 10 {
		t.Errorf("For loop accumulation: expected 10, got %d", regs[0])
	}
}

func TestCompoundAssignment_E2E(t *testing.T) {
	src := `
	int main() {
		int x = 10;
		x += 5; // 15
		x -= 3; // 12
		x *= 2; // 24
		x /= 4; // 6
		return x;
	}
	`
	regs := runCode(t, src)
	if regs[0] != 6 {
		t.Errorf("Compound assignment: expected 6, got %d", regs[0])
	}
}

func TestPostfix_E2E(t *testing.T) {
	src := `
	int main() {
		int x = 5;
		x++; // 6
		x++; // 7
		x--; // 6
		return x;
	}
	`
	regs := runCode(t, src)
	if regs[0] != 6 {
		t.Errorf("Postfix operators: expected 6, got %d", regs[0])
	}
}
