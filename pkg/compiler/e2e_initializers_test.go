package compiler

import "testing"

func TestInitializers_E2E(t *testing.T) {
	t.Run("GlobalArray", func(t *testing.T) {
		src := `
		int g[3] = {10, 20, 30};
		int main() {
			return g[1];
		}
		`
		regs := runCode(t, src)
		if regs[0] != 20 {
			t.Errorf("Global array init failed: expected 20, got %d", regs[0])
		}
	})

	t.Run("LocalArray", func(t *testing.T) {
		src := `
		int main() {
			int arr[3] = {1, 2, 3};
			return arr[2];
		}
		`
		regs := runCode(t, src)
		if regs[0] != 3 {
			t.Errorf("Local array init failed: expected 3, got %d", regs[0])
		}
	})

	t.Run("InferredSize", func(t *testing.T) {
		src := `
		int main() {
			int arr[] = {10, 20, 30, 40};
			return arr[3];
		}
		`
		regs := runCode(t, src)
		if regs[0] != 40 {
			t.Errorf("Inferred array size init failed: expected 40, got %d", regs[0])
		}
	})

	t.Run("MixedInit", func(t *testing.T) {
		src := `
		int g[] = {5, 6};
		int main() {
			int l[] = {7, 8};
			return g[0] + l[1]; // 5 + 8 = 13
		}
		`
		regs := runCode(t, src)
		if regs[0] != 13 {
			t.Errorf("Mixed global/local init failed: expected 13, got %d", regs[0])
		}
	})
}
