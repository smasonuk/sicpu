package compiler

import "fmt"

//  Expression nodes

// Expr is implemented by every node that produces a value.
// genExpr always leaves the result in R0.
type Expr interface {
	exprNode()
	String() string
}

// Literal is a compile-time integer constant.
//
//	int x = 10;
//	         ^^  Literal{Value: 10}
//	int x = 10u;
//	         ^^^  Literal{Value: 10, IsUnsigned: true}
type Literal struct {
	Value      uint16
	IsUnsigned bool // true when the source had a u/U suffix, e.g. 10u
}

func (*Literal) exprNode()        {}
func (l *Literal) String() string { return fmt.Sprintf("%d", l.Value) }

// StringLiteral is a string constant "..."
type StringLiteral struct {
	Value string
}

func (*StringLiteral) exprNode()        {}
func (s *StringLiteral) String() string { return fmt.Sprintf("%q", s.Value) }

// InitializerList represents { expr, expr, ... }
type InitializerList struct {
	Elements []Expr
}

func (*InitializerList) exprNode() {}
func (l *InitializerList) String() string {
	return fmt.Sprintf("InitializerList(len=%d, %v)", len(l.Elements), l.Elements)
}

// VarRef is a read of a named variable.
//
//	return x;
//	       ^  VarRef{Name: "x"}
type VarRef struct {
	Name string
}

func (*VarRef) exprNode()        {}
func (v *VarRef) String() string { return v.Name }

// BinaryExpr represents a binary operation: Left Op Right.
//
//	x + 1
//	^ ^ ^
//	| | |
//	| | Right
//	| Op
//	Left
type BinaryExpr struct {
	Op    TokenType
	Left  Expr
	Right Expr
}

func (*BinaryExpr) exprNode() {}
func (b *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left, b.Op, b.Right)
}

// LogicalExpr represents a logical operation: Left && Right or Left || Right.
// It is separate from BinaryExpr to allow short-circuit evaluation in code generation.
type LogicalExpr struct {
	Op    TokenType
	Left  Expr
	Right Expr
}

func (*LogicalExpr) exprNode() {}
func (l *LogicalExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", l.Left, l.Op, l.Right)
}

// UnaryExpr represents Op Right (e.g., &x, *p).
type UnaryExpr struct {
	Op    TokenType
	Right Expr
}

func (*UnaryExpr) exprNode()        {}
func (u *UnaryExpr) String() string { return fmt.Sprintf("(%s %s)", u.Op, u.Right) }

// PostfixExpr represents Left++ or Left--
type PostfixExpr struct {
	Op   TokenType
	Left Expr
}

func (*PostfixExpr) exprNode()        {}
func (p *PostfixExpr) String() string { return fmt.Sprintf("(%s %s)", p.Left, p.Op) }

// FunctionCall represents name(args)
type FunctionCall struct {
	Name string
	Args []Expr
}

func (*FunctionCall) exprNode() {}
func (c *FunctionCall) String() string {
	return fmt.Sprintf("FunctionCall(%s, args=%v)", c.Name, c.Args)
}

// CastExpr represents (Type) Expr or (Type*) Expr
type CastExpr struct {
	Type         TokenType // INT, CHAR, STRUCT
	StructName   string    // if Type == STRUCT
	PointerLevel int       // 0 for scalar, 1 for *, 2 for **, etc.
	Expr         Expr
}

func (*CastExpr) exprNode() {}
func (c *CastExpr) String() string {
	typeStr := c.Type.String()
	if c.Type == STRUCT {
		typeStr += " " + c.StructName
	}
	for i := 0; i < c.PointerLevel; i++ {
		typeStr += "*"
	}
	return fmt.Sprintf("Cast(%s, %s)", typeStr, c.Expr)
}

// IndexExpr represents Left[Index]
type IndexExpr struct {
	Left    Expr
	Indices []Expr
}

func (*IndexExpr) exprNode()        {}
func (e *IndexExpr) String() string { return fmt.Sprintf("(%s%v)", e.Left, e.Indices) }

// MemberExpr represents Left.Member
type MemberExpr struct {
	Left   Expr
	Member string
}

func (*MemberExpr) exprNode()        {}
func (e *MemberExpr) String() string { return fmt.Sprintf("(%s.%s)", e.Left, e.Member) }

//  Statement nodes

// Stmt is implemented by every node that does not produce a value.
type Stmt interface {
	stmtNode()
	String() string
}

// VariableDecl represents  int name = expr;
type VariableDecl struct {
	Name         string
	Init         Expr
	IsArray      bool
	ArraySizes   []int
	IsStruct     bool
	StructName   string
	IsChar       bool
	PointerLevel int // 0 for scalar, 1 for *, 2 for **, etc.
	IsUnsigned   bool
}

