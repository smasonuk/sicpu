package compiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestBinaryExpr(t *testing.T) {
	input := `
	int main() {
		int x = 1 + 2;
		int y = x - 1;
		int z = x == y;
		int w = (1 + 2) == 3;
		return z;
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
	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Basic check to see if we generated code for binary ops
	if !strings.Contains(code, "ADD") {
		t.Error("Generated code does not contain ADD instructions")
	}
	if !strings.Contains(code, "SUB") {
		t.Error("Generated code does not contain SUB instructions")
	}
	if !strings.Contains(code, "PUSH") || !strings.Contains(code, "POP") {
		t.Error("Generated code does not contain PUSH/POP instructions")
	}
	if !strings.Contains(code, "JZ") {
		t.Error("Generated code does not contain JZ instructions (for EQUALS)")
	}
	fmt.Println(code)
}
