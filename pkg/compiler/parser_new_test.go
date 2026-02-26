package compiler

import (
	"testing"
)

func TestParserNewFeatures(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "Break and Continue",
			src: `
void main() {
    while(1) {
        break;
        continue;
    }
}
`,
		},
		{
			name: "Unary Minus",
			src: `
void main() {
    int x = -1;
    int y = -x;
}
`,
		},
		{
			name: "Explicit Cast",
			src: `
void main() {
    int x = (int)10;
    byte b = (byte)x;
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.src)
			if err != nil {
				t.Fatalf("Lex failed: %v", err)
			}
			stmts, err := Parse(tokens, tt.src)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if len(stmts) == 0 {
				t.Errorf("Expected statements, got 0")
			}
		})
	}
}
