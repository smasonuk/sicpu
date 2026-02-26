package compiler

import (
	"strings"
	"testing"
)

func TestPreprocessDefines(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected string
	}{
		{
			name: "Simple Define",
			src: `
#define A 10
int x = A;
`,
			expected: `

int x = 10;
`,
		},
		{
			name: "Nested Define",
			src: `
#define OFFSET 10
#define BASE (0x100 + OFFSET)
int y = BASE;
`,
			expected: `


int y = (0x100 + 10);
`,
		},
		{
			name: "String Literal Ignored",
			src: `
#define A 10
char *s = "A";
`,
			expected: `

char *s = "A";
`,
		},
		{
			name: "Word Boundary",
			src: `
#define A 10
int AA = A;
`,
			expected: `

int AA = 10;
`,
		},
        {
            name: "Char Literal Ignored",
            src: `
#define C 65
char c = 'C';
`,
            expected: `

char c = 'C';
`,
        },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Preprocess(tt.src, ".")
			if err != nil {
				t.Fatalf("Preprocess failed: %v", err)
			}
			// Normalize newlines for comparison: remove all newlines to just check content sequence
            // or just TrimSpace.
            // Let's use TrimSpace which is what I planned.
			got = strings.TrimSpace(got)
			expected := strings.TrimSpace(tt.expected)
			if got != expected {
				t.Errorf("Preprocess() = %q, want %q", got, expected)
			}
		})
	}
}
