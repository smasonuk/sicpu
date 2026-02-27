package compiler

import (
	"fmt"
	"sort"
	"strings"
)

// CodeGen walks an AST and emits GoCPU assembly source text.
type CodeGen struct {
	syms            *SymbolTable
	out             strings.Builder
	nextLabel       int
	currentFunction string
	stringPool      map[string]string
	dataPool        map[string][]uint16 // Label -> Data
	dataCache       map[string]string   // Content -> Label
	loopStack       []LoopLabel
}

type LoopLabel struct {
	Start string
	End   string
	Post  string // where 'continue' jumps to
}

func newCodeGen(syms *SymbolTable) *CodeGen {
	return &CodeGen{
		syms:       syms,
		stringPool: make(map[string]string),
		dataPool:   make(map[string][]uint16),
		dataCache:  make(map[string]string),
	}
}

func (cg *CodeGen) newDataLabel() string {
	l := fmt.Sprintf("D%d", len(cg.dataPool))
	return l
}

func (cg *CodeGen) newLabel() string {
	l := fmt.Sprintf("L%d", cg.nextLabel)
	cg.nextLabel++
	return l
}

func (cg *CodeGen) newStringLabel() string {
	l := fmt.Sprintf("S%d", len(cg.stringPool))
	return l
}

func (cg *CodeGen) line(format string, args ...any) {
	if strings.Contains(format, ":") {
		// and args[0] is "print"
		if len(args) > 0 {
			if s, ok := args[0].(string); ok && s == "print" {
				fmt.Printf(format+"\n", args...)
			}
		}

	}

	fmt.Fprintf(&cg.out, format+"\n", args...)
}

func (cg *CodeGen) comment(format string, args ...any) {
	cg.line("; "+format, args...)
}

// calcSize determines the size in bytes of a variable declaration.
func (cg *CodeGen) calcSize(decl VariableDecl) (int, error) {
	elemSize := 2 // Default to int/pointer size (2 bytes)

	// Pointers (including pointers to incomplete/opaque structs) are always word-sized.
	// Check this first so we never attempt a struct lookup for a pointer type.
	if decl.PointerLevel == 0 {
		if decl.IsChar {
			elemSize = 1
		} else if decl.IsStruct {
			def, ok := cg.syms.GetStruct(decl.StructName)
			if !ok {
				return 0, fmt.Errorf("unknown struct %q", decl.StructName)
			}
			elemSize = def.Size
		}
	}

	if decl.IsArray {
		total := 1
		for _, s := range decl.ArraySizes {
			total *= s
		}
		return elemSize * total, nil
	}
	return elemSize, nil
}

// getType determines the type of an expression.
func (cg *CodeGen) getType(e Expr) (TypeInfo, error) {
	switch n := e.(type) {
	case *VarRef:
		sym, ok := cg.syms.Lookup(n.Name)
		if !ok {
			return TypeInfo{}, fmt.Errorf("undefined variable %q", n.Name)
		}
		return sym.Type, nil

	case *IndexExpr:
		// Array access. Result type is element type of Left.
		leftType, err := cg.getType(n.Left)
		if err != nil {
			return TypeInfo{}, err
		}

		if leftType.IsArray {
			// Check dimensions
			if len(n.Indices) > len(leftType.ArraySizes) {
				return TypeInfo{}, fmt.Errorf("too many indices for array")
			}

			remainingDims := leftType.ArraySizes[len(n.Indices):]
			if len(remainingDims) == 0 {
				// Fully indexed -> Element type
				return TypeInfo{
					IsArray:      false,
					IsStruct:     leftType.IsStruct,
					StructName:   leftType.StructName,
					IsChar:       leftType.IsChar,
					PointerLevel: leftType.PointerLevel,
					IsUnsigned:   leftType.IsUnsigned,
				}, nil
			} else {
				// Partially indexed -> Sub-array (which decays to pointer to first element of subarray in C, but here we treat as array type)
				return TypeInfo{
					IsArray:      true,
					ArraySizes:   remainingDims,
					IsStruct:     leftType.IsStruct,
					StructName:   leftType.StructName,
					IsChar:       leftType.IsChar,
					PointerLevel: leftType.PointerLevel,
					IsUnsigned:   leftType.IsUnsigned,
				}, nil
			}
		}
		// Pointer indexing: int *p; p[i].
		// Treated as scalar access.
		// If Left is a pointer to byte, result is byte.
		if leftType.PointerLevel > 0 {
			if len(n.Indices) > 1 {
				return TypeInfo{}, fmt.Errorf("multi-dimensional indexing not supported for pointers")
			}
			// Dereferencing decreases pointer level
			return TypeInfo{IsChar: leftType.IsChar, PointerLevel: leftType.PointerLevel - 1, IsUnsigned: leftType.IsUnsigned}, nil
		}
		return TypeInfo{}, nil

	case *MemberExpr:
		leftType, err := cg.getType(n.Left)
		if err != nil {
			return TypeInfo{}, err
		}
		if !leftType.IsStruct {
			return TypeInfo{}, fmt.Errorf("member access on non-struct type")
		}

		def, ok := cg.syms.GetStruct(leftType.StructName)
		if !ok {
			return TypeInfo{}, fmt.Errorf("unknown struct %q", leftType.StructName)
		}
		field, ok := def.Fields[n.Member]
		if !ok {
			return TypeInfo{}, fmt.Errorf("struct %s has no member %q", leftType.StructName, n.Member)
		}
		return field.Type, nil

	case *UnaryExpr:
		if n.Op == STAR {
			// Dereference *ptr.
			rightType, err := cg.getType(n.Right)
			if err != nil {
				return TypeInfo{}, err
			}
			// If rightType is pointer, decrement level.
			if rightType.PointerLevel > 0 {
				return TypeInfo{
					IsChar:       rightType.IsChar,
					PointerLevel: rightType.PointerLevel - 1,
					IsUnsigned:   rightType.IsUnsigned,
				}, nil
			}
			return TypeInfo{}, nil
		}

	case *Literal:
		return TypeInfo{IsUnsigned: n.IsUnsigned}, nil
	}

	// Default scalar
	return TypeInfo{}, nil
}

