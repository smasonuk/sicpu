package compiler

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser consumes the flat token slice produced by the Lexer and builds an AST.
//
// Grammar (Phase 2 subset + Functions + Pointers + Structs + Arrays):
//
//				program    = (functionDecl | statement)* EOF
//				statement  = varDecl | structDecl | assignment | returnStmt | block | if | while | exprStmt
//				varDecl    = ("int" | "struct" IDENTIFIER) ("*")? IDENTIFIER ("[" INTEGER "]")? ("=" expression)? ";"
//				structDecl = "struct" IDENTIFIER "{" (varDecl)* "}" ";"
//				assignment = lvalue "=" expression ";"
//				returnStmt = "return" expression ";"
//				expression = logical_or
//	     logical_or = logical_and ("||" logical_and)*
//	     logical_and = bitwise_or ("&&" bitwise_or)*
//	     bitwise_or = bitwise_xor ("|" bitwise_xor)*
//	     bitwise_xor = bitwise_and ("^" bitwise_and)*
//	     bitwise_and = equality ("&" equality)*
//			 equality   = relational (("=="|"!=") relational)*
//	     relational = shift (("<"|">") shift)*
//	     shift      = additive (("<<"|">>") additive)*
//			 additive   = multiplicative (("+" | "-") multiplicative)*
//	     multiplicative = unary (("*" | "/" | "%") unary)*
//			 unary      = ("&" | "*" | "~" | "!") unary | postfix
//		     postfix    = primary ("[" expression "]" | "." IDENTIFIER | "(" args ")")*
//				primary    = INTEGER | IDENTIFIER | "(" expression ")"
//			 lvalue     = expression (must result in addressable location)
type Parser struct {
	tokens         []Token
	pos            int
	currentRetType TokenType
	sourceLines    []string
}

func NewParser(tokens []Token, rawSource string) *Parser {
	return &Parser{tokens: tokens, sourceLines: strings.Split(rawSource, "\n")}
}

// fmtError wraps an error message with the source line where the token appears.
func (p *Parser) fmtError(tok Token, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	lineIdx := tok.Line - 1 // Lines are 1-based

	snippet := "<source unavailable>"
	if lineIdx >= 0 && lineIdx < len(p.sourceLines) {
		snippet = strings.TrimSpace(p.sourceLines[lineIdx])
	}

	return fmt.Errorf("line %d: %s\n  |> %s", tok.Line, msg, snippet)
}

// peek returns the current token without consuming it.
func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: EOF}
	}
	return p.tokens[p.pos]
}

// peekNext returns the token immediately after the current one.
func (p *Parser) peekNext() Token {
	return p.peekAt(1)
}

// peekAt returns the token at the given offset from the current position.
func (p *Parser) peekAt(offset int) Token {
	if p.pos+offset >= len(p.tokens) {
		return Token{Type: EOF}
	}
	return p.tokens[p.pos+offset]
}

// advance consumes and returns the current token.
func (p *Parser) advance() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

// expect consumes the current token if it matches tt, otherwise returns an error.
func (p *Parser) expect(tt TokenType) (Token, error) {
	tok := p.advance()
	if tok.Type != tt {
		return tok, p.fmtError(tok, "expected %s, got %s (%q)", tt, tok.Type, tok.Lexeme)
	}
	return tok, nil
}

// parseExpression is the entry point for expression parsing.
func (p *Parser) parseExpression() (Expr, error) {
	return p.parseLogicalOr()
}

// parseLogicalOr handles ||
func (p *Parser) parseLogicalOr() (Expr, error) {
	expr, err := p.parseLogicalAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == OR_LOGICAL {
		op := p.advance().Type
		right, err := p.parseLogicalAnd()
		if err != nil {
			return nil, err
		}
		expr = &LogicalExpr{Op: op, Left: expr, Right: right}
	}
	return expr, nil
}

// parseLogicalAnd handles &&
func (p *Parser) parseLogicalAnd() (Expr, error) {
	expr, err := p.parseBitwiseOr()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == AND_LOGICAL {
		op := p.advance().Type
		right, err := p.parseBitwiseOr()
		if err != nil {
			return nil, err
		}
		expr = &LogicalExpr{Op: op, Left: expr, Right: right}
	}
	return expr, nil
}

