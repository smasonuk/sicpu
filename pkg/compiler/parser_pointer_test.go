package compiler

import (
	"testing"
)

func TestParser_MultiLevelPointers(t *testing.T) {
	input := `
	int **pp;
	char ***ppp;
	struct Node **node_pp;
	void f(int **a) {}
	`
	tokens, err := Lex(input)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}

	stmts, err := Parse(tokens, input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(stmts) != 4 {
		t.Fatalf("Expected 4 statements, got %d", len(stmts))
	}

	// 1. int **pp;
	v1, ok := stmts[0].(*VariableDecl)
	if !ok {
		t.Errorf("Stmt 0 not VariableDecl")
	} else {
		if v1.PointerLevel != 2 {
			t.Errorf("Stmt 0 expected PointerLevel=2, got %d", v1.PointerLevel)
		}
	}

	// 2. char ***ppp;
	v2, ok := stmts[1].(*VariableDecl)
	if !ok {
		t.Errorf("Stmt 1 not VariableDecl")
	} else {
		if !v2.IsChar {
			t.Errorf("Stmt 1 expected IsChar=true")
		}
		if v2.PointerLevel != 3 {
			t.Errorf("Stmt 1 expected PointerLevel=3, got %d", v2.PointerLevel)
		}
	}

	// 3. struct Node **node_pp;
	v3, ok := stmts[2].(*VariableDecl)
	if !ok {
		t.Errorf("Stmt 2 not VariableDecl")
	} else {
		// Pointers to structs are treated as scalars (IsStruct=false, PointerLevel > 0)
		if v3.PointerLevel != 2 {
			t.Errorf("Stmt 2 expected PointerLevel=2, got %d", v3.PointerLevel)
		}
	}

	// 4. void f(int **a) {}
	f4, ok := stmts[3].(*FunctionDecl)
	if !ok {
		t.Errorf("Stmt 3 not FunctionDecl")
	} else {
		if len(f4.Params) != 1 {
			t.Errorf("Stmt 3 expected 1 param")
		} else {
			if f4.Params[0].PointerLevel != 2 {
				t.Errorf("Param 0 expected PointerLevel=2, got %d", f4.Params[0].PointerLevel)
			}
		}
	}
}