// genAddress computes the address of an expression and puts it in R1.
// Supports: VarRef, IndexExpr, MemberExpr, UnaryExpr(STAR).
func (cg *CodeGen) genAddress(e Expr) error {
	switch n := e.(type) {
	case *VarRef:
		sym, ok := cg.syms.Lookup(n.Name)
		if !ok {
			return fmt.Errorf("undefined variable %q", n.Name)
		}
		if sym.Scope == ScopeGlobal {
			cg.line("    LDI R1, %s    ; &%s (global)", sym.Label, n.Name)
		} else {
			// Local: Address is FP + offset.
			cg.line("    MOV R1, R2        ; FP")
			// Use offset directly. 2's complement handles negative.
			cg.line("    LDI R3, %d", uint16(sym.Address))
			cg.line("    ADD R1, R3        ; &%s (local/param)", n.Name)
		}
		return nil

	case *IndexExpr:
		// Left[Indices...]
		leftType, err := cg.getType(n.Left)
		if err != nil {
			return err
		}

		// Calculate Base Address
		if err := cg.genExpr(n.Left); err != nil {
			return err
		}
		// R0 has base address.
		cg.line("    PUSH R0")

		// Calculate Offset
		// We use a simplified approach: accumulate offset in a register, then add to base.
		// Since we have few registers, we'll use stack to hold intermediate offset accumulator if needed,
		// but we can just add to the base address iteratively?
		// No, standard `base + (idx0*stride0 + idx1*stride1 + ...)` is better.

		// Let's use R1 to accumulate total byte offset. Initialize to 0.
		cg.line("    LDI R1, 0")
		cg.line("    PUSH R1") // Stack: [Base, Offset=0]

		if leftType.IsArray {
			baseElemSize := 2
			if leftType.IsChar {
				baseElemSize = 1
			} else if leftType.IsStruct {
				def, ok := cg.syms.GetStruct(leftType.StructName)
				if !ok {
					return fmt.Errorf("unknown struct %q", leftType.StructName)
				}
				baseElemSize = def.Size
			}

			// Iterate indices
			for i, idxExpr := range n.Indices {
				// Calculate stride for this dimension
				// Stride = product of remaining dimensions * baseElemSize
				stride := baseElemSize
				// Remaining dims: from i+1 to end
				for j := i + 1; j < len(leftType.ArraySizes); j++ {
					stride *= leftType.ArraySizes[j]
				}

				// Evaluate index
				if err := cg.genExpr(idxExpr); err != nil {
					return err
				}
				// R0 has index value.

				// Multiply by stride
				if stride != 1 {
					cg.line("    LDI R3, %d", stride)
					cg.line("    MUL R0, R3")
				}

				// Add to accumulated offset
				cg.line("    POP R1") // Current offset
				cg.line("    ADD R1, R0")
				cg.line("    PUSH R1") // Save new offset
			}
		} else if leftType.PointerLevel > 0 {
			// Pointer arithmetic: only 1 index supported
			if len(n.Indices) != 1 {
				return fmt.Errorf("pointers only support single index")
			}
			elemSize := 2
			// If pointer to char (level 1), element size is 1.
			// If pointer to pointer (level > 1), element size is 2 (pointer size).
			if leftType.IsChar && leftType.PointerLevel == 1 {
				elemSize = 1
			}

			if err := cg.genExpr(n.Indices[0]); err != nil {
				return err
			}
			// R0 = index
			if elemSize == 2 {
				cg.line("    LDI R3, 1")
				cg.line("    SHL R0, R3")
			}
			// Add to offset (which is 0)
			cg.line("    POP R1")
			cg.line("    ADD R1, R0")
			cg.line("    PUSH R1")
		}

		// Final Address = Base + Offset
		cg.line("    POP R0") // Offset
		cg.line("    POP R1") // Base
		cg.line("    ADD R1, R0")
		return nil

	case *MemberExpr:
		// Left.Member
		// Need struct type info of Left.
		typ, err := cg.getType(n.Left)
		if err != nil {
			return err
		}
		if !typ.IsStruct {
			return fmt.Errorf("member access on non-struct type")
		}

		def, ok := cg.syms.GetStruct(typ.StructName)
		if !ok {
			return fmt.Errorf("unknown struct %q", typ.StructName)
		}
		field, ok := def.Fields[n.Member]
		if !ok {
			return fmt.Errorf("struct %s has no member %q", typ.StructName, n.Member)
		}

		// Address of Left
		// If Left is VarRef (struct instance): genExpr(Left) returns address of struct (because IsStruct=true).
		if err := cg.genExpr(n.Left); err != nil {
			return err
		}
		// R0 has base address.

		cg.line("    MOV R1, R0")
		cg.line("    LDI R3, %d", field.Offset)
		cg.line("    ADD R1, R3")
		return nil

	case *UnaryExpr:
		if n.Op == STAR {
			// *ptr -> Value of ptr is the address we want.
			if err := cg.genExpr(n.Right); err != nil {
				return err
			}
			cg.line("    MOV R1, R0")
			return nil
		}
	}
	return fmt.Errorf("cannot take address of expression type %T", e)
}