// parseBitwiseOr handles | (lowest precedence among bitwise ops)
func (p *Parser) parseBitwiseOr() (Expr, error) {
	expr, err := p.parseBitwiseXor()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == PIPE {
		op := p.advance().Type
		right, err := p.parseBitwiseXor()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Op: op, Left: expr, Right: right}
	}
	return expr, nil
}

// parseBitwiseXor handles ^
func (p *Parser) parseBitwiseXor() (Expr, error) {
	expr, err := p.parseBitwiseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == CARET {
		op := p.advance().Type
		right, err := p.parseBitwiseAnd()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Op: op, Left: expr, Right: right}
	}
	return expr, nil
}

// parseBitwiseAnd handles binary &
// Unary & (address-of) is handled in parseUnary and is never seen here.
func (p *Parser) parseBitwiseAnd() (Expr, error) {
	expr, err := p.parseEquality()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == AND {
		op := p.advance().Type
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Op: op, Left: expr, Right: right}
	}
	return expr, nil
}

// parseEquality handles == and !=
func (p *Parser) parseEquality() (Expr, error) {
	expr, err := p.parseRelational()
	if err != nil {
		return nil, err
	}

	for p.peek().Type == EQUALS || p.peek().Type == NOT_EQ {
		op := p.advance().Type
		right, err := p.parseRelational()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Op: op, Left: expr, Right: right}
	}

	return expr, nil
}

// parseRelational handles < and >
func (p *Parser) parseRelational() (Expr, error) {
	expr, err := p.parseShift()
	if err != nil {
		return nil, err
	}

	for p.peek().Type == LESS || p.peek().Type == GREATER ||
		p.peek().Type == LESS_EQ || p.peek().Type == GREATER_EQ {
		op := p.advance().Type
		right, err := p.parseShift()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Op: op, Left: expr, Right: right}
	}

	return expr, nil
}

// parseShift handles << and >>
func (p *Parser) parseShift() (Expr, error) {
	expr, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == SHL_OP || p.peek().Type == SHR_OP {
		op := p.advance().Type
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Op: op, Left: expr, Right: right}
	}
	return expr, nil
}

// parseAdditive handles + and -
func (p *Parser) parseAdditive() (Expr, error) {
	expr, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}

	for {
		tt := p.peek().Type
		if tt != PLUS && tt != MINUS {
			break
		}
		op := p.advance().Type
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Op: op, Left: expr, Right: right}
	}

	return expr, nil
}

// parseMultiplicative handles * and /
func (p *Parser) parseMultiplicative() (Expr, error) {
	expr, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		tt := p.peek().Type
		if tt != STAR && tt != SLASH && tt != PERCENT {
			break
		}
		op := p.advance().Type
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		expr = &BinaryExpr{Op: op, Left: expr, Right: right}
	}

	return expr, nil
}

// parseUnary handles prefix operators &, *, ~, !, and - (unary minus)
func (p *Parser) parseUnary() (Expr, error) {
	// Handle Casts: (int), (byte), (int*), or (byte*)
	if p.peek().Type == LPAREN {
		nextTok := p.peekAt(1).Type
		if nextTok == INT || nextTok == BYTE {
			// Check if it's a pointer cast
			isPtr := p.peekAt(2).Type == STAR

			// Determine where the closing parenthesis should be
			rparenOffset := 2
			if isPtr {
				rparenOffset = 3
			}

			if p.peekAt(rparenOffset).Type == RPAREN {
				p.advance()            // consume '('
				typeTok := p.advance() // consume 'int' or 'byte'
				if isPtr {
					p.advance() // consume '*'
				}
				p.advance() // consume ')'

				right, err := p.parseUnary()
				if err != nil {
					return nil, err
				}
				return &CastExpr{Type: typeTok.Type, IsPointer: isPtr, Expr: right}, nil
			}
		}
	}

	if p.peek().Type == AND || p.peek().Type == STAR || p.peek().Type == TILDE || p.peek().Type == NOT || p.peek().Type == MINUS {
		op := p.advance().Type
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: op, Right: right}, nil
	}
	return p.parsePostfix()
}

