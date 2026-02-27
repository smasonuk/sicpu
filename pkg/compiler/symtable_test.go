package compiler

import (
	"testing"
)

func TestSymbolTable(t *testing.T) {
	t.Run("GlobalAllocation", func(t *testing.T) {
		s := NewSymbolTable()
		// Allocating 2 bytes (word) for each
		sym1, _ := s.Allocate("g1", TypeInfo{}, 2)
		sym2, _ := s.Allocate("g2", TypeInfo{}, 2)
		sym3, _ := s.Allocate("g3", TypeInfo{}, 2)

		if sym1.Label != "g1" {
			t.Errorf("g1 label: expected 'g1', got '%s'", sym1.Label)
		}
		if sym1.Address != 0 {
			t.Errorf("g1 address: expected 0 (ignored), got 0x%X", sym1.Address)
		}
		if sym1.Scope != ScopeGlobal {
			t.Errorf("g1 scope: expected Global, got %v", sym1.Scope)
		}

		if sym2.Label != "g2" {
			t.Errorf("g2 label: expected 'g2', got '%s'", sym2.Label)
		}

		if sym3.Label != "g3" {
			t.Errorf("g3 label: expected 'g3', got '%s'", sym3.Label)
		}
	})

	t.Run("ByteAllocation", func(t *testing.T) {
		s := NewSymbolTable()
		// byte b1;
		sym1, _ := s.Allocate("b1", TypeInfo{IsChar: true}, 1)
		// byte b2;
		sym2, _ := s.Allocate("b2", TypeInfo{IsChar: true}, 1)
		// int i;
		sym3, _ := s.Allocate("i", TypeInfo{}, 2)

		if sym1.Label != "b1" {
			t.Errorf("b1 label: expected 'b1', got '%s'", sym1.Label)
		}
		if sym1.Size != 1 {
			t.Errorf("b1 size: expected 1, got %d", sym1.Size)
		}

		if sym2.Label != "b2" {
			t.Errorf("b2 label: expected 'b2', got '%s'", sym2.Label)
		}

		if sym3.Label != "i" {
			t.Errorf("i label: expected 'i', got '%s'", sym3.Label)
		}
		if sym3.Size != 2 {
			t.Errorf("i size: expected 2, got %d", sym3.Size)
		}
	})

	t.Run("ArrayAllocation", func(t *testing.T) {
		s := NewSymbolTable()
		// int arr[10]; size = 20 bytes
		sym, _ := s.Allocate("arr", TypeInfo{IsArray: true, ArraySizes: []int{10}}, 20)
		if sym.Label != "arr" {
			t.Errorf("arr label: expected 'arr', got '%s'", sym.Label)
		}
		if sym.Size != 20 {
			t.Errorf("arr size: expected 20, got %d", sym.Size)
		}

		// next global
		next, _ := s.Allocate("next", TypeInfo{}, 2)
		if next.Label != "next" {
			t.Errorf("next label: expected 'next', got '%s'", next.Label)
		}
	})

	t.Run("StructAllocation", func(t *testing.T) {
		s := NewSymbolTable()
		// Define struct size 6 bytes (3 ints)
		s.DefineStruct(StructDef{Name: "Point3D", Size: 6})

		// Allocate variable of struct type
		sym, _ := s.Allocate("p", TypeInfo{IsStruct: true, StructName: "Point3D"}, 6)
		if sym.Label != "p" {
			t.Errorf("p label: expected 'p', got '%s'", sym.Label)
		}
		if sym.Size != 6 {
			t.Errorf("p size: expected 6, got %d", sym.Size)
		}

		// Next global
		next, _ := s.Allocate("next", TypeInfo{}, 2)
		if next.Label != "next" {
			t.Errorf("next label: expected 'next', got '%s'", next.Label)
		}
	})

	t.Run("LocalScoping", func(t *testing.T) {
		s := NewSymbolTable()
		s.EnterFunction()

		// Outer x (int)
		outer, _ := s.Allocate("x", TypeInfo{}, 2)

		s.EnterScope()
		// Inner x (shadow) (int)
		inner, _ := s.Allocate("x", TypeInfo{}, 2)

		lookedUp, found := s.Lookup("x")
		if !found {
			t.Errorf("Lookup(x) failed in inner scope")
		}
		if lookedUp.Address == outer.Address {
			t.Errorf("Lookup(x) returned outer symbol, expected inner")
		}
		if lookedUp.Address != inner.Address {
			t.Errorf("Lookup(x) returned wrong address")
		}
		// Locals should NOT have labels
		if lookedUp.Label != "" {
			t.Errorf("Local variable should not have a label, got '%s'", lookedUp.Label)
		}

		s.ExitScope()

		lookedUp, found = s.Lookup("x")
		if !found {
			t.Errorf("Lookup(x) failed in outer scope")
		}
		if lookedUp.Address != outer.Address {
			t.Errorf("Lookup(x) returned wrong address in outer scope")
		}
	})

	t.Run("ParamDefinition", func(t *testing.T) {
		s := NewSymbolTable()
		s.EnterFunction()
		s.DefineParam(VariableDecl{Name: "a"}, 0) // Param index 0 (register arg)

		sym, found := s.Lookup("a")
		if !found {
			t.Errorf("Lookup(a) failed")
		}
		// Register arg spilled to local stack -> negative offset
		// int size = 2. s.nextLocal decremented by 2.
		if sym.Address != -2 {
			t.Errorf("Param a address: expected -2, got %d", sym.Address)
		}
		if sym.Scope != ScopeLocal {
			t.Errorf("Param a scope: expected Local, got %v", sym.Scope)
		}
	})

	t.Run("LookupFailure", func(t *testing.T) {
		s := NewSymbolTable()
		_, found := s.Lookup("nonexistent")
		if found {
			t.Errorf("Lookup(nonexistent) succeeded, expected failure")
		}
	})

	t.Run("ExitFunction", func(t *testing.T) {
		s := NewSymbolTable()
		s.EnterFunction()
		s.Allocate("local", TypeInfo{}, 2)

		s.ExitFunction()

		_, found := s.Lookup("local")
		if found {
			t.Errorf("Lookup(local) succeeded after ExitFunction, expected failure")
		}

		// Globals should still remain (if we had any)
		s.Allocate("global", TypeInfo{}, 2)
		_, found = s.Lookup("global")
		if !found {
			t.Errorf("Lookup(global) failed")
		}
		if found && s.globals["global"].Label != "global" {
			t.Errorf("Expected global to have label 'global'")
		}
	})
}