// genExpr emits the instructions that evaluate expr and leave the result in R0.
func (cg *CodeGen) genExpr(e Expr) error {
	switch n := e.(type) {

	case *VarRef:
		sym, ok := cg.syms.Lookup(n.Name)
		if !ok {
			return fmt.Errorf("undefined variable %q", n.Name)
		}

		// If Array or Struct, return address.
		if sym.Type.IsArray || sym.Type.IsStruct {
			if err := cg.genAddress(e); err != nil {
				return err
			}
			cg.line("    MOV R0, R1")
			return nil
		}

		// Scalar/Pointer: Load value.
		if err := cg.genAddress(e); err != nil {
			return err
		}
		if sym.Type.IsChar && sym.Type.PointerLevel == 0 {
			cg.line("    LDB R0, [R1]")
		} else {
			cg.line("    LD  R0, [R1]")
		}
		return nil

	case *IndexExpr, *MemberExpr:
		// Check type. If aggregate, return address. Else load.
		typ, err := cg.getType(e)
		if err != nil {
			return err
		}

		if typ.IsArray || typ.IsStruct {
			if err := cg.genAddress(e); err != nil {
				return err
			}
			cg.line("    MOV R0, R1")
			return nil
		}

		if err := cg.genAddress(e); err != nil {
			return err
		}
		if typ.IsChar && typ.PointerLevel == 0 {
			cg.line("    LDB R0, [R1]")
		} else {
			cg.line("    LD  R0, [R1]")
		}
		return nil

	case *LogicalExpr:
		if n.Op == AND_LOGICAL {
			endLabel := cg.newLabel()

			if err := cg.genExpr(n.Left); err != nil {
				return err
			}
			cg.line("    LDI R1, 0")
			cg.line("    SUB R0, R1")
			cg.line("    JZ  %s", endLabel) // Short-circuit: return 0

			if err := cg.genExpr(n.Right); err != nil {
				return err
			}
			cg.line("    LDI R1, 0")
			cg.line("    SUB R0, R1")
			cg.line("    JZ  %s", endLabel) // Return 0

			// If we are here, both were non-zero. Return 1.
			cg.line("    LDI R0, 1")
			cg.line("%s:", endLabel)
			return nil
		}

		if n.Op == OR_LOGICAL {
			endLabel := cg.newLabel()
			trueLabel := cg.newLabel()

			if err := cg.genExpr(n.Left); err != nil {
				return err
			}
			cg.line("    LDI R1, 0")
			cg.line("    SUB R0, R1")
			cg.line("    JNZ %s", trueLabel) // Short-circuit: return 1

			if err := cg.genExpr(n.Right); err != nil {
				return err
			}
			cg.line("    LDI R1, 0")
			cg.line("    SUB R0, R1")
			cg.line("    JNZ %s", trueLabel) // Return 1

			// Both 0. R0 is 0.
			cg.line("    JMP %s", endLabel)

			cg.line("%s:", trueLabel)
			cg.line("    LDI R0, 1")
			cg.line("%s:", endLabel)
			return nil
		}
		return fmt.Errorf("codegen: unknown logical operator %s", n.Op)

	case *BinaryExpr:
		// Optimization: Constant Folding
		// If both operands are literals, compute the result at compile time.
		if left, ok := n.Left.(*Literal); ok {
			if right, ok := n.Right.(*Literal); ok {
				// If either literal is explicitly unsigned (u/U suffix), treat the
				// whole operation as unsigned; otherwise default to signed (int).
				leftType, _ := cg.getType(n.Left)
				rightType, _ := cg.getType(n.Right)
				isUnsigned := leftType.IsUnsigned || rightType.IsUnsigned

				var res uint16
				switch n.Op {
				case PLUS:
					res = left.Value + right.Value
				case MINUS:
					res = left.Value - right.Value
				case STAR:
					res = left.Value * right.Value
				case SLASH:
					if right.Value == 0 {
						return fmt.Errorf("division by zero in constant expression")
					}
					if isUnsigned {
						res = left.Value / right.Value
					} else {
						res = uint16(int16(left.Value) / int16(right.Value))
					}
				case PERCENT:
					if right.Value == 0 {
						return fmt.Errorf("modulo by zero in constant expression")
					}
					if isUnsigned {
						res = left.Value % right.Value
					} else {
						res = uint16(int16(left.Value) % int16(right.Value))
					}
				case AND:
					res = left.Value & right.Value
				case PIPE:
					res = left.Value | right.Value
				case CARET:
					res = left.Value ^ right.Value
				case SHL_OP:
					res = left.Value << right.Value
				case SHR_OP:
					res = left.Value >> right.Value
				case EQUALS:
					if left.Value == right.Value {
						res = 1
					} else {
						res = 0
					}
				case NOT_EQ:
					if left.Value != right.Value {
						res = 1
					} else {
						res = 0
					}
				case LESS:
					if isUnsigned {
						if left.Value < right.Value {
							res = 1
						} else {
							res = 0
						}
					} else {
						if int16(left.Value) < int16(right.Value) {
							res = 1
						} else {
							res = 0
						}
					}
				case GREATER:
					if isUnsigned {
						if left.Value > right.Value {
							res = 1
						} else {
							res = 0
						}
					} else {
						if int16(left.Value) > int16(right.Value) {
							res = 1
						} else {
							res = 0
						}
					}
				default:
					goto RuntimeEval
				}
				cg.line("    LDI R0, %d", res)
				return nil
			}
		}

	RuntimeEval:
		if err := cg.genExpr(n.Left); err != nil {
			return err
		}
		cg.line("    PUSH R0")
		if err := cg.genExpr(n.Right); err != nil {
			return err
		}
		cg.line("    POP  R1")

		switch n.Op {
		case LESS_EQ:
			typ, err := cg.getType(n.Left)
			if err != nil {
				return err
			}
			labelFalse := cg.newLabel()
			labelEnd := cg.newLabel()
			cg.line("    SUB R0, R1") // Right - Left
			if typ.IsUnsigned {
				cg.line("    JC  %s", labelFalse) // Unsigned Right < Left (False)
			} else {
				cg.line("    JN  %s", labelFalse) // Signed Right < Left (False)
			}
			cg.line("    LDI R0, 1") // True
			cg.line("    JMP %s", labelEnd)
			cg.line("%s:", labelFalse)
			cg.line("    LDI R0, 0") // False
			cg.line("%s:", labelEnd)

		case GREATER_EQ:
			typ, err := cg.getType(n.Left)
			if err != nil {
				return err
			}
			labelFalse := cg.newLabel()
			labelEnd := cg.newLabel()
			cg.line("    SUB R1, R0") // Left - Right
			if typ.IsUnsigned {
				cg.line("    JC  %s", labelFalse) // Unsigned Left < Right (False)
			} else {
				cg.line("    JN  %s", labelFalse) // Signed Left < Right (False)
			}
			cg.line("    LDI R0, 1") // True
			cg.line("    JMP %s", labelEnd)
			cg.line("%s:", labelFalse)
			cg.line("    LDI R0, 0") // False
			cg.line("%s:", labelEnd)
		case PLUS:
			cg.line("    ADD R1, R0")
			cg.line("    MOV R0, R1")
		case MINUS:
			cg.line("    SUB R1, R0")
			cg.line("    MOV R0, R1")
		case STAR:
			cg.line("    MUL R1, R0")
			cg.line("    MOV R0, R1")
		case SLASH:
			// Check if we need signed or unsigned division
			typ, err := cg.getType(n.Left)
			if err != nil {
				return err
			}
			if typ.IsUnsigned {
				cg.line("    DIV R1, R0")
			} else {
				cg.line("    IDIV R1, R0")
			}
			cg.line("    MOV R0, R1")
		case EQUALS:
			label := cg.newLabel()
			cg.line("    SUB R1, R0")
			cg.line("    LDI R0, 1")
			cg.line("    JZ  %s", label)
			cg.line("    LDI R0, 0")
			cg.line("%s:", label)
		case NOT_EQ:
			label := cg.newLabel()
			cg.line("    SUB R1, R0")
			cg.line("    LDI R0, 1")
			cg.line("    JNZ %s", label)
			cg.line("    LDI R0, 0")
			cg.line("%s:", label)
		case LESS:
			// Check if we need signed or unsigned comparison
			typ, err := cg.getType(n.Left)
			if err != nil {
				return err
			}
			label := cg.newLabel()
			cg.line("    SUB R1, R0") // Left - Right
			cg.line("    LDI R0, 1")
			if typ.IsUnsigned {
				// Unsigned: Left < Right => Borrow (Carry)
				cg.line("    JC  %s", label)
			} else {
				// Signed: Left < Right => Negative
				cg.line("    JN  %s", label)
			}
			cg.line("    LDI R0, 0")
			cg.line("%s:", label)
		case GREATER:
			// Check if we need signed or unsigned comparison
			typ, err := cg.getType(n.Left)
			if err != nil {
				return err
			}
			label := cg.newLabel()
			cg.line("    SUB R0, R1") // Right - Left
			cg.line("    LDI R0, 1")
			if typ.IsUnsigned {
				// Unsigned: Right < Left => Borrow (Carry) => Left > Right
				cg.line("    JC  %s", label)
			} else {
				// Signed: Right < Left => Negative => Left > Right
				cg.line("    JN  %s", label)
			}
			cg.line("    LDI R0, 0")
			cg.line("%s:", label)
		case AND:
			cg.line("    AND R1, R0")
			cg.line("    MOV R0, R1")
		case PIPE:
			cg.line("    OR  R1, R0")
			cg.line("    MOV R0, R1")
		case CARET:
			cg.line("    XOR R1, R0")
			cg.line("    MOV R0, R1")
		case PERCENT:
			cg.line("    MOV R3, R1")
			cg.line("    DIV R1, R0")
			cg.line("    MUL R1, R0")
			cg.line("    SUB R3, R1")
			cg.line("    MOV R0, R3")
		case SHL_OP:
			// R1 = left operand, R0 = shift amount
			cg.line("    SHL R1, R0")
			cg.line("    MOV R0, R1")
		case SHR_OP:
			// R1 = left operand, R0 = shift amount
			cg.line("    SHR R1, R0")
			cg.line("    MOV R0, R1")
		default:
			return fmt.Errorf("codegen: unknown binary operator %s", n.Op)
		}

	case *UnaryExpr:
		if n.Op == AND {
			// Address-of: &x
			// Must be valid lvalue (VarRef, IndexExpr, MemberExpr)
			// genAddress checks type.
			if err := cg.genAddress(n.Right); err != nil {
				return err
			}
			cg.line("    MOV R0, R1")
			return nil
		}
		if n.Op == STAR {
			// Dereference: *ptr
			// Check type of result (*ptr).
			typ, err := cg.getType(e)
			if err != nil {
				return err
			}

			if err := cg.genExpr(n.Right); err != nil {
				return err
			}
			// R0 has address.
			cg.line("    MOV R1, R0")
			if typ.IsChar && typ.PointerLevel == 0 {
				cg.line("    LDB R0, [R1]")
			} else {
				cg.line("    LD  R0, [R1]")
			}
			return nil
		}
		if n.Op == TILDE {
			if err := cg.genExpr(n.Right); err != nil {
				return err
			}
			cg.line("    NOT R0")
			return nil
		}
		if n.Op == NOT {
			if err := cg.genExpr(n.Right); err != nil {
				return err
			}
			// If R0 == 0 -> 1, else -> 0
			labelTrue := cg.newLabel()
			labelEnd := cg.newLabel()
			cg.line("    LDI R1, 0")
			cg.line("    SUB R0, R1")
			cg.line("    JZ  %s", labelTrue)
			cg.line("    LDI R0, 0")
			cg.line("    JMP %s", labelEnd)
			cg.line("%s:", labelTrue)
			cg.line("    LDI R0, 1")
			cg.line("%s:", labelEnd)
			return nil
		}
		if n.Op == MINUS {
			if err := cg.genExpr(n.Right); err != nil {
				return err
			}
			cg.line("    LDI R1, 0")
			cg.line("    SUB R1, R0")
			cg.line("    MOV R0, R1")
			return nil
		}
		return fmt.Errorf("codegen: unknown unary operator %s", n.Op)

	case *CastExpr:
		if err := cg.genExpr(n.Expr); err != nil {
			return err
		}
		if n.Type == CHAR && n.PointerLevel == 0 {
			// Truncate to 8 bits
			cg.line("    LDI R1, 0x00FF")
			cg.line("    AND R0, R1")
		}
		// INT/Pointer/Struct pointer casts are no-ops on 16-bit machine (bit representation doesn't change)
		return nil

	case *Literal:
		cg.line("    LDI R0, %d", n.Value)

	case *StringLiteral:
		label, ok := cg.stringPool[n.Value]
		if !ok {
			label = cg.newStringLabel()
			cg.stringPool[n.Value] = label
		}
		cg.line("    LDI R0, %s", label)

	case *FunctionCall:
		for i := len(n.Args) - 1; i >= 0; i-- {
			if err := cg.genExpr(n.Args[i]); err != nil {
				return err
			}
			cg.line("    PUSH R0")
		}

		// Pop up to 4 args into registers
		regs := []string{"R4", "R5", "R6", "R7"}
		numRegArgs := len(n.Args)
		if numRegArgs > 4 {
			numRegArgs = 4
		}
		for i := 0; i < numRegArgs; i++ {
			cg.line("    POP %s", regs[i])
		}

		cg.line("    CALL %s", n.Name)

		if len(n.Args) > 4 {
			cg.line("    LDI R1, %d", (len(n.Args)-4)*2)
			cg.line("    LDSP R3")
			cg.line("    ADD R3, R1")
			cg.line("    STSP R3")
		}

	case *PostfixExpr:
		// x++
		// R0 = x. x = x + 1.
		if err := cg.genAddress(n.Left); err != nil {
			return err
		}
		// R1 = &x.
		cg.line("    LD  R0, [R1]")
		cg.line("    PUSH R0") // Save original value (result)

		// Calculate new value
		// R0 is current value.
		if n.Op == PLUS_PLUS {
			cg.line("    LDI R3, 1")
			cg.line("    ADD R0, R3")
		} else if n.Op == MINUS_MINUS {
			cg.line("    LDI R3, 1")
			cg.line("    SUB R0, R3")
		} else {
			return fmt.Errorf("codegen: unknown postfix op %s", n.Op)
		}

		// Store new value
		cg.line("    ST  [R1], R0")

		// Restore original value to R0
		cg.line("    POP R0")
		return nil

	default:
		return fmt.Errorf("codegen: unknown expression node %T", e)
	}
	return nil
}