// parsePostfix handles array index [], struct access ., and function calls ()
func (p *Parser) parsePostfix() (Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		if p.peek().Type == LBRACKET {
			var indices []Expr
			for p.peek().Type == LBRACKET {
				p.advance() // [
				index, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				if _, err := p.expect(RBRACKET); err != nil {
					return nil, err
				}
				indices = append(indices, index)
			}
			expr = &IndexExpr{Left: expr, Indices: indices}
		} else if p.peek().Type == DOT {
			p.advance() // .
			memberTok, err := p.expect(IDENTIFIER)
			if err != nil {
				return nil, err
			}
			expr = &MemberExpr{Left: expr, Member: memberTok.Lexeme}
		} else if p.peek().Type == LPAREN {
			// Function call conversion
			if varRef, ok := expr.(*VarRef); ok {
				p.advance() // (
				args, err := p.parseCallArgs()
				if err != nil {
					return nil, err
				}
				expr = &FunctionCall{Name: varRef.Name, Args: args}
			} else {
				// We don't support computed function calls like (ptr)(args) yet
				return nil, fmt.Errorf("line %d: expected function name before '('", p.peek().Line)
			}
		} else if p.peek().Type == PLUS_PLUS || p.peek().Type == MINUS_MINUS {
			op := p.advance().Type
			expr = &PostfixExpr{Left: expr, Op: op}
		} else {
			break
		}
	}
	return expr, nil
}

func (p *Parser) parseCallArgs() ([]Expr, error) {
	var args []Expr
	if p.peek().Type != RPAREN {
		for {
			arg, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if p.peek().Type != COMMA {
				break
			}
			p.advance()
		}
	}

	if _, err := p.expect(RPAREN); err != nil {
		return nil, err
	}
	return args, nil
}

// parsePrimary handles literals, variables, and parenthesised expressions.
func (p *Parser) parsePrimary() (Expr, error) {
	tok := p.peek()
	switch tok.Type {
	case INTEGER:
		p.advance()
		val, err := strconv.ParseUint(tok.Lexeme, 0, 16)
		if err != nil {
			return nil, fmt.Errorf("line %d: integer %q out of 16-bit range", tok.Line, tok.Lexeme)
		}
		return &Literal{Value: uint16(val)}, nil

	case UNSIGNED_LIT:
		p.advance()
		val, err := strconv.ParseUint(tok.Lexeme, 0, 16)
		if err != nil {
			return nil, fmt.Errorf("line %d: unsigned integer %q out of 16-bit range", tok.Line, tok.Lexeme)
		}
		return &Literal{Value: uint16(val), IsUnsigned: true}, nil

	case STRING:
		p.advance()
		return &StringLiteral{Value: tok.Lexeme}, nil

	case IDENTIFIER:
		p.advance()
		return &VarRef{Name: tok.Lexeme}, nil

	case LPAREN:
		p.advance()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(RPAREN); err != nil {
			return nil, err
		}
		return expr, nil

	default:
		return nil, p.fmtError(tok, "expected expression, got %s (%q)", tok.Type, tok.Lexeme)
	}
}

func (p *Parser) parseInitializerList() (*InitializerList, error) {
	if _, err := p.expect(LBRACE); err != nil {
		return nil, err
	}

	var elements []Expr
	if p.peek().Type != RBRACE {
		for {
			expr, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			elements = append(elements, expr)

			if p.peek().Type == COMMA {
				p.advance()
			} else {
				break
			}
		}
	}

	if _, err := p.expect(RBRACE); err != nil {
		return nil, err
	}

	return &InitializerList{Elements: elements}, nil
}

