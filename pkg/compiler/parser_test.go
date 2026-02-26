package compiler

import (
	"reflect"
	"testing"
)

// TestParse verifies that Parse produces the correct AST for valid inputs.
func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Stmt
	}{
		{
			name:  "Variable Declaration",
			input: "int x = 10;",
			expected: []Stmt{
				&VariableDecl{Name: "x", Init: &Literal{Value: 10}},
			},
		},
		{
			name:  "Pointer Declaration",
			input: "int* p = &x;",
			expected: []Stmt{
				&VariableDecl{Name: "p", Init: &UnaryExpr{Op: AND, Right: &VarRef{Name: "x"}}, IsPointer: true},
			},
		},
		{
			name:  "Assignment",
			input: "int main() { x = 20; }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&Assignment{Op: ASSIGN, Left: &VarRef{Name: "x"}, Value: &Literal{Value: 20}},
				}}},
			},
		},
		{
			name:  "Pointer Assignment",
			input: "int main() { *p = 30; }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&Assignment{
						Op:    ASSIGN,
						Left:  &UnaryExpr{Op: STAR, Right: &VarRef{Name: "p"}},
						Value: &Literal{Value: 30},
					},
				}}},
			},
		},
		{
			name:  "Function Call",
			input: "int main() { foo(1, x); }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&ExprStmt{
						Expr: &FunctionCall{
							Name: "foo",
							Args: []Expr{
								&Literal{Value: 1},
								&VarRef{Name: "x"},
							},
						},
					},
				}}},
			},
		},
		{
			name:  "If Statement",
			input: "int main() { if (x == 1) { x = 2; } }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&IfStmt{
						Condition: &BinaryExpr{
							Op:    EQUALS,
							Left:  &VarRef{Name: "x"},
							Right: &Literal{Value: 1},
						},
						Body: &BlockStmt{
							Stmts: []Stmt{
								&Assignment{Op: ASSIGN, Left: &VarRef{Name: "x"}, Value: &Literal{Value: 2}},
							},
						},
					},
				}}},
			},
		},
		{
			name:  "If-Else Statement",
			input: "int main() { if (x == 1) { x = 2; } else { x = 3; } }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&IfStmt{
						Condition: &BinaryExpr{
							Op:    EQUALS,
							Left:  &VarRef{Name: "x"},
							Right: &Literal{Value: 1},
						},
						Body: &BlockStmt{
							Stmts: []Stmt{
								&Assignment{Op: ASSIGN, Left: &VarRef{Name: "x"}, Value: &Literal{Value: 2}},
							},
						},
						ElseBody: &BlockStmt{
							Stmts: []Stmt{
								&Assignment{Op: ASSIGN, Left: &VarRef{Name: "x"}, Value: &Literal{Value: 3}},
							},
						},
					},
				}}},
			},
		},
		{
			name:  "While Loop",
			input: "int main() { while (x == 0) { x = 1; } }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&WhileStmt{
						Condition: &BinaryExpr{
							Op:    EQUALS,
							Left:  &VarRef{Name: "x"},
							Right: &Literal{Value: 0},
						},
						Body: &BlockStmt{
							Stmts: []Stmt{
								&Assignment{Op: ASSIGN, Left: &VarRef{Name: "x"}, Value: &Literal{Value: 1}},
							},
						},
					},
				}}},
			},
		},
		{
			name:  "Function Declaration",
			input: "int main() { return 0; }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int",
					Name:   "main",
					Params: nil,
					Body: &BlockStmt{
						Stmts: []Stmt{
							&ReturnStmt{Expr: &Literal{Value: 0}},
						},
					},
				},
			},
		},
		{
			name:  "Function Declaration with Params",
			input: "int add(int a, int b) { return a + b; }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int",
					Name:   "add",
					Params: []VariableDecl{{Name: "a"}, {Name: "b"}},
					Body: &BlockStmt{
						Stmts: []Stmt{
							&ReturnStmt{
								Expr: &BinaryExpr{
									Op:    PLUS,
									Left:  &VarRef{Name: "a"},
									Right: &VarRef{Name: "b"},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "Complex Expression",
			input: "int main() { x = 1 + 2 - 3; }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&Assignment{
						Op:   ASSIGN,
						Left: &VarRef{Name: "x"},
						Value: &BinaryExpr{
							Op: MINUS,
							Left: &BinaryExpr{
								Op:    PLUS,
								Left:  &Literal{Value: 1},
								Right: &Literal{Value: 2},
							},
							Right: &Literal{Value: 3},
						},
					},
				}}},
			},
		},
		{
			name:  "Complex Expression Precedence",
			input: "int main() { x = *p + 5; }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&Assignment{
						Op:   ASSIGN,
						Left: &VarRef{Name: "x"},
						Value: &BinaryExpr{
							Op: PLUS,
							Left: &UnaryExpr{
								Op:    STAR,
								Right: &VarRef{Name: "p"},
							},
							Right: &Literal{Value: 5},
						},
					},
				}}},
			},
		},
		{
			name:  "Operator Precedence: Mul vs Add",
			input: "int main() { x = 1 + 2 * 3; }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&Assignment{
						Op:   ASSIGN,
						Left: &VarRef{Name: "x"},
						Value: &BinaryExpr{
							Op:   PLUS,
							Left: &Literal{Value: 1},
							Right: &BinaryExpr{
								Op:    STAR,
								Left:  &Literal{Value: 2},
								Right: &Literal{Value: 3},
							},
						},
					},
				}}},
			},
		},
		{
			name:  "Operator Precedence: Add vs Relational",
			input: "int main() { x = 1 < 2 + 3; }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&Assignment{
						Op:   ASSIGN,
						Left: &VarRef{Name: "x"},
						Value: &BinaryExpr{
							Op:   LESS,
							Left: &Literal{Value: 1},
							Right: &BinaryExpr{
								Op:    PLUS,
								Left:  &Literal{Value: 2},
								Right: &Literal{Value: 3},
							},
						},
					},
				}}},
			},
		},
		{
			name:  "Operator Precedence: Relational vs Equality",
			input: "int main() { x = 1 == 2 < 3; }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&Assignment{
						Op:   ASSIGN,
						Left: &VarRef{Name: "x"},
						Value: &BinaryExpr{
							Op:   EQUALS,
							Left: &Literal{Value: 1},
							Right: &BinaryExpr{
								Op:    LESS,
								Left:  &Literal{Value: 2},
								Right: &Literal{Value: 3},
							},
						},
					},
				}}},
			},
		},
		{
			name:  "Else If Chaining",
			input: "int main() { if (x == 1) { } else if (x == 2) { } else { } }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&IfStmt{
						Condition: &BinaryExpr{Op: EQUALS, Left: &VarRef{Name: "x"}, Right: &Literal{Value: 1}},
						Body:      &BlockStmt{Stmts: nil},
						ElseBody: &IfStmt{
							Condition: &BinaryExpr{Op: EQUALS, Left: &VarRef{Name: "x"}, Right: &Literal{Value: 2}},
							Body:      &BlockStmt{Stmts: nil},
							ElseBody:  &BlockStmt{Stmts: nil},
						},
					},
				}}},
			},
		},
		{
			name:  "Deeply Nested Expression",
			input: "int main() { x = (((1 + 2))); }",
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&Assignment{
						Op:   ASSIGN,
						Left: &VarRef{Name: "x"},
						Value: &BinaryExpr{
							Op:    PLUS,
							Left:  &Literal{Value: 1},
							Right: &Literal{Value: 2},
						},
					},
				}}},
			},
		},
		{
			name:  "Asm Statement",
			input: `int main() { asm("WFI"); }`,
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&AsmStmt{Instruction: "WFI"},
				}}},
			},
		},
		{
			name:  "Switch Statement",
			input: `int main() { switch (x) { case 1: x=2; default: x=3; } }`,
			expected: []Stmt{
				&FunctionDecl{ReturnType: "int", Name: "main", Params: nil, Body: &BlockStmt{Stmts: []Stmt{
					&SwitchStmt{
						Target: &VarRef{Name: "x"},
						Cases: []CaseClause{
							{
								Value: &Literal{Value: 1},
								Body: []Stmt{
									&Assignment{Op: ASSIGN, Left: &VarRef{Name: "x"}, Value: &Literal{Value: 2}},
								},
							},
						},
						Default: []Stmt{
							&Assignment{Op: ASSIGN, Left: &VarRef{Name: "x"}, Value: &Literal{Value: 3}},
						},
					},
				}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("Lex failed: %v", err)
			}

			stmts, err := Parse(tokens, tt.input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if !reflect.DeepEqual(stmts, tt.expected) {
				t.Errorf("Parse mismatch:\nGot:      %v\nExpected: %v", stmts, tt.expected)
			}
		})
	}
}

