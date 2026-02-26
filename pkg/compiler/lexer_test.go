package compiler

import (
	"reflect"
	"testing"
)

func TestLex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
		wantErr  bool
	}{
		{
			name:  "Empty",
			input: "",
			expected: []Token{
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "Basic Tokens",
			input: "+ - * / & = == != < > ; , { } ( )",
			expected: []Token{
				{Type: PLUS, Lexeme: "+", Line: 1},
				{Type: MINUS, Lexeme: "-", Line: 1},
				{Type: STAR, Lexeme: "*", Line: 1},
				{Type: SLASH, Lexeme: "/", Line: 1},
				{Type: AND, Lexeme: "&", Line: 1},
				{Type: ASSIGN, Lexeme: "=", Line: 1},
				{Type: EQUALS, Lexeme: "==", Line: 1},
				{Type: NOT_EQ, Lexeme: "!=", Line: 1},
				{Type: LESS, Lexeme: "<", Line: 1},
				{Type: GREATER, Lexeme: ">", Line: 1},
				{Type: SEMICOLON, Lexeme: ";", Line: 1},
				{Type: COMMA, Lexeme: ",", Line: 1},
				{Type: LBRACE, Lexeme: "{", Line: 1},
				{Type: RBRACE, Lexeme: "}", Line: 1},
				{Type: LPAREN, Lexeme: "(", Line: 1},
				{Type: RPAREN, Lexeme: ")", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "Keywords and Identifiers",
			input: "int if else while return variableName _under_score",
			expected: []Token{
				{Type: INT, Lexeme: "int", Line: 1},
				{Type: IF, Lexeme: "if", Line: 1},
				{Type: ELSE, Lexeme: "else", Line: 1},
				{Type: WHILE, Lexeme: "while", Line: 1},
				{Type: RETURN, Lexeme: "return", Line: 1},
				{Type: IDENTIFIER, Lexeme: "variableName", Line: 1},
				{Type: IDENTIFIER, Lexeme: "_under_score", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "Integers",
			input: "123 0 0x1A 0Xff",
			expected: []Token{
				{Type: INTEGER, Lexeme: "123", Line: 1},
				{Type: INTEGER, Lexeme: "0", Line: 1},
				{Type: INTEGER, Lexeme: "0x1A", Line: 1},
				{Type: INTEGER, Lexeme: "0Xff", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "Equality",
			input: "a == b",
			expected: []Token{
				{Type: IDENTIFIER, Lexeme: "a", Line: 1},
				{Type: EQUALS, Lexeme: "==", Line: 1},
				{Type: IDENTIFIER, Lexeme: "b", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "Comments",
			input: "x // comment\n y /* block */ z",
			expected: []Token{
				{Type: IDENTIFIER, Lexeme: "x", Line: 1},
				{Type: IDENTIFIER, Lexeme: "y", Line: 2},
				{Type: IDENTIFIER, Lexeme: "z", Line: 2},
				{Type: EOF, Lexeme: "", Line: 2},
			},
		},
		{
			name:    "Unterminated Block Comment",
			input:   "/* start",
			wantErr: true,
		},
		{
			name:    "Unexpected Character",
			input:   "@",
			wantErr: true,
		},
		{
			name:  "Bitwise and Shift Operators",
			input: "| ^ ~ % << >>",
			expected: []Token{
				{Type: PIPE, Lexeme: "|", Line: 1},
				{Type: CARET, Lexeme: "^", Line: 1},
				{Type: TILDE, Lexeme: "~", Line: 1},
				{Type: PERCENT, Lexeme: "%", Line: 1},
				{Type: SHL_OP, Lexeme: "<<", Line: 1},
				{Type: SHR_OP, Lexeme: ">>", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "Single Less and Greater not confused with shifts",
			input: "a < b > c",
			expected: []Token{
				{Type: IDENTIFIER, Lexeme: "a", Line: 1},
				{Type: LESS, Lexeme: "<", Line: 1},
				{Type: IDENTIFIER, Lexeme: "b", Line: 1},
				{Type: GREATER, Lexeme: ">", Line: 1},
				{Type: IDENTIFIER, Lexeme: "c", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "Logical Operators",
			input: "&& || !",
			expected: []Token{
				{Type: AND_LOGICAL, Lexeme: "&&", Line: 1},
				{Type: OR_LOGICAL, Lexeme: "||", Line: 1},
				{Type: NOT, Lexeme: "!", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "String Literal",
			input: "\"hello\"",
			expected: []Token{
				{Type: STRING, Lexeme: "hello", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "String with Escapes",
			input: "\"a\\nb\"",
			expected: []Token{
				{Type: STRING, Lexeme: "a\nb", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:    "Unterminated String",
			input:   "\"hello",
			wantErr: true,
		},
		{
			name:  "Keywords: void struct",
			input: "void struct",
			expected: []Token{
				{Type: VOID, Lexeme: "void", Line: 1},
				{Type: STRUCT, Lexeme: "struct", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "Punctuation: . [ ]",
			input: ". [ ]",
			expected: []Token{
				{Type: DOT, Lexeme: ".", Line: 1},
				{Type: LBRACKET, Lexeme: "[", Line: 1},
				{Type: RBRACKET, Lexeme: "]", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "Hex No Digits (Edge Case)",
			input: "0x",
			expected: []Token{
				{Type: INTEGER, Lexeme: "0x", Line: 1}, // Verify current behavior
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
		{
			name:  "Adjacent Tokens",
			input: "x+y",
			expected: []Token{
				{Type: IDENTIFIER, Lexeme: "x", Line: 1},
				{Type: PLUS, Lexeme: "+", Line: 1},
				{Type: IDENTIFIER, Lexeme: "y", Line: 1},
				{Type: EOF, Lexeme: "", Line: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Lex(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("Lex() = %v, want %v", got, tt.expected)
				}
			}
		})
	}
}