// parseVarDeclInternal handles variable declarations.
// isField: true if parsing struct fields (no initializers allowed).
func (p *Parser) parseVarDeclInternal(isField bool) (*VariableDecl, error) {
	var decl VariableDecl

	// Parse type
	if p.peek().Type == UNSIGNED {
		p.advance()
		decl.IsUnsigned = true
		if p.peek().Type == INT {
			p.advance()
		}
		// Optional *
		if p.peek().Type == STAR {
			p.advance()
			decl.IsPointer = true
		}
	} else if p.peek().Type == INT {
		p.advance()
		// Optional *
		if p.peek().Type == STAR {
			p.advance()
			decl.IsPointer = true
		}
	} else if p.peek().Type == BYTE {
		p.advance()
		decl.IsByte = true
		// Optional *
		if p.peek().Type == STAR {
			p.advance()
			decl.IsPointer = true
		}
	} else if p.peek().Type == STRUCT {
		p.advance()
		decl.IsStruct = true
		nameTok, err := p.expect(IDENTIFIER)
		if err != nil {
			return nil, err
		}
		decl.StructName = nameTok.Lexeme
		// Optional *
		if p.peek().Type == STAR {
			p.advance()
			// Pointer to struct. Treat as scalar int (pointer).
			decl.IsStruct = false
			decl.StructName = "" // clear it to treat as scalar
			decl.IsPointer = true
		}
	} else {
		return nil, fmt.Errorf("line %d: expected type (int, byte, or struct)", p.peek().Line)
	}

	nameTok, err := p.expect(IDENTIFIER)
	if err != nil {
		return nil, err
	}
	decl.Name = nameTok.Lexeme

	// Check for array: [size]...
	for p.peek().Type == LBRACKET {
		p.advance()

		var size int
		if p.peek().Type == RBRACKET {
			// Empty size [], allowed if initializer is present (checked later)
			size = 0
		} else {
			sizeTok, err := p.expect(INTEGER)
			if err != nil {
				return nil, err
			}
			size, _ = strconv.Atoi(sizeTok.Lexeme)
		}

		decl.IsArray = true
		decl.ArraySizes = append(decl.ArraySizes, size)
		if _, err := p.expect(RBRACKET); err != nil {
			return nil, err
		}
	}

	if isField {
		if _, err := p.expect(SEMICOLON); err != nil {
			return nil, err
		}
		return &decl, nil
	}

	// For variables, optional initializer
	if p.peek().Type == ASSIGN {
		p.advance()

		if p.peek().Type == LBRACE {
			initList, err := p.parseInitializerList()
			if err != nil {
				return nil, err
			}
			decl.Init = initList

			// Array Size Inference
			// int arr[] = {1, 2, 3}; -> int arr[3] = ...
			if decl.IsArray && len(decl.ArraySizes) > 0 && decl.ArraySizes[0] == 0 {
				decl.ArraySizes[0] = len(initList.Elements)
			}

		} else {
			if decl.IsArray || decl.IsStruct {
				return nil, fmt.Errorf("line %d: array/struct initialization requires '{...}'", nameTok.Line)
			}
			init, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			decl.Init = init
		}
	}

	if _, err := p.expect(SEMICOLON); err != nil {
		return nil, err
	}
	return &decl, nil
}

func (p *Parser) parseVarDecl() (Stmt, error) {
	return p.parseVarDeclInternal(false)
}

func (p *Parser) parseStructDecl() (Stmt, error) {
	// "struct" consumed by caller? No, caller calls this when it sees STRUCT but doesn't consume it?
	// parseStatement checks peek. If STRUCT, calls parseStructDecl.
	// So we need to consume STRUCT here.
	if _, err := p.expect(STRUCT); err != nil {
		return nil, err
	}

	nameTok, err := p.expect(IDENTIFIER)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(LBRACE); err != nil {
		return nil, err
	}

	var fields []VariableDecl
	for p.peek().Type != RBRACE && p.peek().Type != EOF {
		// Field declaration: type name;
		field, err := p.parseVarDeclInternal(true) // true = isField
		if err != nil {
			return nil, err
		}
		fields = append(fields, *field)
	}

	if _, err := p.expect(RBRACE); err != nil {
		return nil, err
	}
	if _, err := p.expect(SEMICOLON); err != nil {
		return nil, err
	}

	return &StructDecl{Name: nameTok.Lexeme, Fields: fields}, nil
}

