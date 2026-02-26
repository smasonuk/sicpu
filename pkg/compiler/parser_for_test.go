package compiler

import (
	"reflect"
	"testing"
)

func TestParse_ForLoop(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Stmt
	}{
		{
			name:  "For loop with declaration",
			input: "for (int i = 0; i < 10; i++) { }",
			expected: []Stmt{
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
							Left: &VarRef{Name: "i"},
							Op:   PLUS_PLUS,
						},
					},
					Body: &BlockStmt{},
				},
			},
		},
		{
			name:  "For loop with assignment init",
			input: "for (i = 0; i < 10; i++) { }",
			expected: []Stmt{
				&ForStmt{
					Init: &Assignment{
						Left: &VarRef{Name: "i"},
						Op:   ASSIGN,
						Value: &Literal{Value: 0},
					},
					Cond: &BinaryExpr{
						Op:    LESS,
						Left:  &VarRef{Name: "i"},
						Right: &Literal{Value: 10},
					},
					Post: &ExprStmt{
						Expr: &PostfixExpr{
							Left: &VarRef{Name: "i"},
							Op:   PLUS_PLUS,
						},
					},
					Body: &BlockStmt{},
				},
			},
		},
		{
			name:  "For loop with empty parts",
			input: "for (;;) { }",
			expected: []Stmt{
				&ForStmt{
					Init: nil,
					Cond: nil,
					Post: nil,
					Body: &BlockStmt{},
				},
			},
		},
		{
			name:  "For loop with compound assignment post",
			input: "for (i = 0; i < 10; i += 2) { }",
			expected: []Stmt{
				&ForStmt{
					Init: &Assignment{
						Left: &VarRef{Name: "i"},
						Op:   ASSIGN,
						Value: &Literal{Value: 0},
					},
					Cond: &BinaryExpr{
						Op:    LESS,
						Left:  &VarRef{Name: "i"},
						Right: &Literal{Value: 10},
					},
					Post: &Assignment{
						Left:  &VarRef{Name: "i"},
						Op:    PLUS_ASSIGN,
						Value: &Literal{Value: 2},
					},
					Body: &BlockStmt{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
            // Need to wrap in function because Parse enforces top-level declarations
            input := "int main() { " + tt.input + " }"

			tokens, err := Lex(input)
			if err != nil {
				t.Fatalf("Lex failed: %v", err)
			}
			stmts, err := Parse(tokens, input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

            // Unwrap function body
            funcDecl := stmts[0].(*FunctionDecl)
            bodyStmts := funcDecl.Body.(*BlockStmt).Stmts
            if len(bodyStmts) != len(tt.expected) {
                 t.Fatalf("Expected %d statements, got %d", len(tt.expected), len(bodyStmts))
            }

			if !reflect.DeepEqual(bodyStmts, tt.expected) {
				t.Errorf("Parse mismatch:\nGot:      %v\nExpected: %v", bodyStmts, tt.expected)
			}
		})
	}
}
