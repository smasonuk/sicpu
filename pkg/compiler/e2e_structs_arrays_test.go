package compiler

import (
	"testing"
)

func TestArrays_E2E(t *testing.T) {
	t.Run("ConstantIndex", func(t *testing.T) {
		src := `
		int main() {
			int arr[5];
			arr[0] = 42;
			arr[4] = 99;
			return arr[0];
		}
		`
		regs := runCode(t, src)
		if regs[0] != 42 {
			t.Errorf("Array constant index write/read failed: expected 42, got %d", regs[0])
		}
	})

	t.Run("ComputedIndex", func(t *testing.T) {
		src := `
		int main() {
			int arr[5];
			int i = 3;
			arr[i] = 77;
			return arr[3];
		}
		`
		regs := runCode(t, src)
		if regs[0] != 77 {
			t.Errorf("Array computed index write/read failed: expected 77, got %d", regs[0])
		}
	})
}

func TestStructs_E2E(t *testing.T) {
	t.Run("MemberAccess", func(t *testing.T) {
		src := `
		struct Point { int x; int y; };
		int main() {
			struct Point p;
			p.x = 10;
			p.y = 20;
			// Use pointer to write p.x to a known location, effectively testing both read of p.x and write to *pointer
			int* out = 0x3000;
			*out = p.x;
			return p.y;
		}
		`
		regs := runCode(t, src)
		if regs[0] != 20 {
			t.Errorf("Struct member read failed: expected 20, got %d", regs[0])
		}
	})
}

func TestPointers_E2E(t *testing.T) {
	t.Run("PointerArithmetic", func(t *testing.T) {
		src := `
		int main() {
			int* p = 0x3000;
			*(p + 0) = 11;
			*(p + 1) = 22;
			return *(p + 1);
		}
		`
		regs := runCode(t, src)
		if regs[0] != 22 {
			t.Errorf("Pointer arithmetic failed: expected 22, got %d", regs[0])
		}
	})
}