func (*VariableDecl) stmtNode() {}
func (d *VariableDecl) String() string {
	typeStr := "int"
	if d.IsChar {
		typeStr = "char"
	} else if d.IsStruct {
		typeStr = "struct " + d.StructName
	}

	for i := 0; i < d.PointerLevel; i++ {
		typeStr += "*"
	}

	if d.IsArray {
		return fmt.Sprintf("VariableDecl(%s %s%v)", typeStr, d.Name, d.ArraySizes)
	}
	return fmt.Sprintf("VariableDecl(%s %s = %s)", typeStr, d.Name, d.Init)
}

// StructDecl represents struct Name { int field1; ... }
type StructDecl struct {
	Name   string
	Fields []VariableDecl // Init is nil
}

func (*StructDecl) stmtNode() {}
func (s *StructDecl) String() string {
	return fmt.Sprintf("StructDecl(struct %s, fields=%v)", s.Name, s.Fields)
}

// Assignment represents  Left = Value;
type Assignment struct {
	Left  Expr
	Op    TokenType
	Value Expr
}

func (*Assignment) stmtNode() {}
func (a *Assignment) String() string {
	return fmt.Sprintf("Assignment(%s %s %s)", a.Left, a.Op, a.Value)
}

// ReturnStmt represents  return expr;
type ReturnStmt struct {
	Expr Expr
}

func (*ReturnStmt) stmtNode() {}
func (r *ReturnStmt) String() string {
	return fmt.Sprintf("ReturnStmt(%s)", r.Expr)
}

// BlockStmt represents { statement; ... }
type BlockStmt struct {
	Stmts []Stmt
}

func (*BlockStmt) stmtNode() {}
func (b *BlockStmt) String() string {
	return fmt.Sprintf("BlockStmt(len=%d)", len(b.Stmts))
}

// IfStmt represents if (cond) body [else elseBody]
type IfStmt struct {
	Condition Expr
	Body      Stmt
	ElseBody  Stmt // may be nil
}

func (*IfStmt) stmtNode() {}
func (i *IfStmt) String() string {
	if i.ElseBody != nil {
		return fmt.Sprintf("IfStmt(if %s then %s else %s)", i.Condition, i.Body, i.ElseBody)
	}
	return fmt.Sprintf("IfStmt(if %s then %s)", i.Condition, i.Body)
}

// WhileStmt represents while (cond) body
type WhileStmt struct {
	Condition Expr
	Body      Stmt
}

func (*WhileStmt) stmtNode() {}
func (w *WhileStmt) String() string {
	return fmt.Sprintf("WhileStmt(while %s do %s)", w.Condition, w.Body)
}

// ForStmt represents for (init; cond; post) body
type ForStmt struct {
	Init Stmt
	Cond Expr
	Post Stmt
	Body Stmt
}

func (*ForStmt) stmtNode() {}
func (f *ForStmt) String() string {
	return fmt.Sprintf("ForStmt(init=%s, cond=%s, post=%s, body=%s)", f.Init, f.Cond, f.Post, f.Body)
}

// FunctionDecl represents int name(params) { body }
type FunctionDecl struct {
	Name       string
	Params     []VariableDecl
	Body       Stmt // typically BlockStmt
	ReturnType string
}

func (*FunctionDecl) stmtNode() {}
func (f *FunctionDecl) String() string {
	return fmt.Sprintf("FunctionDecl(%s %s, params=%v, body=%s)", f.ReturnType, f.Name, f.Params, f.Body)
}

// ExprStmt represents an expression evaluated for its side effects (e.g. a function call).
type ExprStmt struct {
	Expr Expr
}

func (*ExprStmt) stmtNode() {}
func (e *ExprStmt) String() string {
	return fmt.Sprintf("ExprStmt(%s)", e.Expr)
}

// AsmStmt represents asm("instruction");
type AsmStmt struct {
	Instruction string
}

func (*AsmStmt) stmtNode() {}
func (a *AsmStmt) String() string {
	return fmt.Sprintf("AsmStmt(%q)", a.Instruction)
}

// CaseClause represents case Value: Body
type CaseClause struct {
	Value Expr
	Body  []Stmt
}

// SwitchStmt represents switch (Target) { Cases... Default... }
type SwitchStmt struct {
	Target  Expr
	Cases   []CaseClause
	Default []Stmt
}

func (*SwitchStmt) stmtNode() {}
func (s *SwitchStmt) String() string {
	return fmt.Sprintf("SwitchStmt(target=%s, cases=%d, default=%d)", s.Target, len(s.Cases), len(s.Default))
}

// BreakStmt represents break;
type BreakStmt struct{}

func (*BreakStmt) stmtNode()        {}
func (s *BreakStmt) String() string { return "BreakStmt" }

// ContinueStmt represents continue;
type ContinueStmt struct{}

func (*ContinueStmt) stmtNode()        {}
func (s *ContinueStmt) String() string { return "ContinueStmt" }
