package compiler

import (
	"fmt"
	"unicode"
)

// keywords maps source text to its keyword TokenType.
var keywords = map[string]TokenType{
	"int":      INT,
	"char":     CHAR,
	"unsigned": UNSIGNED,
	"void":     VOID,
	"if":       IF,
	"else":     ELSE,
	"while":    WHILE,
	"return":   RETURN,
	"struct":   STRUCT,
	"for":      FOR,
	"asm":      ASM,
	"switch":   SWITCH,
	"case":     CASE,
	"default":  DEFAULT,
	"break":    BREAK,
	"continue": CONTINUE,
	"volatile": VOLATILE,
	"const":    CONST,
	"static":   STATIC,
	"extern":   EXTERN,
}

// Lexer holds all mutable state for a single scanning pass over src.
type Lexer struct {
	src  []rune
	pos  int // index of the next rune to consume
	line int // current 1-based source line
}

func newLexer(src string) *Lexer {
	return &Lexer{src: []rune(src), pos: 0, line: 1}
}

// peek returns the rune at the current position without advancing.
func (l *Lexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

// peek2 returns the rune one position ahead of the current position.
func (l *Lexer) peek2() rune {
	if l.pos+1 >= len(l.src) {
		return 0
	}
	return l.src[l.pos+1]
}

// advance consumes one rune and returns it.
func (l *Lexer) advance() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	r := l.src[l.pos]
	l.pos++
	if r == '\n' {
		l.line++
	}
	return r
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.src) && unicode.IsSpace(l.peek()) {
		l.advance()
	}
}

// skipLineComment discards everything from the current position to end-of-line.
// The opening "//" must already have been consumed.
func (l *Lexer) skipLineComment() {
	for l.pos < len(l.src) && l.peek() != '\n' {
		l.advance()
	}
}

// skipBlockComment discards everything up to and including the closing "*/".
// The opening "/*" must already have been consumed.
func (l *Lexer) skipBlockComment() error {
	startLine := l.line
	for l.pos < len(l.src) {
		if l.peek() == '*' && l.peek2() == '/' {
			l.advance() // *
			l.advance() // /
			return nil
		}
		l.advance()
	}
	return fmt.Errorf("unterminated block comment (opened on line %d)", startLine)
}

// scanIdent collects a full identifier or keyword token.
// The first character (letter or '_') must still be at l.peek().
func (l *Lexer) scanIdent() Token {
	line := l.line
	start := l.pos
	for l.pos < len(l.src) {
		r := l.peek()
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			break
		}
		l.advance()
	}
	lexeme := string(l.src[start:l.pos])
	tt := IDENTIFIER
	if kw, ok := keywords[lexeme]; ok {
		tt = kw
	}
	return Token{Type: tt, Lexeme: lexeme, Line: line}
}

// scanInt collects a decimal or hex integer literal, including an optional
// u/U suffix that marks the literal as unsigned (e.g. 10u, 0xFFFFu).
// The first digit must still be at l.peek().
func (l *Lexer) scanInt() Token {
	line := l.line
	start := l.pos

	// Check for '0x' or '0X' prefix
	if l.peek() == '0' && (l.peek2() == 'x' || l.peek2() == 'X') {
		l.advance() // consume '0'
		l.advance() // consume 'x'
		// Consume hex digits
		for l.pos < len(l.src) {
			r := l.peek()
			if unicode.IsDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
				l.advance()
			} else {
				break
			}
		}
	} else {
		// Normal decimal digits
		for l.pos < len(l.src) && unicode.IsDigit(l.peek()) {
			l.advance()
		}
	}

	// Check for optional u/U suffix marking an unsigned literal.
	numEnd := l.pos
	if l.pos < len(l.src) && (l.peek() == 'u' || l.peek() == 'U') {
		l.advance() // consume the suffix
		return Token{Type: UNSIGNED_LIT, Lexeme: string(l.src[start:numEnd]), Line: line}
	}

	return Token{Type: INTEGER, Lexeme: string(l.src[start:l.pos]), Line: line}
}

// scanChar collects a character literal 'c'
func (l *Lexer) scanChar() (Token, error) {
	line := l.line
	l.advance() // consume opening '

	r := l.peek()
	var val rune

	if r == '\'' {
		return Token{}, fmt.Errorf("empty character literal on line %d", line)
	}

	if r == '\\' {
		l.advance() // consume backslash
		next := l.peek()
		switch next {
		case 'n':
			val = '\n'
		case 'r':
			val = '\r'
		case 't':
			val = '\t'
		case '0':
			val = 0
		case '\\':
			val = '\\'
		case '\'':
			val = '\''
		case '"':
			val = '"'
		default:
			return Token{}, fmt.Errorf("unknown escape sequence \\%c on line %d", next, line)
		}
		l.advance()
	} else {
		val = r
		l.advance()
	}

	if l.peek() != '\'' {
		return Token{}, fmt.Errorf("unterminated character literal on line %d", line)
	}
	l.advance() // consume closing '

	// Character literals are emitted as INTEGER tokens with their ASCII value
	return Token{Type: INTEGER, Lexeme: fmt.Sprintf("%d", val), Line: line}, nil
}

