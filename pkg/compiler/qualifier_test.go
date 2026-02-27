package compiler

import (
	"testing"
)

// TestLexer_Qualifiers verifies that volatile, const, static, and extern
// each produce a distinct keyword token.
func TestLexer_Qualifiers(t *testing.T) {
	tests := []struct {
		input    string
		wantType TokenType
	}{
		{"volatile", VOLATILE},
		{"const", CONST},
		{"static", STATIC},
		{"extern", EXTERN},
	}

	for _, tt := range tests {
		tokens, err := Lex(tt.input)
		if err != nil {
			t.Fatalf("Lex(%q) failed: %v", tt.input, err)
		}
		if len(tokens) < 1 || tokens[0].Type != tt.wantType {
			got := EOF
			if len(tokens) > 0 {
				got = tokens[0].Type
			}
			t.Errorf("Lex(%q): expected token %s, got %s", tt.input, tt.wantType, got)
		}
	}
}

// TestParser_QualifierOnVariableDecl verifies that qualifiers are silently
// consumed and the resulting AST is identical to the unqualified form.
func TestParser_QualifierOnVariableDecl(t *testing.T) {
	// static const int x = 5; should parse identically to int x = 5;
	qualified := `static const int x = 5;`
	plain := `int x = 5;`

	for _, src := range []string{qualified, plain} {
		tokens, err := Lex(src)
		if err != nil {
			t.Fatalf("Lex(%q): %v", src, err)
		}
		stmts, err := Parse(tokens, src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		if len(stmts) != 1 {
			t.Fatalf("Parse(%q): expected 1 stmt, got %d", src, len(stmts))
		}
		decl, ok := stmts[0].(*VariableDecl)
		if !ok {
			t.Fatalf("Parse(%q): expected *VariableDecl", src)
		}
		if decl.Name != "x" {
			t.Errorf("Parse(%q): Name = %q, want %q", src, decl.Name, "x")
		}
		lit, ok := decl.Init.(*Literal)
		if !ok || lit.Value != 5 {
			t.Errorf("Parse(%q): Init = %v, want Literal(5)", src, decl.Init)
		}
	}
}

// TestParser_VolatileUnsignedPointer verifies the acceptance-criteria example
// from Ticket 5: volatile unsigned int * timer = 0xFF10;
func TestParser_VolatileUnsignedPointer(t *testing.T) {
	src := `volatile unsigned int * timer = 0xFF10;`
	tokens, err := Lex(src)
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	stmts, err := Parse(tokens, src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(stmts))
	}
	decl, ok := stmts[0].(*VariableDecl)
	if !ok {
		t.Fatalf("expected *VariableDecl, got %T", stmts[0])
	}
	if decl.Name != "timer" {
		t.Errorf("Name = %q, want %q", decl.Name, "timer")
	}
	if !decl.IsUnsigned {
		t.Error("expected IsUnsigned=true")
	}
	if decl.PointerLevel != 1 {
		t.Errorf("PointerLevel = %d, want 1", decl.PointerLevel)
	}
}

// TestParser_ExternFunctionDecl verifies that extern before a function
// declaration is silently consumed.
func TestParser_ExternFunctionDecl(t *testing.T) {
	src := `extern int add(int a, int b) { return a; }`
	tokens, err := Lex(src)
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	stmts, err := Parse(tokens, src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(stmts))
	}
	fn, ok := stmts[0].(*FunctionDecl)
	if !ok {
		t.Fatalf("expected *FunctionDecl, got %T", stmts[0])
	}
	if fn.Name != "add" {
		t.Errorf("Name = %q, want %q", fn.Name, "add")
	}
}