// parseAssignment parses  lvalue = expr ;
// The left-hand side expression (lvalue) is passed in.
func (p *Parser) parseAssignment(left Expr) (Stmt, error) {
	op := p.advance().Type
	// We expect ASSIGN or Compound Assignment
	if op != ASSIGN && op != PLUS_ASSIGN && op != MINUS_ASSIGN && op != STAR_ASSIGN && op != SLASH_ASSIGN {
		return nil, fmt.Errorf("line %d: expected assignment operator, got %s", p.peek().Line, op)
	}

	val, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(SEMICOLON); err != nil {
		return nil, err
	}
	return &Assignment{Left: left, Op: op, Value: val}, nil
}

// parseReturn parses  return expr ;
// The leading RETURN token has already been consumed by parseStatement.
func (p *Parser) parseReturn() (Stmt, error) {
	if p.currentRetType == VOID {
		if p.peek().Type == SEMICOLON {
			p.advance()
			return &ReturnStmt{Expr: nil}, nil
		}
		return nil, fmt.Errorf("line %d: void function cannot return a value", p.peek().Line)
	}

	if p.peek().Type == SEMICOLON {
		return nil, fmt.Errorf("line %d: non-void function must return a value", p.peek().Line)
	}

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(SEMICOLON); err != nil {
		return nil, err
	}
	return &ReturnStmt{Expr: expr}, nil
}

// parseBlock parses { stmt1; stmt2; ... }
// The leading LBRACE token has already been consumed by parseStatement.
func (p *Parser) parseBlock() (Stmt, error) {
	var stmts []Stmt
	for p.peek().Type != RBRACE && p.peek().Type != EOF {
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}
	if _, err := p.expect(RBRACE); err != nil {
		return nil, err
	}
	return &BlockStmt{Stmts: stmts}, nil
}

// parseIf parses if ( cond ) body [ else elseBody ]
// The leading IF token has already been consumed by parseStatement.
func (p *Parser) parseIf() (Stmt, error) {
	if _, err := p.expect(LPAREN); err != nil {
		return nil, err
	}
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(RPAREN); err != nil {
		return nil, err
	}
	body, err := p.parseStatement()
	if err != nil {
		return nil, err
	}

	var elseBody Stmt
	if p.peek().Type == ELSE {
		p.advance()
		elseBody, err = p.parseStatement()
		if err != nil {
			return nil, err
		}
	}

	return &IfStmt{Condition: cond, Body: body, ElseBody: elseBody}, nil
}

// parseWhile parses while ( cond ) body
// The leading WHILE token has already been consumed by parseStatement.
func (p *Parser) parseWhile() (Stmt, error) {
	if _, err := p.expect(LPAREN); err != nil {
		return nil, err
	}
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(RPAREN); err != nil {
		return nil, err
	}
	body, err := p.parseStatement()
	if err != nil {
		return nil, err
	}
	return &WhileStmt{Condition: cond, Body: body}, nil
}

// parseSwitchStmt parses switch ( expr ) { case val: ... default: ... }
func (p *Parser) parseSwitchStmt() (Stmt, error) {
	if _, err := p.expect(SWITCH); err != nil {
		return nil, err
	}
	if _, err := p.expect(LPAREN); err != nil {
		return nil, err
	}
	target, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(RPAREN); err != nil {
		return nil, err
	}
	if _, err := p.expect(LBRACE); err != nil {
		return nil, err
	}

	var cases []CaseClause
	var defaultBody []Stmt
	hasDefault := false

	for p.peek().Type != RBRACE && p.peek().Type != EOF {
		if p.peek().Type == CASE {
			p.advance()
			val, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(COLON); err != nil {
				return nil, err
			}

			var body []Stmt
			for p.peek().Type != CASE && p.peek().Type != DEFAULT && p.peek().Type != RBRACE && p.peek().Type != EOF {
				stmt, err := p.parseStatement()
				if err != nil {
					return nil, err
				}
				if stmt != nil {
					body = append(body, stmt)
				}
			}
			cases = append(cases, CaseClause{Value: val, Body: body})

		} else if p.peek().Type == DEFAULT {
			if hasDefault {
				return nil, fmt.Errorf("line %d: multiple default labels in switch", p.peek().Line)
			}
			p.advance()
			if _, err := p.expect(COLON); err != nil {
				return nil, err
			}
			hasDefault = true

			for p.peek().Type != CASE && p.peek().Type != DEFAULT && p.peek().Type != RBRACE && p.peek().Type != EOF {
				stmt, err := p.parseStatement()
				if err != nil {
					return nil, err
				}
				if stmt != nil {
					defaultBody = append(defaultBody, stmt)
				}
			}
		} else {
			return nil, fmt.Errorf("line %d: expected case or default in switch, got %s", p.peek().Line, p.peek().Type)
		}
	}

	if _, err := p.expect(RBRACE); err != nil {
		return nil, err
	}
	return &SwitchStmt{Target: target, Cases: cases, Default: defaultBody}, nil
}

