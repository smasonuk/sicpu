package compiler

import (
	"testing"
)

func TestLexer_Char(t *testing.T) {
	input := "char c = 'a';"
	tokens, err := Lex(input)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}

	expected := []TokenType{
		CHAR, IDENTIFIER, ASSIGN, INTEGER, SEMICOLON, EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("Expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("Token %d: expected %s, got %s", i, expected[i], tok.Type)
		}
	}
}

func TestLexer_ByteDeprecated(t *testing.T) {
	input := "byte b = 10;"
	tokens, err := Lex(input)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}

	// byte should now be an IDENTIFIER, not a keyword
	expected := []TokenType{
		IDENTIFIER, IDENTIFIER, ASSIGN, INTEGER, SEMICOLON, EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("Expected %d tokens, got %d", len(expected), len(tokens))
	}

	if tokens[0].Type != IDENTIFIER {
		t.Errorf("Expected 'byte' to be lexed as IDENTIFIER, got %s", tokens[0].Type)
	}
}
