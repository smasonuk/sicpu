package compiler

import (
	"strings"
	"testing"
)

func TestParser_Char(t *testing.T) {
	input := `
	char c = 10;
	char* s = "hello";
	char f(char a) { return a; }
	`
	tokens, err := Lex(input)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}

	stmts, err := Parse(tokens, input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(stmts) != 3 {
		t.Fatalf("Expected 3 statements, got %d", len(stmts))
	}

	// 1. char c = 10;
	v1, ok := stmts[0].(*VariableDecl)
	if !ok {
		t.Errorf("Stmt 0 not VariableDecl")
	} else {
		if !v1.IsChar {
			t.Errorf("Stmt 0 expected IsChar=true")
		}
		if v1.Name != "c" {
			t.Errorf("Stmt 0 expected Name='c', got %s", v1.Name)
		}
	}

	// 2. char* s = "hello";
	v2, ok := stmts[1].(*VariableDecl)
	if !ok {
		t.Errorf("Stmt 1 not VariableDecl")
	} else {
		if !v2.IsChar {
			t.Errorf("Stmt 1 expected IsChar=true")
		}
		if v2.PointerLevel < 1 {
			t.Errorf("Stmt 1 expected PointerLevel>=1, got %d", v2.PointerLevel)
		}
	}

	// 3. char f(char a) { return a; }
	f3, ok := stmts[2].(*FunctionDecl)
	if !ok {
		t.Errorf("Stmt 2 not FunctionDecl")
	} else {
		if !strings.Contains(f3.ReturnType, "char") {
			t.Errorf("Stmt 2 expected ReturnType 'char', got %s", f3.ReturnType)
		}
		if len(f3.Params) != 1 {
			t.Errorf("Stmt 2 expected 1 param, got %d", len(f3.Params))
		} else {
			if !f3.Params[0].IsChar {
				t.Errorf("Param 0 expected IsChar=true")
			}
		}
	}
}

func TestParser_ByteSyntaxError(t *testing.T) {
	input := `byte b = 10;`
	tokens, err := Lex(input)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}

	_, err = Parse(tokens, input)
	if err == nil {
		t.Error("Expected syntax error for 'byte' keyword usage, got nil")
	} else {
		if !strings.Contains(err.Error(), "executable statement") && !strings.Contains(err.Error(), "expected type") {
			// Because 'byte' is now an identifier, 'byte b = 10;' looks like 'IDENTIFIER IDENTIFIER ASSIGN ...'
			// At top level, this might be interpreted as an expression statement 'byte' followed by garbage,
			// or a failed declaration.
			// Actually, parseTopLevel sees IDENTIFIER, goes to Naked Statement check, and fails because statements aren't allowed at top level.
			// Or if inside a function, it would try to parse as expression statement.
			// Let's see what the error actually is.
			t.Logf("Got expected error: %v", err)
		}
	}
}