// parseForStmt parses for ( init; cond; post ) body
func (p *Parser) parseForStmt() (Stmt, error) {
	if _, err := p.expect(FOR); err != nil {
		return nil, err
	}
	if _, err := p.expect(LPAREN); err != nil {
		return nil, err
	}

	var init Stmt
	if p.peek().Type != SEMICOLON {
		if p.peek().Type == INT || p.peek().Type == BYTE || p.peek().Type == UNSIGNED {
			var err error
			init, err = p.parseVarDecl()
			if err != nil {
				return nil, err
			}
		} else {
			expr, err := p.parseExpression()
			if err != nil {
				return nil, err
			}

			op := p.peek().Type
			if op == ASSIGN || op == PLUS_ASSIGN || op == MINUS_ASSIGN || op == STAR_ASSIGN || op == SLASH_ASSIGN {
				init, err = p.parseAssignment(expr)
				if err != nil {
					return nil, err
				}
			} else {
				if _, err := p.expect(SEMICOLON); err != nil {
					return nil, err
				}
				init = &ExprStmt{Expr: expr}
			}
		}
	} else {
		p.advance() // consume ;
	}

	var cond Expr
	if p.peek().Type != SEMICOLON {
		var err error
		cond, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	}
	if _, err := p.expect(SEMICOLON); err != nil {
		return nil, err
	}

	var post Stmt
	if p.peek().Type != RPAREN {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		op := p.peek().Type
		if op == ASSIGN || op == PLUS_ASSIGN || op == MINUS_ASSIGN || op == STAR_ASSIGN || op == SLASH_ASSIGN {
			p.advance() // consume op
			val, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			post = &Assignment{Left: expr, Op: op, Value: val}
		} else {
			post = &ExprStmt{Expr: expr}
		}
	}

	if _, err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	body, err := p.parseStatement()
	if err != nil {
		return nil, err
	}

	return &ForStmt{Init: init, Cond: cond, Post: post, Body: body}, nil
}

// parseStatement dispatches to the correct sub-parser based on the leading token.
func (p *Parser) parseStatement() (Stmt, error) {
	tok := p.peek()
	switch tok.Type {

	case LBRACE:
		p.advance()
		return p.parseBlock()

	case IF:
		p.advance()
		return p.parseIf()

	case WHILE:
		p.advance()
		return p.parseWhile()

	case FOR:
		return p.parseForStmt()

	case SWITCH:
		return p.parseSwitchStmt()

	case ASM:
		p.advance()
		if _, err := p.expect(LPAREN); err != nil {
			return nil, err
		}
		strTok, err := p.expect(STRING)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(RPAREN); err != nil {
			return nil, err
		}
		if _, err := p.expect(SEMICOLON); err != nil {
			return nil, err
		}
		return &AsmStmt{Instruction: strTok.Lexeme}, nil

	case BREAK:
		p.advance()
		if _, err := p.expect(SEMICOLON); err != nil {
			return nil, err
		}
		return &BreakStmt{}, nil

	case CONTINUE:
		p.advance()
		if _, err := p.expect(SEMICOLON); err != nil {
			return nil, err
		}
		return &ContinueStmt{}, nil

	case INT, BYTE, UNSIGNED:
		return p.parseVarDecl()

	case STRUCT:
		// Variable declaration: struct Point p;
		// OR Struct definition: struct Point { ... };
		// Check lookahead
		if p.peekAt(2).Type == LBRACE {
			return p.parseStructDecl()
		}
		return p.parseVarDecl()

	case IDENTIFIER, STAR, LPAREN:
		// Expression statement or Assignment
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if p.peek().Type == ASSIGN ||
			p.peek().Type == PLUS_ASSIGN ||
			p.peek().Type == MINUS_ASSIGN ||
			p.peek().Type == STAR_ASSIGN ||
			p.peek().Type == SLASH_ASSIGN {
			return p.parseAssignment(expr)
		}
		if _, err := p.expect(SEMICOLON); err != nil {
			return nil, err
		}
		return &ExprStmt{Expr: expr}, nil

	case RETURN:
		p.advance()
		return p.parseReturn()

	case EOF:
		p.advance()
		return nil, nil // signals the top-level loop to stop

	default:
		p.advance()
		return nil, fmt.Errorf("line %d: unexpected token %s (%q)",
			tok.Line, tok.Type, tok.Lexeme)
	}
}

