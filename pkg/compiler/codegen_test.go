package compiler

import (
	"strings"
	"testing"
)

// assertContains checks if the generated code contains the expected substring.
func assertContains(t *testing.T, code, expected string) {
	t.Helper()
	if !strings.Contains(code, expected) {
		t.Errorf("Expected code to contain %q, but it didn't.\nCode:\n%s", expected, code)
	}
}

func TestGenerate_GlobalVars(t *testing.T) {
	syms := NewSymbolTable()
	stmts := []Stmt{
		&VariableDecl{
			Name: "g1",
			Init: &Literal{Value: 100},
		},
		&FunctionDecl{
			Name: "main",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&Assignment{Op: ASSIGN,
						Left:  &VarRef{Name: "g1"},
						Value: &Literal{Value: 200},
					},
				},
			},
		},
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify global allocation via label
	// We expect "g1:" label and ".WORD 100" data
	assertContains(t, code, "g1:")
	assertContains(t, code, ".WORD 100")

	// Verify assignment uses label
	assertContains(t, code, "LDI R0, 200")
	// The assignment re-generates address load using label
	assertContains(t, code, "LDI R1, g1    ; &g1 (global)")

	// Ensure .ORG 0x4000 is NOT present
	if strings.Contains(code, ".ORG 0x4000") {
		t.Errorf("Expected code NOT to contain '.ORG 0x4000', but it did.")
	}
}

func TestGenerate_UninitializedGlobals(t *testing.T) {
	syms := NewSymbolTable()
	stmts := []Stmt{
		&VariableDecl{Name: "u1"}, // int is default
		&VariableDecl{Name: "u2", IsByte: true}, // byte
		&FunctionDecl{Name: "main", Body: &BlockStmt{}},
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// u1 (int, size 2) -> 1 word
	assertContains(t, code, "u1:")
	assertContains(t, code, ".WORD 0")

	// u2 (byte, size 1) -> (1+1)/2 = 1 word
	assertContains(t, code, "u2:")
	assertContains(t, code, ".WORD 0")
}

func TestGenerate_Functions(t *testing.T) {
	t.Run("regular function", func(t *testing.T) {
		syms := NewSymbolTable()
		stmts := []Stmt{
			&FunctionDecl{
				Name:   "myFunc",
				Params: []VariableDecl{{Name: "p1"}},
				Body: &BlockStmt{
					Stmts: []Stmt{
						&VariableDecl{
							Name: "local1",
							Init: &Literal{Value: 5},
						},
						&ReturnStmt{
							Expr: &VarRef{Name: "local1"},
						},
					},
				},
			},
			&FunctionDecl{
				Name: "main",
				Body: &BlockStmt{
					Stmts: []Stmt{
						&ExprStmt{
							Expr: &FunctionCall{
								Name: "myFunc",
								Args: []Expr{&Literal{Value: 0}},
							},
						},
					},
				},
			},
		}

		code, err := Generate(stmts, syms)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		assertContains(t, code, "myFunc:")
		assertContains(t, code, "PUSH R2")
		assertContains(t, code, "LDSP R2")

		// Local variable allocation (1 word = 2 bytes)
		// Now includes spilled param (2 bytes) = 4 bytes total
		assertContains(t, code, "LDI R1, 4")
		assertContains(t, code, "SUB R3, R1")
		assertContains(t, code, "STSP R3")

		// Variable init
		assertContains(t, code, "LDI R0, 5")
		assertContains(t, code, "MOV R1, R2") // FP
	})

	t.Run("isr function", func(t *testing.T) {
		syms := NewSymbolTable()
		stmts := []Stmt{
			&FunctionDecl{
				Name:   "isr",
				Params: []VariableDecl{},
				Body: &BlockStmt{
					Stmts: []Stmt{
						&ReturnStmt{Expr: &Literal{Value: 0}},
					},
				},
			},
			&FunctionDecl{
				Name:   "main",
				Params: []VariableDecl{},
				Body:   &BlockStmt{},
			},
		}

		code, err := Generate(stmts, syms)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		// Vector table assertions
		assertContains(t, code, ".ORG 0x0010")
		assertContains(t, code, "JMP isr")

		// ISR body ends with RETI, not RET
		assertContains(t, code, "isr:")
		assertContains(t, code, "RETI")
	})
}

func TestGenerate_Expressions(t *testing.T) {
	syms := NewSymbolTable()
	// int a = 10; int b = 20;
	// int x = a + b;
	// int y = x == 30;
	stmts := []Stmt{
		&VariableDecl{Name: "a", Init: &Literal{Value: 10}},
		&VariableDecl{Name: "b", Init: &Literal{Value: 20}},
		&VariableDecl{
			Name: "x",
			Init: &BinaryExpr{
				Op:    PLUS,
				Left:  &VarRef{Name: "a"},
				Right: &VarRef{Name: "b"},
			},
		},
		&VariableDecl{
			Name: "y",
			Init: &BinaryExpr{
				Op:    EQUALS,
				Left:  &VarRef{Name: "x"},
				Right: &Literal{Value: 30},
			},
		},
		&FunctionDecl{Name: "main", Body: &BlockStmt{}}, // Trigger __init
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "ADD R1, R0")
	assertContains(t, code, "SUB R1, R0") // For equality check
	assertContains(t, code, "JZ")         // For equality check
}

func TestGenerate_ControlFlow(t *testing.T) {
	syms := NewSymbolTable()
	// if (1) { }
	stmts := []Stmt{
		&FunctionDecl{
			Name: "main",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&IfStmt{
						Condition: &Literal{Value: 1},
						Body:      &BlockStmt{},
					},
					&WhileStmt{
						Condition: &Literal{Value: 1},
						Body:      &BlockStmt{},
					},
				},
			},
		},
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "JZ")
	assertContains(t, code, "JMP")
}