// countLocals recursively counts needed stack space.
func (cg *CodeGen) countLocals(stmt Stmt) (int, error) {
	count := 0
	switch s := stmt.(type) {
	case *BlockStmt:
		for _, child := range s.Stmts {
			c, err := cg.countLocals(child)
			if err != nil {
				return 0, err
			}
			count += c
		}
	case *VariableDecl:
		size, err := cg.calcSize(*s)
		if err != nil {
			return 0, err
		}
		count += size
	case *IfStmt:
		c, err := cg.countLocals(s.Body)
		if err != nil {
			return 0, err
		}
		count += c
		if s.ElseBody != nil {
			c, err := cg.countLocals(s.ElseBody)
			if err != nil {
				return 0, err
			}
			count += c
		}
	case *WhileStmt:
		c, err := cg.countLocals(s.Body)
		if err != nil {
			return 0, err
		}
		count += c
	case *ForStmt:
		if s.Init != nil {
			c, err := cg.countLocals(s.Init)
			if err != nil {
				return 0, err
			}
			count += c
		}
		if s.Body != nil {
			c, err := cg.countLocals(s.Body)
			if err != nil {
				return 0, err
			}
			count += c
		}
	case *SwitchStmt:
		for _, clause := range s.Cases {
			for _, child := range clause.Body {
				c, err := cg.countLocals(child)
				if err != nil {
					return 0, err
				}
				count += c
			}
		}
		for _, child := range s.Default {
			c, err := cg.countLocals(child)
			if err != nil {
				return 0, err
			}
			count += c
		}
	case *StructDecl:
		// Define struct layout in symtable so subsequent VariableDecls can find it.
		// Note: This defines it globally/in the map. C allows local struct definitions.
		// Our SymbolTable has a single struct map, so this effectively makes it global
		// or overwrites previous definitions. For this C-subset, this is acceptable.
		def := StructDef{
			Name:   s.Name,
			Fields: make(map[string]FieldInfo),
			Size:   0,
		}
		byteOffset := 0
		for _, field := range s.Fields {
			size, err := cg.calcSize(field)
			if err != nil {
				return 0, err
			}

			typeInfo := TypeInfo{
				IsArray:      field.IsArray,
				ArraySizes:   field.ArraySizes,
				IsStruct:     field.IsStruct,
				StructName:   field.StructName,
				IsChar:       field.IsChar,
				PointerLevel: field.PointerLevel,
				IsUnsigned:   field.IsUnsigned,
			}

			def.Fields[field.Name] = FieldInfo{Offset: byteOffset, Type: typeInfo}
			byteOffset += size
		}
		def.Size = byteOffset
		cg.syms.DefineStruct(def)

	case *ExprStmt, *Assignment, *ReturnStmt:
		return 0, nil

	default:
		// Other statements don't allocate locals (e.g. FunctionDecl inside function? Not valid C)
		return 0, nil
	}
	return count, nil
}

