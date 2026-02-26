package compiler

import (
	"fmt"
	"sort"
	"strings"
)

type ScopeType int

const (
	ScopeGlobal ScopeType = iota
	ScopeLocal
)

type TypeInfo struct {
	IsArray    bool
	ArraySizes []int
	IsStruct   bool
	StructName string
	IsByte     bool
	IsPointer  bool
	IsUnsigned bool
}

type FieldInfo struct {
	Offset int
	Type   TypeInfo
}

type StructDef struct {
	Name   string
	Fields map[string]FieldInfo
	Size   int
}

type Symbol struct {
	Address int // offset from FP for locals; ignored for globals
	Label   string
	Size    int
	Scope   ScopeType
	Type    TypeInfo
}

// SymbolTable maps variable names to memory addresses or stack offsets.
// Globals use labels resolved by the assembler.
// Locals are assigned negative offsets from FP.
type SymbolTable struct {
	globals map[string]Symbol

	// Stack of local scopes.
	// Each scope maps name -> Symbol.
	locals []map[string]Symbol

	// Next available local offset (monotonically decreasing).
	nextLocal int16

	structs map[string]StructDef
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		globals: make(map[string]Symbol),
		structs: make(map[string]StructDef),
	}
}

func (s *SymbolTable) EnterFunction() {
	// Initialize stack with one scope (function body).
	s.locals = []map[string]Symbol{make(map[string]Symbol)}
	// FP points to saved FP (which is 2 bytes). Locals start at FP.
	// We allocate downwards.
	s.nextLocal = 0
}

func (s *SymbolTable) EnterScope() {
	if len(s.locals) == 0 {
		panic("EnterScope called outside function")
	}
	s.locals = append(s.locals, make(map[string]Symbol))
}

func (s *SymbolTable) ExitScope() {
	if len(s.locals) > 0 {
		s.locals = s.locals[:len(s.locals)-1]
	}
}

func (s *SymbolTable) ExitFunction() {
	s.locals = nil
}

func (s *SymbolTable) DefineParam(decl VariableDecl, paramIndex int) {
	if len(s.locals) == 0 {
		panic("DefineParam called outside function scope")
	}
	typeInfo := TypeInfo{
		IsArray:    decl.IsArray,
		ArraySizes: decl.ArraySizes,
		IsStruct:   decl.IsStruct,
		StructName: decl.StructName,
		IsByte:     decl.IsByte,
		IsPointer:  decl.IsPointer,
		IsUnsigned: decl.IsUnsigned,
	}

	// Calculate size
	size := 2
	if decl.IsByte {
		size = 1
	} else if decl.IsStruct {
		if def, ok := s.GetStruct(decl.StructName); ok {
			size = def.Size
		}
	}

	if decl.IsPointer {
		size = 2
	}

	if decl.IsArray {
		total := 1
		for _, s := range decl.ArraySizes {
			total *= s
		}
		size *= total
	}

	var offset int
	if paramIndex < 4 {
		// Register arguments are spilled to local stack space
		s.nextLocal -= int16(size)
		offset = int(s.nextLocal)
	} else {
		// Arguments 5+ are on the caller's stack
		offset = 4 + (paramIndex-4)*2
	}

	// Params are defined in the function-level scope (index 0).
	s.locals[0][decl.Name] = Symbol{
		Address: offset,
		Size:    size,
		Scope:   ScopeLocal,
		Type:    typeInfo,
	}
}

func (s *SymbolTable) DefineStruct(def StructDef) {
	s.structs[def.Name] = def
}

func (s *SymbolTable) GetStruct(name string) (StructDef, bool) {
	d, ok := s.structs[name]
	return d, ok
}

// Allocate assigns the next free address/offset to name in the CURRENT scope.
// If name is already in the current scope, existing symbol is returned.
func (s *SymbolTable) Allocate(name string, typeInfo TypeInfo, size int) (Symbol, bool) {
	if len(s.locals) > 0 {
		currentScope := s.locals[len(s.locals)-1]
		if sym, ok := currentScope[name]; ok {
			return sym, true
		}

		// For locals (growing down):
		// Reserve 'size' bytes.
		s.nextLocal -= int16(size)
		offset := s.nextLocal

		sym := Symbol{
			Address: int(offset),
			Size:    size,
			Scope:   ScopeLocal,
			Type:    typeInfo,
		}
		currentScope[name] = sym
		return sym, false
	}

	if sym, ok := s.globals[name]; ok {
		return sym, true
	}

	sym := Symbol{
		Label: name,
		Size:  size,
		Scope: ScopeGlobal,
		Type:  typeInfo,
	}
	s.globals[name] = sym
	return sym, false
}

// Lookup returns the symbol and whether it was found.
func (s *SymbolTable) Lookup(name string) (Symbol, bool) {
	// Search locals from top of stack down
	for i := len(s.locals) - 1; i >= 0; i-- {
		if sym, ok := s.locals[i][name]; ok {
			return sym, true
		}
	}

	// Search globals
	sym, ok := s.globals[name]
	return sym, ok
}

// inFunction returns true if we are inside a function.
func (s *SymbolTable) inFunction() bool {
	return len(s.locals) > 0
}

// String returns a deterministically ordered dump of the table.
func (s *SymbolTable) String() string {
	var sb strings.Builder
	if len(s.globals) > 0 {
		sb.WriteString("Globals:\n")
		names := make([]string, 0, len(s.globals))
		for name := range s.globals {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			sym := s.globals[name]
			fmt.Fprintf(&sb, "  %-20s  Label: %s (Size: %d, Type: %+v)\n", name, sym.Label, sym.Size, sym.Type)
		}
	} else {
		sb.WriteString("Globals: (empty)\n")
	}

	if len(s.locals) > 0 {
		sb.WriteString("Locals (Active Stack):\n")
		for i, scope := range s.locals {
			fmt.Fprintf(&sb, "  Scope %d:\n", i)
			names := make([]string, 0, len(scope))
			for name := range scope {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				sym := scope[name]
				fmt.Fprintf(&sb, "    %-20s  Offset: %d (Size: %d, Type: %+v)\n", name, sym.Address, sym.Size, sym.Type)
			}
		}
	}

	if len(s.structs) > 0 {
		sb.WriteString("Structs:\n")
		names := make([]string, 0, len(s.structs))
		for name := range s.structs {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			def := s.structs[name]
			fmt.Fprintf(&sb, "  struct %s (Size: %d): %v\n", name, def.Size, def.Fields)
		}
	}
	return sb.String()
}
