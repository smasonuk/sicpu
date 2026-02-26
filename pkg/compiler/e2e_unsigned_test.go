package compiler

import "testing"

// TestUnsigned_E2E tests the signed/unsigned integer feature added to the compiler.
// Key behavioral differences from signed int:
//   - Division uses DIV (unsigned) instead of IDIV (signed)
//   - Comparisons < and > use JC (carry/borrow) instead of JN (negative flag)
func TestUnsigned_E2E(t *testing.T) {
	t.Run("BasicUnsignedVar", func(t *testing.T) {
		src := `
		int main() {
			unsigned x = 42;
			return x;
		}
		`
		regs := runCode(t, src)
		if regs[0] != 42 {
			t.Errorf("expected 42, got %d", regs[0])
		}
	})

	t.Run("UnsignedIntKeyword", func(t *testing.T) {
		src := `
		int main() {
			unsigned int x = 100;
			return x;
		}
		`
		regs := runCode(t, src)
		if regs[0] != 100 {
			t.Errorf("expected 100, got %d", regs[0])
		}
	})

	t.Run("SignedDivisionNegative", func(t *testing.T) {
		// int uses IDIV (signed), so -10 / 2 = -5
		src := `
		int main() {
			int x = -10;
			return x / 2;
		}
		`
		regs := runCode(t, src)
		if int16(regs[0]) != -5 {
			t.Errorf("signed division: expected -5, got %d", int16(regs[0]))
		}
	})

	t.Run("UnsignedDivision", func(t *testing.T) {
		// 0xFFF6 = 65526 unsigned; 65526 / 2 = 32763
		// Signed IDIV would interpret 0xFFF6 as -10, giving -5 (0xFFFB) — a different result.
		src := `
		int main() {
			unsigned int x = 0xFFF6;
			return x / 2;
		}
		`
		regs := runCode(t, src)
		if regs[0] != 32763 {
			t.Errorf("unsigned division: expected 32763 (0x7FFB), got %d (0x%X)", regs[0], regs[0])
		}
	})

	t.Run("UnsignedLessThanLargeValue", func(t *testing.T) {
		// 0x8000 = 32768 unsigned; 32768 < 5 is false.
		// Signed JN would interpret 0x8000 as -32768, and (-32768 - 5) wraps to a
		// positive number, so the signed path also returns 0 here — but for the wrong
		// reason. The unsigned JC path is guaranteed correct for all values.
		src := `
		int main() {
			unsigned int x = 0x8000;
			if (x < 5) return 1;
			return 0;
		}
		`
		regs := runCode(t, src)
		if regs[0] != 0 {
			t.Errorf("unsigned <: 32768 < 5 should be false, got %d", regs[0])
		}
	})

	t.Run("UnsignedGreaterThanLargeValue", func(t *testing.T) {
		// 0x8000 = 32768 unsigned; 32768 > 5 is true.
		// Signed JN would see (5 - 32768) = 0x7FFD which is positive, giving 0 (wrong).
		// Unsigned JC: borrow occurs because 5 < 32768, giving 1 (correct).
		src := `
		int main() {
			unsigned int x = 0x8000;
			if (x > 5) return 1;
			return 0;
		}
		`
		regs := runCode(t, src)
		if regs[0] != 1 {
			t.Errorf("unsigned >: 32768 > 5 should be true, got %d", regs[0])
		}
	})

	t.Run("UnsignedLessThanComparison", func(t *testing.T) {
		// Normal-range unsigned comparison: 10 < 20 is true.
		src := `
		int main() {
			unsigned int a = 10;
			unsigned int b = 20;
			if (a < b) return 1;
			return 0;
		}
		`
		regs := runCode(t, src)
		if regs[0] != 1 {
			t.Errorf("unsigned <: 10 < 20 should be true, got %d", regs[0])
		}
	})

	t.Run("UnsignedGreaterThanComparison", func(t *testing.T) {
		// Normal-range unsigned comparison: 30 > 20 is true.
		src := `
		int main() {
			unsigned int a = 30;
			unsigned int b = 20;
			if (a > b) return 1;
			return 0;
		}
		`
		regs := runCode(t, src)
		if regs[0] != 1 {
			t.Errorf("unsigned >: 30 > 20 should be true, got %d", regs[0])
		}
	})

	t.Run("UnsignedGlobalVar", func(t *testing.T) {
		src := `
		unsigned g = 1000;
		int main() {
			g = g + 500;
			return g;
		}
		`
		regs := runCode(t, src)
		if regs[0] != 1500 {
			t.Errorf("unsigned global: expected 1500, got %d", regs[0])
		}
	})

	t.Run("UnsignedFunctionParam", func(t *testing.T) {
		// The division uses DIV (unsigned) because the parameter x is unsigned.
		// 0xFFF6 / 2 = 32763, not -5.
		src := `
		int halve(unsigned int x) {
			return x / 2;
		}
		int main() {
			return halve(0xFFF6);
		}
		`
		regs := runCode(t, src)
		if regs[0] != 32763 {
			t.Errorf("unsigned param division: expected 32763, got %d", regs[0])
		}
	})

	t.Run("UnsignedReturnType", func(t *testing.T) {
		src := `
		unsigned int make_large() {
			unsigned int x = 0xFFF0;
			return x;
		}
		int main() {
			return make_large();
		}
		`
		regs := runCode(t, src)
		if regs[0] != 0xFFF0 {
			t.Errorf("unsigned return: expected 0xFFF0 (%d), got 0x%X (%d)", uint16(0xFFF0), regs[0], regs[0])
		}
	})

	t.Run("UnsignedForLoop", func(t *testing.T) {
		// Sum even numbers 0, 2, 4, 6, 8 using an unsigned loop variable.
		src := `
		int main() {
			unsigned int sum = 0;
			for (unsigned int i = 0; i < 10; i += 2) {
				sum = sum + i;
			}
			return sum;
		}
		`
		regs := runCode(t, src)
		if regs[0] != 20 {
			t.Errorf("unsigned for loop: expected 20, got %d", regs[0])
		}
	})

	t.Run("UnsignedArithmetic", func(t *testing.T) {
		// Basic add/subtract with unsigned, result stays unsigned.
		src := `
		int main() {
			unsigned int x = 500;
			x = x + 300;
			x = x - 100;
			return x;
		}
		`
		regs := runCode(t, src)
		if regs[0] != 700 {
			t.Errorf("unsigned arithmetic: expected 700, got %d", regs[0])
		}
	})
}
