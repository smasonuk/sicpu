package compiler

import "testing"

func TestLexerCharLiterals(t *testing.T) {
	input := `'A' '\n' '\0' '\'' '\\'`
	tokens, err := Lex(input)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}

	expected := []string{"65", "10", "0", "39", "92"}
	if len(tokens)-1 != len(expected) { // -1 for EOF
		t.Errorf("Expected %d tokens, got %d", len(expected), len(tokens)-1)
	}

	for i, exp := range expected {
		if tokens[i].Lexeme != exp {
			t.Errorf("Token %d: expected %s, got %s", i, exp, tokens[i].Lexeme)
		}
		if tokens[i].Type != INTEGER {
			t.Errorf("Token %d: expected INTEGER, got %s", i, tokens[i].Type)
		}
	}
}

func TestLexerKeywords(t *testing.T) {
    input := "break continue"
    tokens, err := Lex(input)
    if err != nil {
        t.Fatalf("Lex failed: %v", err)
    }

    if len(tokens)-1 != 2 {
        t.Fatalf("Expected 2 tokens")
    }

    if tokens[0].Type != BREAK {
        t.Errorf("Expected BREAK, got %s", tokens[0].Type)
    }
    if tokens[1].Type != CONTINUE {
        t.Errorf("Expected CONTINUE, got %s", tokens[1].Type)
    }
}
