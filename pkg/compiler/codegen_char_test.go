package compiler

import (
	"strings"
	"testing"
)

func TestCodeGen_Char(t *testing.T) {
	input := `
	char g = 42;
	char *g_ptr = &g;

	void main() {
		char c = 10;
		char *p = &c;
		*p = 20;
		g = *p;
	}
	`
	tokens, err := Lex(input)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}
	stmts, err := Parse(tokens, input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	syms := NewSymbolTable()
	asm, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify key instructions
	if !strings.Contains(asm, "STB [R1], R0") {
		t.Error("Expected STB instruction for char assignment")
	}
	// Note: loading a char uses LDB
	if !strings.Contains(asm, "LDB R0, [R1]") {
		t.Error("Expected LDB instruction for char load")
	}

	// Check global initialization
	if !strings.Contains(asm, ".WORD 42") {
		t.Error("Expected global char to be initialized with .WORD")
	}
}
