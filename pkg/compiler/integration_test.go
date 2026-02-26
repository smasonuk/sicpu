package compiler_test

import (
	"gocpu/pkg/asm"
	"gocpu/pkg/compiler"
	"strings"
	"testing"
)

func TestIntegration_DynamicGlobals(t *testing.T) {
	// A C program with various global variables
	src := `
	int a = 10;
	int b;
	byte c = 5;
	byte d;
	int arr[2] = {1, 2};

	int main() {
		return a + c + arr[0];
	}
	`

	// 1. Compile
	tokens, err := compiler.Lex(src)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}

	stmts, err := compiler.Parse(tokens, src)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	syms := compiler.NewSymbolTable()
	assembly, err := compiler.Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 2. Verify assembly structure
	// Check for labels
	if !strings.Contains(assembly, "a:") {
		t.Errorf("Assembly missing label 'a:'")
	}
	if !strings.Contains(assembly, "b:") {
		t.Errorf("Assembly missing label 'b:'")
	}
	if !strings.Contains(assembly, "c:") {
		t.Errorf("Assembly missing label 'c:'")
	}
	if !strings.Contains(assembly, "d:") {
		t.Errorf("Assembly missing label 'd:'")
	}
	if !strings.Contains(assembly, "arr:") {
		t.Errorf("Assembly missing label 'arr:'")
	}

	// Check for Data Segment at the end (simplistic check: labels should appear after HLT)
	lastHLT := strings.LastIndex(assembly, "HLT")
	labelA := strings.Index(assembly, "a:")
	if labelA < lastHLT {
		t.Errorf("Label 'a:' appears before last HLT (Data segment should be at end)")
	}

	// 3. Assemble
	// This ensures the generated labels are valid and the code layout is correct
	binary, _, err := asm.Assemble(assembly)
	if err != nil {
		t.Fatalf("Assembler failed: %v\nAssembly:\n%s", err, assembly)
	}

	// 4. Inspect Binary (optional, but good sanity check)
	// The binary size should be reasonable.
	// We can't easily check exact offsets without parsing the symbol map from assembler,
	// but success implies valid labels.
	if len(binary) == 0 {
		t.Errorf("Assembler produced empty binary")
	}
}
