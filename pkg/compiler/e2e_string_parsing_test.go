package compiler

import (
	"bytes"
	"testing"
	"gocpu/pkg/asm"
	"gocpu/pkg/cpu"
)

func TestEndToEnd_StringParsing(t *testing.T) {
	source := `
#include "../../lib/stdio.c"
#include "../../lib/video.c"

int main() {
    int* ctrl = 0xFF05;
    *ctrl = 7; // Text + Graphics + Buffered
    
    print("GoCPU v2.0 Online"); 
    
    video_flip(0);
    return 0;
}
`

	// 1. Compile C -> ASM
	processed, err := Preprocess(source, ".")
	if err != nil {
		t.Fatalf("Preprocess failed: %v", err)
	}

	tokens, err := Lex(processed)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}
	stmts, err := Parse(tokens, processed)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	syms := NewSymbolTable()
	assembly, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 2. Assemble ASM -> Machine Code
	machineCode, _, err := asm.Assemble(assembly)
	if err != nil {
		t.Fatalf("Assemble failed: %v\nAssembly:\n%s", err, assembly)
	}

	// 3. Run CPU
	vm := cpu.NewCPU()
	if len(machineCode) > len(vm.Memory) {
		t.Fatalf("Program too large")
	}
	copy(vm.Memory[:], machineCode)

	var output bytes.Buffer
	vm.Output = &output

	// Run until halted or max steps
	for i := 0; i < 5000; i++ {
		if vm.Halted {
			break
		}
		vm.Step()
	}

	if !vm.Halted {
		t.Errorf("VM did not halt within 5000 steps")
	}

	// 4. Verify Output
	got := output.String()
	expected := "GoCPU v2.0 Online"
	if got != expected {
		t.Errorf("Expected output %q, got %q", expected, got)
	}
}
