package compiler

import (
	"strings"
	"testing"
)

// assertContainsNew checks if the generated code contains the expected substring.
func assertContainsNew(t *testing.T, code, expected string) {
	t.Helper()
	if !strings.Contains(code, expected) {
		t.Errorf("Expected code to contain %q, but it didn't.\nCode:\n%s", expected, code)
	}
}

func TestGenerate_ForLoop(t *testing.T) {
	syms := NewSymbolTable()
	// for (int i = 0; i < 10; i++) { }
	stmts := []Stmt{
		&FunctionDecl{
			Name: "main",
				Params: []VariableDecl{},
			Body: &BlockStmt{
				Stmts: []Stmt{
					&ForStmt{
						Init: &VariableDecl{
							Name: "i",
							Init: &Literal{Value: 0},
						},
						Cond: &BinaryExpr{
							Op:    LESS,
							Left:  &VarRef{Name: "i"},
							Right: &Literal{Value: 10},
						},
						Post: &ExprStmt{
							Expr: &PostfixExpr{
								Op:   PLUS_PLUS,
								Left: &VarRef{Name: "i"},
							},
						},
						Body: &BlockStmt{},
					},
				},
			},
		},
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContainsNew(t, code, "LDI R0, 0") // init
	assertContainsNew(t, code, "LDI R0, 10") // cond
	assertContainsNew(t, code, "JN") // cond check
	assertContainsNew(t, code, "ADD R0, R3") // increment (i++)
}

func TestGenerate_CompoundAssignment(t *testing.T) {
	syms := NewSymbolTable()
	// int x = 10; x += 5;
	stmts := []Stmt{
		&FunctionDecl{
			Name: "main",
				Params: []VariableDecl{},
			Body: &BlockStmt{
				Stmts: []Stmt{
					&VariableDecl{
						Name: "x",
						Init: &Literal{Value: 10},
					},
					&Assignment{
						Left:  &VarRef{Name: "x"},
						Op:    PLUS_ASSIGN,
						Value: &Literal{Value: 5},
					},
				},
			},
		},
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContainsNew(t, code, "ADD R1, R0") // x += 5
}
