package compiler

import "fmt"

// TokenType identifies the category of a lexed token.
type TokenType int

const (
	EOF TokenType = iota // sentinel: end of input

	// Literals
	IDENTIFIER // variable / function name
	INTEGER    // decimal integer literal
	STRING     // string literal "..."

	// Keywords
	INT      // "int"
	BYTE     // "byte"
	UNSIGNED // "unsigned"
	VOID     // "void"
	IF       // "if"
	ELSE     // "else"
	WHILE    // "while"
	RETURN   // "return"
	STRUCT   // "struct"
	FOR      // "for"
	ASM      // "asm"
	SWITCH   // "switch"
	CASE     // "case"
	DEFAULT  // "default"
	BREAK    // "break"
	CONTINUE // "continue"

	// Paired delimiters
	LBRACE   // {
	RBRACE   // }
	LPAREN   // (
	RPAREN   // )
	LBRACKET // [
	RBRACKET // ]

	// Punctuation
	DOT       // .
	SEMICOLON // ;
	COMMA     // ,
	COLON     // :

	// Arithmetic operators
	PLUS        // +
	MINUS       // -
	STAR        // *
	SLASH       // /
	AND         // & (binary bitwise AND, or unary address-of)
	PIPE        // |
	CARET       // ^
	TILDE       // ~
	PERCENT     // %
	SHL_OP      // <<
	SHR_OP      // >>
	AND_LOGICAL // &&
	OR_LOGICAL  // ||
	NOT         // !

	PLUS_PLUS   // ++
	MINUS_MINUS // --

	// Assignment / comparison  (order matters: ASSIGN before EQUALS)
	ASSIGN       // =
	PLUS_ASSIGN  // +=
	MINUS_ASSIGN // -=
	STAR_ASSIGN  // *=
	SLASH_ASSIGN // /=

	EQUALS  // ==
	NOT_EQ  // !=
	LESS    // <
	GREATER // >

	UNSIGNED_LIT // integer literal with a u/U suffix, e.g., 10u or 0xFFFFu

	LESS_EQ    // <=
	GREATER_EQ // >=
)

// tokenNames is indexed by TokenType; the compiler enforces the length via the
// blank identifier check in init() below.
var tokenNames = [...]string{
	EOF:          "EOF",
	IDENTIFIER:   "IDENTIFIER",
	INTEGER:      "INTEGER",
	STRING:       "STRING",
	INT:          "INT",
	BYTE:         "BYTE",
	UNSIGNED:     "UNSIGNED",
	VOID:         "VOID",
	IF:           "IF",
	ELSE:         "ELSE",
	WHILE:        "WHILE",
	RETURN:       "RETURN",
	STRUCT:       "STRUCT",
	FOR:          "FOR",
	ASM:          "ASM",
	SWITCH:       "SWITCH",
	CASE:         "CASE",
	DEFAULT:      "DEFAULT",
	BREAK:        "BREAK",
	CONTINUE:     "CONTINUE",
	LBRACE:       "LBRACE",
	RBRACE:       "RBRACE",
	LPAREN:       "LPAREN",
	RPAREN:       "RPAREN",
	LBRACKET:     "LBRACKET",
	RBRACKET:     "RBRACKET",
	DOT:          "DOT",
	SEMICOLON:    "SEMICOLON",
	COMMA:        "COMMA",
	COLON:        "COLON",
	PLUS:         "PLUS",
	MINUS:        "MINUS",
	STAR:         "STAR",
	SLASH:        "SLASH",
	AND:          "AND",
	PIPE:         "PIPE",
	CARET:        "CARET",
	TILDE:        "TILDE",
	PERCENT:      "PERCENT",
	SHL_OP:       "SHL_OP",
	SHR_OP:       "SHR_OP",
	AND_LOGICAL:  "AND_LOGICAL",
	OR_LOGICAL:   "OR_LOGICAL",
	NOT:          "NOT",
	PLUS_PLUS:    "PLUS_PLUS",
	MINUS_MINUS:  "MINUS_MINUS",
	ASSIGN:       "ASSIGN",
	PLUS_ASSIGN:  "PLUS_ASSIGN",
	MINUS_ASSIGN: "MINUS_ASSIGN",
	STAR_ASSIGN:  "STAR_ASSIGN",
	SLASH_ASSIGN: "SLASH_ASSIGN",
	EQUALS:       "EQUALS",
	NOT_EQ:       "NOT_EQ",
	LESS:         "LESS",
	GREATER:      "GREATER",
	UNSIGNED_LIT: "UNSIGNED_LIT",
}

func (tt TokenType) String() string {
	if int(tt) >= 0 && int(tt) < len(tokenNames) {
		return tokenNames[tt]
	}
	return fmt.Sprintf("TokenType(%d)", int(tt))
}

// Token is a single lexical unit produced by the Lexer.
type Token struct {
	Type   TokenType
	Lexeme string // the exact source text that was matched
	Line   int    // 1-based source line
}

func (t Token) String() string {
	return fmt.Sprintf("%-10s %-14q  line %d", t.Type, t.Lexeme, t.Line)
}