// scanString collects a string literal "..."
func (l *Lexer) scanString() (Token, error) {
	line := l.line
	l.advance() // consume opening "
	var val []rune

	for l.pos < len(l.src) {
		r := l.peek()
		if r == '"' {
			break
		}
		if r == '\n' {
			return Token{}, fmt.Errorf("unterminated string literal on line %d", line)
		}
		if r == '\\' {
			l.advance() // consume backslash
			next := l.peek()
			switch next {
			case 'n':
				val = append(val, '\n')
			case 't':
				val = append(val, '\t')
			case '"':
				val = append(val, '"')
			case '\\':
				val = append(val, '\\')
			default:
				return Token{}, fmt.Errorf("unknown escape sequence \\%c on line %d", next, line)
			}
			l.advance()
			continue
		}
		val = append(val, r)
		l.advance()
	}

	if l.pos >= len(l.src) {
		return Token{}, fmt.Errorf("unterminated string literal on line %d", line)
	}
	l.advance() // consume closing "

	return Token{Type: STRING, Lexeme: string(val), Line: line}, nil
}

// nextToken skips whitespace/comments and returns the next Token.
func (l *Lexer) nextToken() (Token, error) {
	// Skip whitespace and both comment styles in a loop so that
	// a comment followed immediately by more whitespace is handled.
	for {
		l.skipWhitespace()
		if l.pos >= len(l.src) {
			return Token{Type: EOF, Lexeme: "", Line: l.line}, nil
		}
		if l.peek() == '/' && l.peek2() == '/' {
			l.advance()
			l.advance()
			l.skipLineComment()
			continue
		}
		if l.peek() == '/' && l.peek2() == '*' {
			l.advance()
			l.advance()
			if err := l.skipBlockComment(); err != nil {
				return Token{}, err
			}
			continue
		}
		break
	}

	ch := l.peek()
	line := l.line

	if unicode.IsLetter(ch) || ch == '_' {
		return l.scanIdent(), nil
	}
	if unicode.IsDigit(ch) {
		return l.scanInt(), nil
	}

	if ch == '"' {
		return l.scanString()
	}

	if ch == '\'' {
		return l.scanChar()
	}

	l.advance() // consume the character before the switch
	switch ch {
	case '{':
		return Token{LBRACE, "{", line}, nil
	case '}':
		return Token{RBRACE, "}", line}, nil
	case '(':
		return Token{LPAREN, "(", line}, nil
	case ')':
		return Token{RPAREN, ")", line}, nil
	case '[':
		return Token{LBRACKET, "[", line}, nil
	case ']':
		return Token{RBRACKET, "]", line}, nil
	case '.':
		return Token{DOT, ".", line}, nil
	case ';':
		return Token{SEMICOLON, ";", line}, nil
	case ',':
		return Token{COMMA, ",", line}, nil
	case ':':
		return Token{COLON, ":", line}, nil

	case '+':
		if l.peek() == '+' {
			l.advance()
			return Token{PLUS_PLUS, "++", line}, nil
		}
		if l.peek() == '=' {
			l.advance()
			return Token{PLUS_ASSIGN, "+=", line}, nil
		}
		return Token{PLUS, "+", line}, nil
	case '-':
		if l.peek() == '-' {
			l.advance()
			return Token{MINUS_MINUS, "--", line}, nil
		}
		if l.peek() == '=' {
			l.advance()
			return Token{MINUS_ASSIGN, "-=", line}, nil
		}
		return Token{MINUS, "-", line}, nil
	case '*':
		if l.peek() == '=' {
			l.advance()
			return Token{STAR_ASSIGN, "*=", line}, nil
		}
		return Token{STAR, "*", line}, nil
	case '/':
		if l.peek() == '=' {
			l.advance()
			return Token{SLASH_ASSIGN, "/=", line}, nil
		}
		return Token{SLASH, "/", line}, nil
	case '&':
		if l.peek() == '&' {
			l.advance()
			return Token{AND_LOGICAL, "&&", line}, nil
		}
		return Token{AND, "&", line}, nil
	case '|':
		if l.peek() == '|' {
			l.advance()
			return Token{OR_LOGICAL, "||", line}, nil
		}
		return Token{PIPE, "|", line}, nil
	case '^':
		return Token{CARET, "^", line}, nil
	case '~':
		return Token{TILDE, "~", line}, nil
	case '%':
		return Token{PERCENT, "%", line}, nil
	case '!':
		if l.peek() == '=' {
			l.advance()
			return Token{NOT_EQ, "!=", line}, nil
		}
		return Token{NOT, "!", line}, nil
	case '<':
		if l.peek() == '=' {
			l.advance()
			return Token{LESS_EQ, "<=", line}, nil
		}
		if l.peek() == '<' {
			l.advance()
			return Token{SHL_OP, "<<", line}, nil
		}
		return Token{LESS, "<", line}, nil
	case '>':
		if l.peek() == '=' {
			l.advance()
			return Token{GREATER_EQ, ">=", line}, nil
		}
		if l.peek() == '>' {
			l.advance()
			return Token{SHR_OP, ">>", line}, nil
		}
		return Token{GREATER, ">", line}, nil
	case '=':
		if l.peek() == '=' { // lookahead: distinguish = vs ==
			l.advance()
			return Token{EQUALS, "==", line}, nil
		}
		return Token{ASSIGN, "=", line}, nil
	default:
		return Token{}, fmt.Errorf("unexpected character %q on line %d", ch, line)
	}
}

// Lex tokenises src and returns all tokens including the final EOF token.
// It returns a non-nil error on the first illegal character or unterminated comment.
func Lex(src string) ([]Token, error) {
	l := newLexer(src)
	var tokens []Token
	for {
		tok, err := l.nextToken()
		if err != nil {
			return tokens, err
		}
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			return tokens, nil
		}
	}
}