// parseFunctionDecl parses int name(params) { ... } or void name(params) { ... }
func (p *Parser) parseFunctionDecl() (Stmt, error) {
	var retType string
	if p.peek().Type == UNSIGNED {
		p.advance()
		retType = "unsigned"
		if p.peek().Type == INT {
			p.advance()
			retType += " int"
		}
		p.currentRetType = INT // Treat unsigned as INT for return checking
	} else if p.peek().Type == INT {
		p.advance()
		retType = "int"
		p.currentRetType = INT
	} else if p.peek().Type == BYTE {
		p.advance()
		retType = "byte"
		p.currentRetType = BYTE // This assumes we add BYTE to TokenType enum, which we did.
		// Note: p.currentRetType is TokenType. INT/VOID/BYTE match.
	} else if p.peek().Type == VOID {
		p.advance()
		retType = "void"
		p.currentRetType = VOID
	} else {
		return nil, fmt.Errorf("line %d: expected return type (int, byte, or void)", p.peek().Line)
	}

	// Step over optional '*' for the return type
	if p.peek().Type == STAR {
		p.advance()
		// If pointer, treat as int/pointer (size 2)
		// Return type string representation might need to be "int*" or "byte*"
		// But codegen likely doesn't check this string strictly, mainly symbol table.
		// However, currentRetType uses INT for pointers usually.
		retType += "*"
		p.currentRetType = INT // Pointers are word-sized (handled as INT in return checking usually)
	}

	nameTok, err := p.expect(IDENTIFIER)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(LPAREN); err != nil {
		return nil, err
	}

	var params []VariableDecl
	if p.peek().Type != RPAREN {
		for {
			var param VariableDecl
			if p.peek().Type == UNSIGNED {
				p.advance()
				param.IsUnsigned = true
				if p.peek().Type == INT {
					p.advance()
				}
				if p.peek().Type == STAR {
					p.advance()
					param.IsPointer = true
				}
			} else if p.peek().Type == INT {
				p.advance()
				if p.peek().Type == STAR {
					p.advance()
					param.IsPointer = true
				}
			} else if p.peek().Type == BYTE {
				p.advance()
				param.IsByte = true
				if p.peek().Type == STAR {
					p.advance()
					param.IsPointer = true
				}
			} else {
				return nil, fmt.Errorf("line %d: expected parameter type (int or byte)", p.peek().Line)
			}

			paramName, err := p.expect(IDENTIFIER)
			if err != nil {
				return nil, err
			}
			param.Name = paramName.Lexeme
			params = append(params, param)

			if p.peek().Type != COMMA {
				break
			}
			p.advance()
		}
	}

	if _, err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	if _, err := p.expect(LBRACE); err != nil {
		return nil, err
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &FunctionDecl{Name: nameTok.Lexeme, Params: params, Body: body, ReturnType: retType}, nil
}

// parseTopLevel parses either a function declaration or a statement.
func (p *Parser) parseTopLevel() (Stmt, error) {
	// Check for FunctionDecl: INT/BYTE IDENTIFIER LPAREN or INT/BYTE STAR IDENTIFIER LPAREN
	isFunc := false
	if p.peek().Type == UNSIGNED {
		// unsigned func()
		if p.peekAt(1).Type == IDENTIFIER && p.peekAt(2).Type == LPAREN {
			isFunc = true
		} else if p.peekAt(1).Type == STAR && p.peekAt(2).Type == IDENTIFIER && p.peekAt(3).Type == LPAREN {
			isFunc = true
		} else if p.peekAt(1).Type == INT {
			// unsigned int func()
			if p.peekAt(2).Type == IDENTIFIER && p.peekAt(3).Type == LPAREN {
				isFunc = true
			} else if p.peekAt(2).Type == STAR && p.peekAt(3).Type == IDENTIFIER && p.peekAt(4).Type == LPAREN {
				isFunc = true
			}
		}
	} else if p.peek().Type == INT || p.peek().Type == BYTE {
		if p.peekAt(1).Type == IDENTIFIER && p.peekAt(2).Type == LPAREN {
			isFunc = true
		} else if p.peekAt(1).Type == STAR && p.peekAt(2).Type == IDENTIFIER && p.peekAt(3).Type == LPAREN {
			isFunc = true
		}
	} else if p.peek().Type == VOID {
		if p.peekAt(1).Type == IDENTIFIER && p.peekAt(2).Type == LPAREN {
			isFunc = true
		} else if p.peekAt(1).Type == STAR && p.peekAt(2).Type == IDENTIFIER && p.peekAt(3).Type == LPAREN {
			isFunc = true
		}
	}
	// struct definitions can be top level
	if p.peek().Type == STRUCT && p.peekAt(2).Type == LBRACE {
		return p.parseStructDecl()
	}

	if isFunc {
		return p.parseFunctionDecl()
	}
	return p.parseStatement()
}

// Parse now enforces that only declarations are allowed at the top level.
func Parse(tokens []Token, rawSource string) ([]Stmt, error) {
	p := NewParser(tokens, rawSource)
	var stmts []Stmt
	for p.peek().Type != EOF {
		// 1. Check for Function Declaration
		isFunc := false
		if p.peek().Type == UNSIGNED {
			// unsigned func()
			if p.peekAt(1).Type == IDENTIFIER && p.peekAt(2).Type == LPAREN {
				isFunc = true
			} else if p.peekAt(1).Type == STAR && p.peekAt(2).Type == IDENTIFIER && p.peekAt(3).Type == LPAREN {
				isFunc = true
			} else if p.peekAt(1).Type == INT {
				// unsigned int func()
				if p.peekAt(2).Type == IDENTIFIER && p.peekAt(3).Type == LPAREN {
					isFunc = true
				} else if p.peekAt(2).Type == STAR && p.peekAt(3).Type == IDENTIFIER && p.peekAt(4).Type == LPAREN {
					isFunc = true
				}
			}
		} else if p.peek().Type == INT || p.peek().Type == BYTE {
			// Check lookahead for name(...) or *name(...)
			if p.peekAt(1).Type == IDENTIFIER && p.peekAt(2).Type == LPAREN {
				isFunc = true
			} else if p.peekAt(1).Type == STAR && p.peekAt(2).Type == IDENTIFIER && p.peekAt(3).Type == LPAREN {
				isFunc = true
			}
		} else if p.peek().Type == VOID {
			if p.peekAt(1).Type == IDENTIFIER && p.peekAt(2).Type == LPAREN {
				isFunc = true
			} else if p.peekAt(1).Type == STAR && p.peekAt(2).Type == IDENTIFIER && p.peekAt(3).Type == LPAREN {
				isFunc = true
			}
		}

		if isFunc {
			f, err := p.parseFunctionDecl()
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, f)
			continue
		}

		// 2. Check for Struct Definition
		if p.peek().Type == STRUCT && p.peekAt(2).Type == LBRACE {
			s, err := p.parseStructDecl()
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, s)
			continue
		}

		// 3. Check for Global Variable Declaration
		if p.peek().Type == INT || p.peek().Type == BYTE || p.peek().Type == STRUCT || p.peek().Type == UNSIGNED {
			v, err := p.parseVarDecl()
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, v)
			continue
		}

		// 4. If we hit anything else, it's a naked statement!
		tok := p.peek()
		return nil, fmt.Errorf("line %d: executable statement %q found outside of function body",
			tok.Line, tok.Lexeme)
	}
	return stmts, nil
}
