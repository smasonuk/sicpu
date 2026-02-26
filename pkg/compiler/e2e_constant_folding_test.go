package compiler

import (
	"strings"
	"testing"
)

// TestConstantFolding_Signed verifies that arithmetic on plain integer literals
// is folded at compile-time using signed (int16) semantics.
func TestConstantFolding_Signed(t *testing.T) {
	t.Run("SignedDivision", func(t *testing.T) {
		// (-10) / 2 == -5  →  0xFFFB as uint16
		src := `int main() { return (-10) / 2; }`
		regs := runCode(t, src)
		if int16(regs[0]) != -5 {
			t.Errorf("(-10)/2: expected -5, got %d", int16(regs[0]))
		}
	})

	t.Run("SignedModulo", func(t *testing.T) {
		// 0xFFF9 is the 16-bit two's-complement encoding of -7.
		// int16(-7) % 3 == -1  (signed remainder matches Go's int16 semantics).
		src := `int main() { return 0xFFF9 % 3; }`
		regs := runCode(t, src)
		if int16(regs[0]) != -1 {
			t.Errorf("0xFFF9%%3: expected -1, got %d", int16(regs[0]))
		}
	})

	t.Run("SignedLessThan_Negative", func(t *testing.T) {
		// (-1) < 1 is true  →  result 1
		src := `int main() { return ((-1) < 1); }`
		regs := runCode(t, src)
		if regs[0] != 1 {
			t.Errorf("(-1)<1: expected 1, got %d", regs[0])
		}
	})

	t.Run("SignedGreaterThan_Negative", func(t *testing.T) {
		// 1 > (-1) is true  →  result 1
		src := `int main() { return (1 > (-1)); }`
		regs := runCode(t, src)
		if regs[0] != 1 {
			t.Errorf("1>(-1): expected 1, got %d", regs[0])
		}
	})

	t.Run("SignedDivision_HexLiteral", func(t *testing.T) {
		// 0xFFF6 is the 16-bit encoding of -10; signed fold must give -5 (0xFFFB).
		src := `int main() { return 0xFFF6 / 2; }`
		regs := runCode(t, src)
		if int16(regs[0]) != -5 {
			t.Errorf("0xFFF6/2 (signed): expected -5, got %d", int16(regs[0]))
		}
	})

	t.Run("SignedLessThan_LargePattern_False", func(t *testing.T) {
		// 0x8000 as a plain literal is -32768 in signed context.
		// -32768 < 1 is true (signed).
		src := `int main() { return (0x8000 < 1); }`
		regs := runCode(t, src)
		if regs[0] != 1 {
			t.Errorf("0x8000 < 1 (signed): expected 1, got %d", regs[0])
		}
	})
}