func TestGenerate_Errors(t *testing.T) {
	tests := []struct {
		name      string
		stmts     []Stmt
		errSubstr string
	}{
		{
			name: "Undefined variable",
			stmts: []Stmt{
				&FunctionDecl{
					Name: "main",
					Body: &BlockStmt{
						Stmts: []Stmt{
							&Assignment{Op: ASSIGN,
								Left:  &VarRef{Name: "undefined_var"},
								Value: &Literal{Value: 1},
							},
						},
					},
				},
			},
			errSubstr: "undefined variable",
		},
		{
			name: "Address of non-variable",
			stmts: []Stmt{
				&VariableDecl{
					Name: "x",
					Init: &UnaryExpr{
						Op:    AND,
						Right: &Literal{Value: 1},
					},
				},
				&FunctionDecl{Name: "main", Body: &BlockStmt{}}, // Trigger __init
			},
			errSubstr: "cannot take address of expression type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syms := NewSymbolTable()
			_, err := Generate(tt.stmts, syms)
			if err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.errSubstr)
			} else if !strings.Contains(err.Error(), tt.errSubstr) {
				t.Errorf("Expected error containing %q, got %q", tt.errSubstr, err.Error())
			}
		})
	}
}

func TestPointerArithmetic(t *testing.T) {
	// Source: int* p = 0x2000; *(p + 1) = 10;
	// AST:
	// VariableDecl{Name: "p", Init: Literal{0x2000}}
	// Assignment{Op: ASSIGN, Left: UnaryExpr{Op: STAR, Right: BinaryExpr{PLUS, VarRef{p}, Literal{1}}}, Value: Literal{10}}

	syms := NewSymbolTable()
	stmts := []Stmt{
		&VariableDecl{
			Name: "p",
			Init: &Literal{Value: 0x2000},
		},
		&FunctionDecl{
			Name: "main",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&Assignment{Op: ASSIGN,
						Left: &UnaryExpr{
							Op: STAR,
							Right: &BinaryExpr{
								Op:    PLUS,
								Left:  &VarRef{Name: "p"},
								Right: &Literal{Value: 1},
							},
						},
						Value: &Literal{Value: 10},
					},
				},
			},
		},
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "ADD R1, R0")

	assertContains(t, code, "PUSH R0")
	assertContains(t, code, "LDI R0, 10")
	assertContains(t, code, "POP R1")
	assertContains(t, code, "ST  [R1], R0")
}

