package compiler

import (
	"testing"
)

func TestPointerCasts(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "cast_int_pointer",
			source: `
				int main() {
					int* p = (int*)0x8000;
					return 0;
				}
			`,
		},
		{
			name: "cast_char_pointer",
			source: `
				int main() {
					char* p = (char*)0x8000;
					return 0;
				}
			`,
		},
		{
			name: "cast_memset_example",
			source: `
				void memset(int* ptr, int val, int size) {}
				int main() {
					memset((int*)0x8000, 0, 10);
					return 0;
				}
			`,
		},
		{
			name: "cast_char_pointer_dereference",
			source: `
				int main() {
					char* p = (char*)0x8000;
					char val = *p;
					return 0;
				}
			`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.source)
			if err != nil {
				t.Fatalf("Lexer error: %v", err)
			}
			stmts, err := Parse(tokens, tt.source)
			if err != nil {
				t.Fatalf("Parser error: %v", err)
			}
			syms := NewSymbolTable()
			_, err = Generate(stmts, syms)
			if err != nil {
				t.Fatalf("Codegen error: %v", err)
			}
		})
	}
}
