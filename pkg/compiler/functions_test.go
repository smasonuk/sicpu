package compiler

import (
	"strings"
	"testing"
)

func TestFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "simple function",
			input: `
			int foo() {
				return 42;
			}
			int main() {
				int x = foo();
				return x;
			}
			`,
			contains: []string{
				"foo:",
				"PUSH R2",    // Save FP
				"LDSP R2",    // Set FP
				"LDI R0, 42",
				"STSP R2",    // Restore SP
				"POP R2",     // Restore FP
				"RET",
				"CALL foo",
			},
		},
		{
			name: "function with args",
			input: `
			int add(int a, int b) {
				return a + b;
			}
			int main() {
				int x = add(1, 2);
				return x;
			}
			`,
			contains: []string{
				"add:",
				"LDI R3, 65534", // Offset for param 'a' (FP-2) -> 0xFFFE
				"ADD R1, R3",
				"LDI R3, 65532", // Offset for param 'b' (FP-4) -> 0xFFFC
				"ADD R1, R3",
				"CALL add",
				"STSP R3",    // Stack cleanup (only for args > 4)
			},
		},
		{
			name: "local variables",
			input: `
			int foo() {
				int a = 10;
				int b = 20;
				return a + b;
			}
			int main() {
				int x = foo();
				return x;
			}
			`,
			contains: []string{
				"STSP R3",       // Allocate space for locals
				"LDI R3, 65534", // Offset -2 (for 'a') -> 0xFFFE
				"LDI R3, 65532", // Offset -4 (for 'b') -> 0xFFFC
			},
		},
		{
			name: "recursion (fibonacci)",
			input: `
			int fib(int n) {
				if (n == 0) return 0;
				if (n == 1) return 1;
				return fib(n-1) + fib(n-2);
			}
			int main() {
				int x = fib(5);
				return x;
			}
			`,
			contains: []string{
				"CALL fib",
				"PUSH R2",
				"LDSP R2",
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

			syms := NewSymbolTable()
			code, err := Generate(stmts, syms)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			for _, s := range tt.contains {
				if !strings.Contains(code, s) {
					t.Errorf("Generated code missing %q. Code:\n%s", s, code)
				}
			}
		})
	}
}