func TestGenerate_BitwiseOperators(t *testing.T) {
	syms := NewSymbolTable()
	// Use vars to prevent constant folding
	stmts := []Stmt{
		&VariableDecl{Name: "v1", Init: &Literal{Value: 0xFF0F}},
		&VariableDecl{Name: "v2", Init: &Literal{Value: 0x00FF}},
		&VariableDecl{Name: "a", Init: &BinaryExpr{Op: AND, Left: &VarRef{Name: "v1"}, Right: &VarRef{Name: "v2"}}},
		&VariableDecl{Name: "b", Init: &BinaryExpr{Op: PIPE, Left: &VarRef{Name: "v1"}, Right: &VarRef{Name: "v2"}}},
		&VariableDecl{Name: "c", Init: &BinaryExpr{Op: CARET, Left: &VarRef{Name: "v1"}, Right: &VarRef{Name: "v2"}}},
		&FunctionDecl{Name: "main", Body: &BlockStmt{}}, // Trigger __init
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "AND R1, R0")
	assertContains(t, code, "OR  R1, R0")
	assertContains(t, code, "XOR R1, R0")
}

func TestGenerate_BitwiseNot(t *testing.T) {
	syms := NewSymbolTable()
	stmts := []Stmt{
		&VariableDecl{Name: "x", Init: &Literal{Value: 0}},
		&VariableDecl{Name: "d", Init: &UnaryExpr{Op: TILDE, Right: &VarRef{Name: "x"}}},
		&FunctionDecl{Name: "main", Body: &BlockStmt{}}, // Trigger __init
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "NOT R0")
}

func TestGenerate_Modulo(t *testing.T) {
	syms := NewSymbolTable()
	stmts := []Stmt{
		&VariableDecl{Name: "v1", Init: &Literal{Value: 10}},
		&VariableDecl{Name: "v2", Init: &Literal{Value: 3}},
		&VariableDecl{Name: "e", Init: &BinaryExpr{Op: PERCENT, Left: &VarRef{Name: "v1"}, Right: &VarRef{Name: "v2"}}},
		&FunctionDecl{Name: "main", Body: &BlockStmt{}}, // Trigger __init
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "MOV R3, R1")
	assertContains(t, code, "DIV R1, R0")
	assertContains(t, code, "MUL R1, R0")
	assertContains(t, code, "SUB R3, R1")
	assertContains(t, code, "MOV R0, R3")
}

func TestGenerate_Shifts(t *testing.T) {
	syms := NewSymbolTable()
	stmts := []Stmt{
		&VariableDecl{Name: "v1", Init: &Literal{Value: 1}},
		&VariableDecl{Name: "v2", Init: &Literal{Value: 4}},
		&VariableDecl{Name: "f", Init: &BinaryExpr{Op: SHL_OP, Left: &VarRef{Name: "v1"}, Right: &VarRef{Name: "v2"}}},
		&VariableDecl{Name: "g", Init: &BinaryExpr{Op: SHR_OP, Left: &VarRef{Name: "v1"}, Right: &VarRef{Name: "v2"}}},
		&FunctionDecl{Name: "main", Body: &BlockStmt{}}, // Trigger __init
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "SHL R1, R0")
	assertContains(t, code, "SHR R1, R0")
}

func TestGenerate_NewOperators(t *testing.T) {
	syms := NewSymbolTable()
	stmts := []Stmt{
		&VariableDecl{Name: "v1", Init: &Literal{Value: 2}},
		&VariableDecl{Name: "v2", Init: &Literal{Value: 3}},
		&VariableDecl{Name: "x", Init: &BinaryExpr{Op: STAR, Left: &VarRef{Name: "v1"}, Right: &VarRef{Name: "v2"}}},
		&VariableDecl{Name: "y", Init: &BinaryExpr{Op: SLASH, Left: &VarRef{Name: "v1"}, Right: &VarRef{Name: "v2"}}},
		&VariableDecl{Name: "z", Init: &BinaryExpr{Op: NOT_EQ, Left: &VarRef{Name: "v1"}, Right: &VarRef{Name: "v2"}}},
		&VariableDecl{Name: "w", Init: &BinaryExpr{Op: LESS, Left: &VarRef{Name: "v1"}, Right: &VarRef{Name: "v2"}}},
		&VariableDecl{Name: "v", Init: &BinaryExpr{Op: GREATER, Left: &VarRef{Name: "v1"}, Right: &VarRef{Name: "v2"}}},
		&FunctionDecl{Name: "main", Body: &BlockStmt{}}, // Trigger __init
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "MUL R1, R0")
	assertContains(t, code, "DIV R1, R0")
	assertContains(t, code, "SUB R1, R0")
	assertContains(t, code, "JNZ")
	assertContains(t, code, "JN")
}

