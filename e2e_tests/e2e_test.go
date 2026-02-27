package main

import (
	"testing"

	"gocpu/pkg/asm"
	"gocpu/pkg/compiler"
	"gocpu/pkg/cpu"
)

func TestCompilerAndCPU(t *testing.T) {
	// 1. Define C source
	source := `
int fib(int n) {
    if (n == 0) { return 0; }
    if (n == 1) { return 1; }
    return fib(n - 1) + fib(n - 2);
}

int main() {
    int limit = 6;
    int result = fib(limit);
    int* out = 0x3000;
    *out = result;
    return result;
}
`

	// 2. Lex and Parse
	tokens, err := compiler.Lex(source)
	if err != nil {
		t.Fatalf("Lexing failed: %v", err)
	}

	ast, err := compiler.Parse(tokens, source)
	if err != nil {
		t.Fatalf("Parsing failed: %v", err)
	}

	// 3. Generate Assembly
	syms := compiler.NewSymbolTable()
	assembly, err := compiler.Generate(ast, syms)
	if err != nil {
		t.Fatalf("Code generation failed: %v", err)
	}

	t.Logf("Generated Assembly:\n%s", assembly)

	// 4. Assemble
	machineCode, _, err := asm.Assemble(assembly)
	if err != nil {
		t.Fatalf("Assembly failed: %v", err)
	}

	// 5. Instantiate CPU
	vm := cpu.NewCPU()

	// 6. Load Code
	if len(machineCode) > len(vm.Memory) {
		t.Fatalf("Program too large for memory")
	}
	copy(vm.Memory[:], machineCode)

	// 7. Run
	// Run() runs until Halted is true.
	vm.Run()

	// 8. Assertions

	// Verify R0 equals 8 (the 6th Fibonacci number: 0, 1, 1, 2, 3, 5, 8)
	if vm.Regs[cpu.RegA] != 8 {
		t.Errorf("Expected R0 to be 8, got %d", vm.Regs[cpu.RegA])
	}

	// Verify Memory[0x3000] equals 8 (word read)
	if vm.Read16(0x3000) != 8 {
		t.Errorf("Expected Memory[0x3000] to be 8, got %d", vm.Read16(0x3000))
	}

	// Verify SP equals 0xFFFE (stack fully unwound; initial SP is 0xFFFE)
	if vm.SP != 0xFFFE {
		t.Errorf("Expected SP to be 0xFFFE, got 0x%04X", vm.SP)
	}
}