// genStmt emits the instructions that carry out stmt.
func (cg *CodeGen) genStmt(s Stmt) error {
	switch n := s.(type) {

	case *ExprStmt:
		cg.comment("call: %s", n.Expr)
		if err := cg.genExpr(n.Expr); err != nil {
			return err
		}

	case *StructDecl:
		// Define struct layout in symtable.
		def := StructDef{
			Name:   n.Name,
			Fields: make(map[string]FieldInfo),
			Size:   0,
		}
		byteOffset := 0
		for _, field := range n.Fields {
			size, err := cg.calcSize(field)
			if err != nil {
				return err
			}

			// field.Type info
			typeInfo := TypeInfo{
				IsArray:      field.IsArray,
				ArraySizes:   field.ArraySizes,
				IsStruct:     field.IsStruct,
				StructName:   field.StructName,
				IsChar:       field.IsChar,
				PointerLevel: field.PointerLevel,
				IsUnsigned:   field.IsUnsigned,
			}

			def.Fields[field.Name] = FieldInfo{Offset: byteOffset, Type: typeInfo}
			byteOffset += size
		}
		def.Size = byteOffset
		cg.syms.DefineStruct(def)
		cg.comment("struct %s defined (size %d)", n.Name, def.Size)

	case *VariableDecl:
		size, err := cg.calcSize(*n)
		if err != nil {
			return err
		}

		typeInfo := TypeInfo{
			IsArray:      n.IsArray,
			ArraySizes:   n.ArraySizes,
			IsStruct:     n.IsStruct,
			StructName:   n.StructName,
			IsChar:       n.IsChar,
			PointerLevel: n.PointerLevel,
			IsUnsigned:   n.IsUnsigned,
		}

		sym, exists := cg.syms.Allocate(n.Name, typeInfo, size)
		if exists {
			return fmt.Errorf("redeclaration of %q", n.Name)
		} // Should have been caught? or allowed shadowing?

		cg.comment("var %s (size %d) at offset %d", n.Name, size, sym.Address)

		if n.Init != nil {
			if n.IsArray || n.IsStruct {
				if list, isList := n.Init.(*InitializerList); isList {
					// Local array initialization
					vals := make([]uint16, 0, len(list.Elements))
					keyBuilder := strings.Builder{}
					for i, elem := range list.Elements {
						lit, ok := elem.(*Literal)
						if !ok {
							return fmt.Errorf("local array initializer must be constant")
						}
						vals = append(vals, lit.Value)
						if i > 0 {
							keyBuilder.WriteString(",")
						}
						fmt.Fprintf(&keyBuilder, "%d", lit.Value)
					}
					key := keyBuilder.String()

					label, ok := cg.dataCache[key]
					if !ok {
						label = cg.newDataLabel()
						cg.dataCache[key] = label
						cg.dataPool[label] = vals
					}

					// Calculate Destination Address (R1)
					if sym.Scope == ScopeGlobal {
						cg.line("    LDI R1, %s", sym.Label)
					} else {
						cg.line("    MOV R1, R2")
						cg.line("    LDI R3, %d", uint16(sym.Address))
						cg.line("    ADD R1, R3")
					}

					// Copy from Data (Source R0) to Stack (Dest R1)
					cg.line("    PUSH R2")
					cg.line("    LDI R0, %s", label)
					cg.line("    LDI R2, %d", len(vals))
					cg.line("    COPY R0, R1, R2")
					cg.line("    POP R2")
					return nil
				}

				return fmt.Errorf("array/struct initialization not supported")
			}

			if err := cg.genExpr(n.Init); err != nil {
				return err
			}

			storeOp := "ST "
			if n.IsChar && n.PointerLevel == 0 && !n.IsArray {
				storeOp = "STB"
			}

			if sym.Scope == ScopeGlobal {
				cg.line("    LDI R1, %s", sym.Label)
				cg.line("    %s [R1], R0", storeOp)
			} else {
				cg.line("    MOV R1, R2")
				cg.line("    LDI R3, %d", uint16(sym.Address))
				cg.line("    ADD R1, R3")
				cg.line("    %s [R1], R0", storeOp)
			}
		}

	case *Assignment:
		cg.comment("%s %s ...", n.Left, n.Op)

		// Determine type of LHS to decide on ST vs STB
		lhsType, err := cg.getType(n.Left)
		if err != nil {
			return err
		}

		// If LHS is *ptr = ...
		// genAddress handles *ptr.

		if err := cg.genAddress(n.Left); err != nil {
			return err
		}
		// R1 has address.
		cg.line("    PUSH R1") // Save address

		// If compound assignment, we need to load the current value first.
		if n.Op != ASSIGN {
			if lhsType.IsChar && lhsType.PointerLevel == 0 && !lhsType.IsArray && !lhsType.IsStruct {
				cg.line("    LDB R0, [R1]")
			} else {
				cg.line("    LD  R0, [R1]")
			}
			cg.line("    PUSH R0") // Save current value
		}

		if err := cg.genExpr(n.Value); err != nil {
			return err
		}
		// R0 has RHS value.

		if n.Op != ASSIGN {
			cg.line("    POP R1") // Restore current value (LHS)
			// R0 is RHS, R1 is LHS.
			// Result goes to R0.
			switch n.Op {
			case PLUS_ASSIGN:
				cg.line("    ADD R1, R0")
				cg.line("    MOV R0, R1")
			case MINUS_ASSIGN:
				cg.line("    SUB R1, R0")
				cg.line("    MOV R0, R1")
			case STAR_ASSIGN:
				cg.line("    MUL R1, R0")
				cg.line("    MOV R0, R1")
			case SLASH_ASSIGN:
				cg.line("    DIV R1, R0")
				cg.line("    MOV R0, R1")
			default:
				return fmt.Errorf("codegen: unknown assignment op %s", n.Op)
			}
		}

		cg.line("    POP R1") // Restore address
		storeOp := "ST "
		if lhsType.IsChar && lhsType.PointerLevel == 0 && !lhsType.IsArray && !lhsType.IsStruct {
			storeOp = "STB"
		}
		cg.line("    %s [R1], R0", storeOp)

	case *ReturnStmt:
		if n.Expr != nil {
			cg.comment("return %s", n.Expr)
			if err := cg.genExpr(n.Expr); err != nil {
				return err
			}
		} else {
			cg.comment("return (void)")
		}

		if cg.syms.inFunction() {
			cg.line("    STSP R2")
			cg.line("    POP R2")
			if cg.currentFunction == "isr" {
				cg.line("    RETI")
			} else {
				cg.line("    RET")
			}
		} else {
			cg.line("    HLT")
		}

	case *BlockStmt:
		if cg.syms.inFunction() {
			cg.syms.EnterScope()
			defer cg.syms.ExitScope()
		}
		for _, stmt := range n.Stmts {
			if err := cg.genStmt(stmt); err != nil {
				return err
			}
		}

	case *IfStmt:
		cg.comment("if %s", n.Condition)
		if err := cg.genExpr(n.Condition); err != nil {
			return err
		}
		cg.line("    LDI R1, 0")
		cg.line("    SUB R0, R1")
		falseLabel := cg.newLabel()
		cg.line("    JZ  %s", falseLabel)
		if err := cg.genStmt(n.Body); err != nil {
			return err
		}
		if n.ElseBody != nil {
			endLabel := cg.newLabel()
			cg.line("    JMP %s", endLabel)
			cg.line("%s:", falseLabel)
			if err := cg.genStmt(n.ElseBody); err != nil {
				return err
			}
			cg.line("%s:", endLabel)
		} else {
			cg.line("%s:", falseLabel)
		}

	case *WhileStmt:
		cg.comment("while %s", n.Condition)
		startLabel := cg.newLabel()
		endLabel := cg.newLabel()

		// For while loops, continue jumps to start (condition check)
		cg.loopStack = append(cg.loopStack, LoopLabel{Start: startLabel, End: endLabel, Post: startLabel})

		cg.line("%s:", startLabel)
		if err := cg.genExpr(n.Condition); err != nil {
			return err
		}
		cg.line("    LDI R1, 0")
		cg.line("    SUB R0, R1")
		cg.line("    JZ  %s", endLabel)
		if err := cg.genStmt(n.Body); err != nil {
			return err
		}
		cg.line("    JMP %s", startLabel)
		cg.line("%s:", endLabel)

		cg.loopStack = cg.loopStack[:len(cg.loopStack)-1]

	case *ForStmt:
		if cg.syms.inFunction() {
			cg.syms.EnterScope()
			defer cg.syms.ExitScope()
		}

		if n.Init != nil {
			if err := cg.genStmt(n.Init); err != nil {
				return err
			}
		}

		startLabel := cg.newLabel()
		endLabel := cg.newLabel()
		postLabel := cg.newLabel()

		// For for loops, continue jumps to post-iteration step
		cg.loopStack = append(cg.loopStack, LoopLabel{Start: startLabel, End: endLabel, Post: postLabel})

		cg.line("%s:", startLabel)

		if n.Cond != nil {
			cg.comment("for cond")
			if err := cg.genExpr(n.Cond); err != nil {
				return err
			}
			cg.line("    LDI R1, 0")
			cg.line("    SUB R0, R1")
			cg.line("    JZ  %s", endLabel)
		}

		if err := cg.genStmt(n.Body); err != nil {
			return err
		}

		cg.line("%s:", postLabel)
		if n.Post != nil {
			cg.comment("for post")
			if err := cg.genStmt(n.Post); err != nil {
				return err
			}
		}

		cg.line("    JMP %s", startLabel)
		cg.line("%s:", endLabel)

		cg.loopStack = cg.loopStack[:len(cg.loopStack)-1]

	case *BreakStmt:
		if len(cg.loopStack) == 0 {
			return fmt.Errorf("break statement outside of loop")
		}
		label := cg.loopStack[len(cg.loopStack)-1].End
		cg.line("    JMP %s", label)

	case *ContinueStmt:
		if len(cg.loopStack) == 0 {
			return fmt.Errorf("continue statement outside of loop")
		}
		label := cg.loopStack[len(cg.loopStack)-1].Post
		cg.line("    JMP %s", label)

	case *AsmStmt:
		cg.line("%s", n.Instruction)

	case *SwitchStmt:
		cg.comment("switch %s", n.Target)
		// Evaluate target expression -> R0
		if err := cg.genExpr(n.Target); err != nil {
			return err
		}

		// Push target to stack for comparison against cases
		cg.line("    PUSH R0")

		endLabel := cg.newLabel()

		for _, clause := range n.Cases {
			caseLabel := cg.newLabel()
			nextCaseLabel := cg.newLabel()

			// Get target from stack (PEEK) into R1
			// Note: LDSP gets SP. In GoCPU, SP points to last pushed value.
			cg.line("    LDSP R1")
			cg.line("    LD  R1, [R1]") // R1 = target

			// Evaluate case value -> R0
			if err := cg.genExpr(clause.Value); err != nil {
				return err
			}

			// Compare R1 (target) == R0 (case)
			cg.line("    SUB R1, R0")
			cg.line("    JZ  %s", caseLabel)
			cg.line("    JMP %s", nextCaseLabel)

			cg.line("%s:", caseLabel)
			// Generate body
			for _, stmt := range clause.Body {
				if err := cg.genStmt(stmt); err != nil {
					return err
				}
			}
			cg.line("    JMP %s", endLabel)

			cg.line("%s:", nextCaseLabel)
		}

		// Default case
		if len(n.Default) > 0 {
			for _, stmt := range n.Default {
				if err := cg.genStmt(stmt); err != nil {
					return err
				}
			}
		}

		cg.line("%s:", endLabel)
		cg.line("    POP R0") // Discard target from stack

	case *FunctionDecl:
		skipLabel := cg.newLabel()
		cg.line("    JMP %s", skipLabel)

		cg.syms.EnterFunction()
		cg.currentFunction = n.Name

		for i, param := range n.Params {
			cg.syms.DefineParam(param, i)
		}

		localsSize, err := cg.countLocals(n.Body)
		if err != nil {
			return err
		}

		// Calculate total stack frame size: body locals + spilled register params
		spilledSize := int(-cg.syms.nextLocal)
		totalFrameSize := localsSize + spilledSize

		cg.line("%s:", n.Name)
		cg.line("    PUSH R2")
		cg.line("    LDSP R2")

		if totalFrameSize > 0 {
			cg.line("    LDI R1, %d", totalFrameSize)
			cg.line("    LDSP R3")
			cg.line("    SUB R3, R1")
			cg.line("    STSP R3")
		}

		// Spill register arguments (R4-R7) to their local stack slots
		argRegs := []string{"R4", "R5", "R6", "R7"}
		for i, param := range n.Params {
			if i >= 4 {
				break
			}
			sym, ok := cg.syms.Lookup(param.Name)
			if !ok {
				return fmt.Errorf("param %s not found in symbol table", param.Name)
			}
			cg.comment("Spill param %s (%s) to local offset %d", param.Name, argRegs[i], sym.Address)

			cg.line("    MOV R1, R2")
			cg.line("    LDI R3, %d", uint16(sym.Address))
			cg.line("    ADD R1, R3")

			storeOp := "ST "
			if param.IsChar && param.PointerLevel == 0 && !param.IsArray {
				storeOp = "STB"
			}
			cg.line("    %s [R1], %s", storeOp, argRegs[i])
		}

		if err := cg.genStmt(n.Body); err != nil {
			return err
		}

		cg.line("    STSP R2")
		cg.line("    POP R2")
		if n.Name == "isr" {
			cg.line("    RETI")
		} else {
			cg.line("    RET")
		}

		cg.currentFunction = ""
		cg.syms.ExitFunction()
		cg.line("%s:", skipLabel)

	default:
		return fmt.Errorf("codegen: unknown statement node %T", s)
	}
	return nil
}

