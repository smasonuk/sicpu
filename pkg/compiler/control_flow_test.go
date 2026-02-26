package compiler

import (
	"strings"
	"testing"
)

func TestControlFlow(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name: "if statement",
			input: `
			int main() {
				int x = 1;
				if (x == 1) {
					x = 2;
				}
				return x;
			}
			`,
			contains: []string{"JZ", "LDI R1, 0", "SUB R0, R1"},
		},
		{
			name: "if-else statement",
			input: `
			int main() {
				int x = 1;
				if (x == 1) {
					x = 2;
				} else {
					x = 3;
				}
				return x;
			}
			`,
			contains: []string{"JZ", "JMP", "LDI R1, 0", "SUB R0, R1"},
		},
		{
			name: "while loop",
			input: `
			int main() {
				int x = 0;
				while (x == 0) {
					x = 1;
				}
				return x;
			}
			`,
			contains: []string{"JZ", "JMP", "LDI R1, 0", "SUB R0, R1"},
		},
		{
			name: "nested blocks",
			input: `
			int main() {
				int x = 1;
				{
					int y = 2;
					{
						x = y;
					}
				}
				return x;
			}
			`,
			contains: []string{"LDI", "ST"},
		},
		{
			name: "empty block",
			input: `
			int main() {
				int x = 1;
				{}
				return x;
			}
			`,
			contains: []string{"LDI", "ST"},
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