func TestGenerate_Arrays(t *testing.T) {
	syms := NewSymbolTable()
	// int arr[10];
	// arr[0] = 5;
	stmts := []Stmt{
		&VariableDecl{Name: "arr", IsArray: true, ArraySizes: []int{10}},
		&FunctionDecl{
			Name: "main",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&Assignment{Op: ASSIGN,
						Left: &IndexExpr{
							Left:    &VarRef{Name: "arr"},
							Indices: []Expr{&Literal{Value: 0}},
						},
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

	assertContains(t, code, "LDI R1, arr    ; &arr")
	assertContains(t, code, "PUSH R0")
	assertContains(t, code, "ADD R1, R0")
	assertContains(t, code, "ST  [R1], R0")
}

func TestGenerate_Structs(t *testing.T) {
	syms := NewSymbolTable()
	// struct Point { int x; int y; };
	// struct Point p;
	// p.x = 10;

	stmts := []Stmt{
		&StructDecl{
			Name: "Point",
			Fields: []VariableDecl{
				{Name: "x"},
				{Name: "y"},
			},
		},
		&VariableDecl{Name: "p", IsStruct: true, StructName: "Point"},
		&FunctionDecl{
			Name: "main",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&Assignment{Op: ASSIGN,
						Left: &MemberExpr{
							Left:   &VarRef{Name: "p"},
							Member: "x",
						},
						Value: &Literal{Value: 10},
					},
				},
			},
		},
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "struct Point defined (size 4)")
	assertContains(t, code, "LDI R3, 0") // Offset of x
	assertContains(t, code, "ADD R1, R3")
}

func TestGenerate_BoundaryValues(t *testing.T) {
	syms := NewSymbolTable()
	stmts := []Stmt{
		&VariableDecl{Name: "x", Init: &Literal{Value: 0xFFFF}},
		&VariableDecl{Name: "y", Init: &Literal{Value: 0}},
		&FunctionDecl{Name: "main", Body: &BlockStmt{}}, // Trigger __init
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, ".WORD 65535")
	assertContains(t, code, ".WORD 0")
}

func TestGenerate_ComplexCalls(t *testing.T) {
	syms := NewSymbolTable()
	// Declare functions to avoid lookup errors
	syms.Allocate("foo", TypeInfo{}, 1)
	syms.Allocate("bar", TypeInfo{}, 1)

	// foo(bar(1))
	stmts1 := []Stmt{
		&FunctionDecl{
			Name: "main",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&ExprStmt{
						Expr: &FunctionCall{
							Name: "foo",
							Args: []Expr{
								&FunctionCall{
									Name: "bar",
									Args: []Expr{&Literal{Value: 1}},
								},
							},
						},
					},
				},
			},
		},
	}
	code1, err := Generate(stmts1, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// bar(1) called first
	assertContains(t, code1, "LDI R0, 1")
	assertContains(t, code1, "PUSH R0")
	assertContains(t, code1, "CALL bar")
	// result pushed
	assertContains(t, code1, "PUSH R0")
	// foo called
	assertContains(t, code1, "CALL foo")

	// foo(v1 + v2)
	stmts2 := []Stmt{
		&VariableDecl{Name: "v1", Init: &Literal{Value: 1}},
		&VariableDecl{Name: "v2", Init: &Literal{Value: 2}},
		&FunctionDecl{
			Name: "main",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&ExprStmt{
						Expr: &FunctionCall{
							Name: "foo",
							Args: []Expr{
								&BinaryExpr{
									Op:    PLUS,
									Left:  &VarRef{Name: "v1"},
									Right: &VarRef{Name: "v2"},
								},
							},
						},
					},
				},
			},
		},
	}
	code2, err := Generate(stmts2, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	assertContains(t, code2, "ADD R1, R0") // Result in R0
	assertContains(t, code2, "PUSH R0")    // Push arg
	assertContains(t, code2, "CALL foo")
}