func Generate(stmts []Stmt, syms *SymbolTable) (string, error) {
	// 1. Run Dead Code Elimination
	stmts = eliminateDeadFunctions(stmts)

	cg := newCodeGen(syms)

	// 0. Process Struct Declarations
	for _, s := range stmts {
		if decl, ok := s.(*StructDecl); ok {
			if err := cg.genStmt(decl); err != nil {
				return "", err
			}
		}
	}

	// 1. PRE-PASS: Allocate Global Symbols (to prevent redeclaration errors)
	for _, s := range stmts {
		if decl, ok := s.(*VariableDecl); ok {
			size, _ := cg.calcSize(*decl)
			typeInfo := TypeInfo{
				IsArray:      decl.IsArray,
				ArraySizes:   decl.ArraySizes,
				IsStruct:     decl.IsStruct,
				StructName:   decl.StructName,
				IsChar:       decl.IsChar,
				PointerLevel: decl.PointerLevel,
				IsUnsigned:   decl.IsUnsigned,
			}
			cg.syms.Allocate(decl.Name, typeInfo, size)
		}
	}

	// 2. Entry Point & Interrupt Vector
	hasISR, hasMain := false, false
	for _, s := range stmts {
		if f, ok := s.(*FunctionDecl); ok {
			if f.Name == "isr" {
				hasISR = true
			}
			if f.Name == "main" {
				hasMain = true
			}
		}
	}

	if hasMain {
		cg.line("    JMP __init")
	} else {
		cg.line("    JMP __start")
	}
	cg.line("    .ORG 0x0010")
	if hasISR {
		cg.line("    JMP isr")
	} else {
		cg.line("    RETI")
	}

	// 3. Global Initializations (__init)
	if hasMain {
		cg.line("__init:")
		for _, s := range stmts {
			if decl, ok := s.(*VariableDecl); ok && decl.Init != nil {
				// Optimization: If Init is a literal, we handle it in the .DATA section
				// using .WORD directives, so skip runtime initialization here.
				if _, isLiteral := decl.Init.(*Literal); isLiteral {
					continue
				}
				if _, isList := decl.Init.(*InitializerList); isList {
					continue
				}

				if err := cg.genExpr(decl.Init); err != nil {
					return "", err
				}
				sym, _ := cg.syms.Lookup(decl.Name)
				cg.line("    LDI R1, %s", sym.Label)

				storeOp := "ST "
				if decl.IsChar && decl.PointerLevel == 0 && !decl.IsArray {
					storeOp = "STB"
				}

				cg.line("    %s [R1], R0", storeOp)
			}
		}
		cg.line("    CALL main")
		cg.line("    HLT")
	}

	if !hasMain {
		cg.line("__start:")
	}

	// 4. Function Bodies
	for _, s := range stmts {
		if _, ok := s.(*FunctionDecl); ok {
			cg.out.WriteByte('\n')
			if err := cg.genStmt(s); err != nil {
				return "", err
			}
		}
	}

	cg.line("\n    HLT")

	// 5. Global Data
	cg.line("\n; Global Data")
	// Collect initializers
	initMap := make(map[string]Expr)
	for _, s := range stmts {
		if decl, ok := s.(*VariableDecl); ok {
			initMap[decl.Name] = decl.Init
		}
	}

	// Get global symbols
	var globalNames []string
	for name := range syms.globals {
		globalNames = append(globalNames, name)
	}
	sort.Strings(globalNames)

	for _, name := range globalNames {
		sym := syms.globals[name]
		cg.line("%s:", sym.Label)

		initExpr := initMap[name]
		handled := false

		if initExpr != nil {
			// Helper function to resolve basic literals and negative literals
			resolveConstant := func(e Expr) (uint16, bool) {
				if lit, ok := e.(*Literal); ok {
					return lit.Value, true
				}
				if un, ok := e.(*UnaryExpr); ok && un.Op == MINUS {
					if lit, ok := un.Right.(*Literal); ok {
						return uint16(-int16(lit.Value)), true // 2's complement
					}
				}
				return 0, false
			}

			if val, ok := resolveConstant(initExpr); ok {
				// Handle scalar
				cg.line(".WORD %d", val)
				handled = true
			} else if list, ok := initExpr.(*InitializerList); ok {
				// Handle array
				for _, elem := range list.Elements {
					if val, ok := resolveConstant(elem); ok {
						cg.line(".WORD %d", val)
					} else {
						return "", fmt.Errorf("global arrays must be initialized with constant values")
					}
				}
				handled = true
			}
		}

		if !handled {
			// Uninitialized or runtime-initialized -> emit 0s.
			// Align to 2 bytes (word).
			// If Size is 1 (byte), words = 1.
			words := (sym.Size + 1) / 2
			for k := 0; k < words; k++ {
				cg.line(".WORD 0")
			}
		}
	}

	if len(cg.stringPool) > 0 {
		cg.line("\n; String Literals")
		// We iterate by index to ensure S0, S1, S2 order
		for i := 0; i < len(cg.stringPool); i++ {
			label := fmt.Sprintf("S%d", i)
			var val string
			for v, l := range cg.stringPool {
				if l == label {
					val = v
					break
				}
			}
			// Escape special characters for the assembler
			val = strings.ReplaceAll(val, "\n", "\\n")
			cg.line("%s: .STRING \"%s\"", label, val)
		}
	}

	if len(cg.dataPool) > 0 {
		cg.line("\n; Local Initializer Data")
		for i := 0; i < len(cg.dataPool); i++ {
			label := fmt.Sprintf("D%d", i)
			vals := cg.dataPool[label]
			cg.line("%s:", label)
			for _, v := range vals {
				cg.line(".WORD %d", v)
			}
		}
	}

	return cg.out.String(), nil
}
