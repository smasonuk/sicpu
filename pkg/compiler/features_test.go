package compiler

import "testing"

func TestUnaryMinus_E2E(t *testing.T) {
	src := `
	int main() {
		int x = 10;
		int y = -x; // -10 (0xFFF6)
        int z = -(-5); // 5
		return y + z; // -5
	}
	`
	regs := runCode(t, src)
	if int16(regs[0]) != -5 {
		t.Errorf("Unary minus: expected -5, got %d", int16(regs[0]))
	}
}

func TestCharLiteral_E2E(t *testing.T) {
	src := `
	int main() {
		char c = 'A';
        if (c == 65) return 1;
		return 0;
	}
	`
	regs := runCode(t, src)
	if regs[0] != 1 {
		t.Errorf("Char literal: expected 1, got %d", regs[0])
	}
}

func TestExplicitCast_E2E(t *testing.T) {
	src := `
	int main() {
		int x = 0x1234;
        char b = (char)x; // 0x34
        int y = (int)b;   // 0x0034
		return y;
	}
	`
	regs := runCode(t, src)
	if regs[0] != 0x34 {
		t.Errorf("Explicit cast: expected 0x34, got 0x%X", regs[0])
	}
}

func TestBreakContinue_E2E(t *testing.T) {
	src := `
	int main() {
		int sum = 0;
        for (int i = 0; i < 10; i++) {
            if (i == 5) continue; // Skip 5
            if (i == 8) break;    // Stop at 8
            sum += i;
        }
        // Sum: 0+1+2+3+4+6+7 = 23
		return sum;
	}
	`
	regs := runCode(t, src)
	if regs[0] != 23 {
		t.Errorf("Break/Continue: expected 23, got %d", regs[0])
	}
}

func TestWhileBreakContinue_E2E(t *testing.T) {
    src := `
    int main() {
        int i = 0;
        int sum = 0;
        while (i < 10) {
            i++;
            if (i == 5) continue; // Skip adding 5
            if (i == 8) break;    // Stop loop when i becomes 8
            sum += i;
        }
        return sum;
    }
    `
    regs := runCode(t, src)
    if regs[0] != 23 {
        t.Errorf("While Break/Continue: expected 23, got %d", regs[0])
    }
}