func TestParse_BitwiseAndShift(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Stmt
	}{
		{
			name:  "Bitwise AND",
			input: "int x = a & b;",
			expected: []Stmt{
				&VariableDecl{Name: "x", Init: &BinaryExpr{Op: AND, Left: &VarRef{Name: "a"}, Right: &VarRef{Name: "b"}}},
			},
		},
		{
			name:  "Bitwise OR",
			input: "int x = a | b;",
			expected: []Stmt{
				&VariableDecl{Name: "x", Init: &BinaryExpr{Op: PIPE, Left: &VarRef{Name: "a"}, Right: &VarRef{Name: "b"}}},
			},
		},
		{
			name:  "Bitwise XOR",
			input: "int x = a ^ b;",
			expected: []Stmt{
				&VariableDecl{Name: "x", Init: &BinaryExpr{Op: CARET, Left: &VarRef{Name: "a"}, Right: &VarRef{Name: "b"}}},
			},
		},
		{
			name:  "Bitwise NOT",
			input: "int x = ~a;",
			expected: []Stmt{
				&VariableDecl{Name: "x", Init: &UnaryExpr{Op: TILDE, Right: &VarRef{Name: "a"}}},
			},
		},
		{
			name:  "Modulo",
			input: "int x = a % b;",
			expected: []Stmt{
				&VariableDecl{Name: "x", Init: &BinaryExpr{Op: PERCENT, Left: &VarRef{Name: "a"}, Right: &VarRef{Name: "b"}}},
			},
		},
		{
			name:  "Left Shift",
			input: "int x = a << 2;",
			expected: []Stmt{
				&VariableDecl{Name: "x", Init: &BinaryExpr{Op: SHL_OP, Left: &VarRef{Name: "a"}, Right: &Literal{Value: 2}}},
			},
		},
		{
			name:  "Right Shift",
			input: "int x = a >> 2;",
			expected: []Stmt{
				&VariableDecl{Name: "x", Init: &BinaryExpr{Op: SHR_OP, Left: &VarRef{Name: "a"}, Right: &Literal{Value: 2}}},
			},
		},
		{
			// | has lower precedence than &: a | (b & c)
			name:  "Precedence OR lower than AND",
			input: "int x = a | b & c;",
			expected: []Stmt{
				&VariableDecl{Name: "x", Init: &BinaryExpr{
					Op:   PIPE,
					Left: &VarRef{Name: "a"},
					Right: &BinaryExpr{Op: AND, Left: &VarRef{Name: "b"}, Right: &VarRef{Name: "c"}},
				}},
			},
		},
		{
			// & has lower precedence than ==: a & (b == c)
			name:  "Precedence AND lower than equality",
			input: "int x = a & b == c;",
			expected: []Stmt{
				&VariableDecl{Name: "x", Init: &BinaryExpr{
					Op:   AND,
					Left: &VarRef{Name: "a"},
					Right: &BinaryExpr{Op: EQUALS, Left: &VarRef{Name: "b"}, Right: &VarRef{Name: "c"}},
				}},
			},
		},
		{
			// address-of & in unary position, then binary &
			name:  "Address-of and bitwise AND mixed",
			input: "int x = &p & 0xFF;",
			expected: []Stmt{
				&VariableDecl{Name: "x", Init: &BinaryExpr{
					Op:    AND,
					Left:  &UnaryExpr{Op: AND, Right: &VarRef{Name: "p"}},
					Right: &Literal{Value: 0xFF},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("Lex failed: %v", err)
			}
			stmts, err := Parse(tokens, tt.input)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if !reflect.DeepEqual(stmts, tt.expected) {
				t.Errorf("Parse mismatch:\nGot:      %v\nExpected: %v", stmts, tt.expected)
			}
		})
	}
}

func TestParserErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Missing Semicolon", "int x = 10"},
		{"Invalid Variable Declaration", "int 10 = x;"},
		{"Mismatched Parentheses", "if (x == 1 { x = 2; }"},
		{"Mismatched Braces", "if (x == 1) { x = 2;"},
		{"Unexpected Token", "return;"}, // return expects an expression
		{"Invalid Factor", "x = +;"},
		{"Missing Pointer Name", "int* = 10;"},
		{"Malformed Array", "int arr[;"},
		{"Anonymous Struct", "struct {} ;"},
		{"Unnamed Parameter", "int foo(int) { }"},
		{"Trailing Comma in Params", "int foo(int a, ) { }"},
		{"For Loop Missing 3rd Clause", "for (int i = 0; i < 10) { }"},
		{"If Missing Parens", "if x == 1 { }"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				// If lexer fails, we can't test parser, but strictly speaking these inputs should lex fine.
				t.Fatalf("Lex failed unexpectedly: %v", err)
			}

			_, err = Parse(tokens, tt.input)
			if err == nil {
				t.Errorf("Expected parse error for input: %q, but got none", tt.input)
			}
		})
	}
}