func TestGenerate_MultipleReturns(t *testing.T) {
	syms := NewSymbolTable()
	// int f() { if (1) { return 1; } return 0; }
	stmts := []Stmt{
		&FunctionDecl{
			Name: "f",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&IfStmt{
						Condition: &Literal{Value: 1},
						Body: &BlockStmt{
							Stmts: []Stmt{
								&ReturnStmt{Expr: &Literal{Value: 1}},
							},
						},
					},
					&ReturnStmt{Expr: &Literal{Value: 0}},
				},
			},
		},
		&FunctionDecl{
			Name: "main",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&ExprStmt{
						Expr: &FunctionCall{
							Name: "f",
						},
					},
				},
			},
		},
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify epilogue exists
	assertContains(t, code, "STSP R2")
	assertContains(t, code, "POP R2")
	assertContains(t, code, "RET")
}

func TestGenerate_Asm(t *testing.T) {
	syms := NewSymbolTable()
	stmts := []Stmt{
		&FunctionDecl{
			Name: "main",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&AsmStmt{Instruction: "WFI"},
					&AsmStmt{Instruction: "LDI R0, 1"},
				},
			},
		},
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "WFI")
	assertContains(t, code, "LDI R0, 1")
}

func TestGenerate_Switch(t *testing.T) {
	syms := NewSymbolTable()
	// switch (x) { case 1: y=2; case 2: y=3; default: y=4; }
	syms.Allocate("x", TypeInfo{}, 2)
	syms.Allocate("y", TypeInfo{}, 2)

	stmts := []Stmt{
		&FunctionDecl{
			Name: "main",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&SwitchStmt{
						Target: &VarRef{Name: "x"},
						Cases: []CaseClause{
							{
								Value: &Literal{Value: 1},
								Body: []Stmt{
									&Assignment{Op: ASSIGN, Left: &VarRef{Name: "y"}, Value: &Literal{Value: 2}},
								},
							},
							{
								Value: &Literal{Value: 2},
								Body: []Stmt{
									&Assignment{Op: ASSIGN, Left: &VarRef{Name: "y"}, Value: &Literal{Value: 3}},
								},
							},
						},
						Default: []Stmt{
							&Assignment{Op: ASSIGN, Left: &VarRef{Name: "y"}, Value: &Literal{Value: 4}},
						},
					},
				},
			},
		},
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "switch x")
	assertContains(t, code, "PUSH R0") // push target
	assertContains(t, code, "LDSP R1") // get target
	assertContains(t, code, "SUB R1, R0") // compare
	assertContains(t, code, "JZ") // jump case
	assertContains(t, code, "POP R0") // clean stack
}

func TestDeadCodeElimination(t *testing.T) {
	syms := NewSymbolTable()
	// int used_func() { return 1; }
	// int dead_func() { return 0; }
	// int main() { return used_func(); }

	stmts := []Stmt{
		&FunctionDecl{
			Name: "used_func",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&ReturnStmt{Expr: &Literal{Value: 1}},
				},
			},
		},
		&FunctionDecl{
			Name: "dead_func",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&ReturnStmt{Expr: &Literal{Value: 0}},
				},
			},
		},
		&FunctionDecl{
			Name: "main",
			Body: &BlockStmt{
				Stmts: []Stmt{
					&ReturnStmt{
						Expr: &FunctionCall{
							Name: "used_func",
						},
					},
				},
			},
		},
	}

	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertContains(t, code, "used_func:")

	if strings.Contains(code, "dead_func:") {
		t.Errorf("Expected code NOT to contain 'dead_func:', but it did.")
	}
}
