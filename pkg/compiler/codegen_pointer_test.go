package compiler

import (
	"strings"
	"testing"
)

func TestCodeGen_MultiLevelPointers(t *testing.T) {
	input := `
	void main() {
		int x = 42;
		int *p = &x;
		int **pp = &p;
		int y = **pp;
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

	// For int **pp = &p;
	// 1. Get address of p -> R1
	// 2. Store R1 into pp
	
	// For int y = **pp;
	// 1. Load pp -> R0 (address of p)
	// 2. Dereference -> Load [R0] -> R0 (value of p, which is address of x)
	// 3. Dereference -> Load [R0] -> R0 (value of x, which is 42)
	
	// We expect two LD instructions for the double dereference
	// But `genExpr` handles `**pp` as `UnaryExpr(STAR, UnaryExpr(STAR, VarRef(pp)))`
	// Inner STAR: calls genExpr(STAR, VarRef(pp))
	//   genExpr(VarRef(pp)) -> Loads pp value into R0 (address of p)
	//   dereference -> MOV R1, R0; LD R0, [R1] -> R0 is value of p (address of x)
	// Outer STAR:
	//   calls inner (R0 is address of x)
	//   dereference -> MOV R1, R0; LD R0, [R1] -> R0 is value of x (42)
	
	// So we should see a sequence of loads.
	if !strings.Contains(asm, "LD  R0, [R1]") {
		t.Error("Expected LD instructions")
	}
}