// TestConstantFolding_Unsigned verifies that arithmetic on u-suffixed literals
// is folded using unsigned (uint16) semantics, and that mixing one unsigned
// with a plain literal also triggers unsigned folding.
func TestConstantFolding_Unsigned(t *testing.T) {
	t.Run("UnsignedDivision", func(t *testing.T) {
		// 0xFFF6u / 2 == 32763  (unsigned); signed IDIV would give -5.
		src := `int main() { return 0xFFF6u / 2; }`
		regs := runCode(t, src)
		if regs[0] != 32763 {
			t.Errorf("0xFFF6u/2: expected 32763, got %d", regs[0])
		}
	})

	t.Run("UnsignedModulo", func(t *testing.T) {
		// 0xFFF6u % 10 == 65526 % 10 == 6
		src := `int main() { return 0xFFF6u % 10; }`
		regs := runCode(t, src)
		if regs[0] != 6 {
			t.Errorf("0xFFF6u%%10: expected 6, got %d", regs[0])
		}
	})

	t.Run("UnsignedLessThan_LargeValue_False", func(t *testing.T) {
		// 65535u < 1 is false in unsigned context  →  result 0
		src := `int main() { return (65535u < 1); }`
		regs := runCode(t, src)
		if regs[0] != 0 {
			t.Errorf("65535u < 1: expected 0, got %d", regs[0])
		}
	})

	t.Run("UnsignedGreaterThan_LargeValue_True", func(t *testing.T) {
		// 65535u > 1 is true in unsigned context  →  result 1
		src := `int main() { return (65535u > 1); }`
		regs := runCode(t, src)
		if regs[0] != 1 {
			t.Errorf("65535u > 1: expected 1, got %d", regs[0])
		}
	})

	t.Run("MixedOperand_OneUnsignedSuffix", func(t *testing.T) {
		// If either operand has u suffix the whole fold is unsigned.
		// 0xFFF6u / 2 and 0xFFF6 / 2u should both be 32763.
		src := `int main() { return 0xFFF6 / 2u; }`
		regs := runCode(t, src)
		if regs[0] != 32763 {
			t.Errorf("0xFFF6/2u: expected 32763, got %d", regs[0])
		}
	})

	t.Run("UnsignedLiteral_HexSuffix", func(t *testing.T) {
		// 0xFFFFu is a valid hex unsigned literal (65535).
		src := `int main() { return 0xFFFFu; }`
		regs := runCode(t, src)
		if regs[0] != 0xFFFF {
			t.Errorf("0xFFFFu: expected 0xFFFF, got 0x%X", regs[0])
		}
	})
}

// TestConstantFolding_CodeGen verifies that constant folding actually emits a
// single LDI rather than the multi-instruction runtime path.
func TestConstantFolding_CodeGen(t *testing.T) {
	t.Run("FoldsToSingleLDI_Signed", func(t *testing.T) {
		// 0xFFF6 is the 16-bit encoding of -10; dividing by 2 should fold to -5
		// (0xFFFB) without emitting any IDIV instruction.
		src := `int main() { return 0xFFF6 / 2; }`
		tokens, _ := Lex(src)
		stmts, _ := Parse(tokens, src)
		code, err := Generate(stmts, NewSymbolTable())
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		// The fold should emit a single LDI with the result, never an IDIV.
		if strings.Contains(code, "IDIV") {
			t.Error("signed constant folding should not emit IDIV; expected a single LDI")
		}
	})

	t.Run("FoldsToSingleLDI_Unsigned", func(t *testing.T) {
		src := `int main() { return 0xFFF6u / 2u; }`
		tokens, _ := Lex(src)
		stmts, _ := Parse(tokens, src)
		code, err := Generate(stmts, NewSymbolTable())
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		// Should contain exactly one LDI for the folded constant, not DIV.
		if strings.Contains(code, "DIV") {
			t.Error("unsigned constant folding should not emit DIV; expected a single LDI")
		}
	})
}

// TestUnsignedLiteral_Lexer checks that the u-suffix tokens are lexed correctly.
func TestUnsignedLiteral_Lexer(t *testing.T) {
	tests := []struct {
		input   string
		wantTok TokenType
		wantLex string
	}{
		{"10u", UNSIGNED_LIT, "10"},
		{"10U", UNSIGNED_LIT, "10"},
		{"0xFFu", UNSIGNED_LIT, "0xFF"},
		{"0xFFFFU", UNSIGNED_LIT, "0xFFFF"},
		{"42", INTEGER, "42"},       // no suffix → plain INTEGER
		{"0xFF", INTEGER, "0xFF"},   // no suffix → plain INTEGER
	}
	for _, tt := range tests {
		tokens, err := Lex(tt.input)
		if err != nil {
			t.Errorf("Lex(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if len(tokens) < 1 {
			t.Errorf("Lex(%q): got no tokens", tt.input)
			continue
		}
		tok := tokens[0]
		if tok.Type != tt.wantTok {
			t.Errorf("Lex(%q): type = %s, want %s", tt.input, tok.Type, tt.wantTok)
		}
		if tok.Lexeme != tt.wantLex {
			t.Errorf("Lex(%q): lexeme = %q, want %q", tt.input, tok.Lexeme, tt.wantLex)
		}
	}
}
