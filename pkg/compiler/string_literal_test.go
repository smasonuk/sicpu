package compiler

import (
	"strings"
	"testing"
)

func TestStringLiteral(t *testing.T) {
	// Source code with string literals
	src := `
		int main() {
			print("Hello");
			print("World");
			print("Hello"); // duplicate string, should reuse label
		}
	`

	// 1. Lex
	tokens, err := Lex(src)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}

	// 2. Parse
	stmts, err := Parse(tokens, src)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// 3. Generate
	syms := NewSymbolTable()
	// print function is an intrinsic, so we don't need to declare it, 
	// but we might need to handle the function call.
	// The current codegen handles 'print' as an intrinsic in FunctionCall case.
	
	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 4. Verify
	// Check for string pool labels
	if !strings.Contains(code, "S0: .STRING \"Hello\"") {
		t.Errorf("Expected S0: .STRING \"Hello\", not found in:\n%s", code)
	}
	if !strings.Contains(code, "S1: .STRING \"World\"") {
		t.Errorf("Expected S1: .STRING \"World\", not found in:\n%s", code)
	}

	// Check for usage of labels
	// "Hello" is used twice, so LDI R0, S0 should appear twice
	countS0 := strings.Count(code, "LDI R0, S0")
	if countS0 != 2 {
		t.Errorf("Expected usage of S0 2 times, found %d", countS0)
	}

	// "World" is used once
	countS1 := strings.Count(code, "LDI R0, S1")
	if countS1 != 1 {
		t.Errorf("Expected usage of S1 1 time, found %d", countS1)
	}
}

func TestStringLiteralEscape(t *testing.T) {
	src := `
		int main() {
			print("Line1\nLine2");
		}
	`
	tokens, err := Lex(src)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}
	stmts, err := Parse(tokens, src)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	syms := NewSymbolTable()
	code, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check that \n is correctly emitted in the .STRING directive
	// The scanner converts \n to a newline character.
	// The codegen uses fmt.Sprintf("%q", val), so newline char should become "\n" string in the output.
	// e.g. .STRING "Line1\nLine2"
	
	expected := `.STRING "Line1\nLine2"`
	if !strings.Contains(code, expected) {
		t.Errorf("Expected %s, not found in:\n%s", expected, code)
	}
}
