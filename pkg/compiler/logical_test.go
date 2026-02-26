package compiler

import (
	"fmt"
	"testing"
	"gocpu/pkg/asm"
	"gocpu/pkg/cpu"
)

// helper to run code and return registers
func runCode(t *testing.T, source string) [4]uint16 {
	// Lex -> Parse -> Generate
	tokens, err := Lex(source)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}
	stmts, err := Parse(tokens, source)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	syms := NewSymbolTable()
	assembly, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Assemble
	machineCode, _, err := asm.Assemble(assembly)
	if err != nil {
		t.Fatalf("Assemble failed: %v\nAssembly:\n%s", err, assembly)
	}

	// Run
	vm := cpu.NewCPU()
	if len(machineCode) > len(vm.Memory) {
		t.Fatalf("Program too large")
	}
	copy(vm.Memory[:], machineCode)

	for i := 0; i < 10000; i++ {
		if vm.Halted {
			break
		}
		vm.Step()
	}

	return [4]uint16{vm.Regs[0], vm.Regs[1], vm.Regs[2], vm.Regs[3]}
}

func TestLogicalAnd(t *testing.T) {
	// Test truth table
	tests := []struct {
		a, b     int
		expected int
	}{
		{0, 0, 0},
		{0, 1, 0},
		{1, 0, 0},
		{1, 1, 1},
		{10, 20, 1}, // non-zero treated as true
	}

	for _, tt := range tests {
		src := fmt.Sprintf(`
		int main() {
			int a = %d;
			int b = %d;
			return a && b;
		}
		`, tt.a, tt.b)
		regs := runCode(t, src)
		if int(regs[0]) != tt.expected {
			t.Errorf("%d && %d: expected %d, got %d", tt.a, tt.b, tt.expected, regs[0])
		}
	}
}

func TestLogicalOr(t *testing.T) {
	// Test truth table
	tests := []struct {
		a, b     int
		expected int
	}{
		{0, 0, 0},
		{0, 1, 1},
		{1, 0, 1},
		{1, 1, 1},
		{10, 20, 1},
	}

	for _, tt := range tests {
		src := fmt.Sprintf(`
		int main() {
			int a = %d;
			int b = %d;
			return a || b;
		}
		`, tt.a, tt.b)
		regs := runCode(t, src)
		if int(regs[0]) != tt.expected {
			t.Errorf("%d || %d: expected %d, got %d", tt.a, tt.b, tt.expected, regs[0])
		}
	}
}

func TestNot(t *testing.T) {
	// Test truth table
	tests := []struct {
		a        int
		expected int
	}{
		{0, 1},
		{1, 0},
		{10, 0},
	}

	for _, tt := range tests {
		src := fmt.Sprintf(`
		int main() {
			int a = %d;
			return !a;
		}
		`, tt.a)
		regs := runCode(t, src)
		if int(regs[0]) != tt.expected {
			t.Errorf("!%d: expected %d, got %d", tt.a, tt.expected, regs[0])
		}
	}
}

func TestShortCircuitAnd(t *testing.T) {
	// int global = 0;
	// int side_effect() { global = 1; return 1; }
	// int main() {
	//   int result = 0 && side_effect();
	//   return global; // Should be 0
	// }
	src := `
	int global = 0;
	int side_effect() {
		global = 1;
		return 1;
	}
	int main() {
		int result = 0 && side_effect();
		return global;
	}
	`
	regs := runCode(t, src)
	if regs[0] != 0 {
		t.Errorf("Short-circuit AND failed: side effect executed (global=%d)", regs[0])
	}
}

func TestShortCircuitOr(t *testing.T) {
	// int global = 0;
	// int side_effect() { global = 1; return 1; }
	// int main() {
	//   int result = 1 || side_effect();
	//   return global; // Should be 0
	// }
	src := `
	int global = 0;
	int side_effect() {
		global = 1;
		return 1;
	}
	int main() {
		int result = 1 || side_effect();
		return global;
	}
	`
	regs := runCode(t, src)
	if regs[0] != 0 {
		t.Errorf("Short-circuit OR failed: side effect executed (global=%d)", regs[0])
	}
}

func TestLogicalPrecedence(t *testing.T) {
	// || < &&
	// 1 || 0 && 0 -> 1 || (0) -> 1
	// (1 || 0) && 0 -> 1 && 0 -> 0
	src1 := `
	int main() {
		return 1 || 0 && 0;
	}
	`
	regs1 := runCode(t, src1)
	if regs1[0] != 1 {
		t.Errorf("Precedence error: 1 || 0 && 0 should be 1, got %d", regs1[0])
	}

	src2 := `
	int main() {
		return (1 || 0) && 0;
	}
	`
	regs2 := runCode(t, src2)
	if regs2[0] != 0 {
		t.Errorf("Precedence error: (1 || 0) && 0 should be 0, got %d", regs2[0])
	}
}
