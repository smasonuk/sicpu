package compiler

import (
	"strings"
	"testing"
)

func TestPointers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "address of variable",
			input: `
			int x = 10;
			int p = &x;
			int main() {
				return p;
			}
			`,
			contains: []string{
				"LDI R1, x",      // &x (global label)
				"MOV R0, R1",     // result of &x
			},
		},
		{
			name: "dereference",
			input: `
			int x = 10;
			int p = &x;
			int y = *p;
			int main() {
				return y;
			}
			`,
			contains: []string{
				"MOV R1, R0",   // address of p in R0 -> R1
				"LD  R0, [R1]", // load value at address
			},
		},
		{
			name: "dereference assignment",
			input: `
			int x = 10;
			int p = &x;
			int main() {
				*p = 20;
				return x;
			}
			`,
			contains: []string{
				"PUSH R1",      // save pointer address
				"LDI R0, 20",   // value
				"POP R1",       // restore pointer address
				"ST  [R1], R0", // store
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
