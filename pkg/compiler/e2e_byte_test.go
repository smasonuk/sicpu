package compiler

import (
	"testing"
)

func runByteTest(t *testing.T, code string) uint16 {
	// runCode is defined in logical_test.go (same package)
	regs := runCode(t, code)
	return regs[0]
}

func TestByteVar(t *testing.T) {
	// byte b = 10; return b;
	res := runByteTest(t, "int main() { byte b = 10; return b; }")
	if res != 10 {
		t.Errorf("Expected 10, got %d", res)
	}
}

func TestByteOverflow(t *testing.T) {
	// byte b = 255; b = b + 1; return b;
	// 255 + 1 = 256. Storing to byte should truncate to 0.
	// Loading back should be 0.
	res := runByteTest(t, "int main() { byte b = 255; b = b + 1; return b; }")
	if res != 0 {
		t.Errorf("Expected 0 (overflow), got %d", res)
	}
}

func TestByteArray(t *testing.T) {
	// byte arr[3]; arr[0]=10; arr[1]=20; return arr[0]+arr[1];
	res := runByteTest(t, "int main() { byte arr[3]; arr[0]=10; arr[1]=20; return arr[0]+arr[1]; }")
	if res != 30 {
		t.Errorf("Expected 30, got %d", res)
	}
}

func TestBytePointer(t *testing.T) {
	// byte b = 42; byte *p; p = &b; return *p;
	res := runByteTest(t, "int main() { byte b = 42; byte *p; p = &b; return *p; }")
	if res != 42 {
		t.Errorf("Expected 42, got %d", res)
	}
}

func TestBytePointerWrite(t *testing.T) {
	// byte b = 0; byte *p; p = &b; *p = 50; return b;
	res := runByteTest(t, "int main() { byte b = 0; byte *p; p = &b; *p = 50; return b; }")
	if res != 50 {
		t.Errorf("Expected 50, got %d", res)
	}
}

func TestIntVsByteSize(t *testing.T) {
	// int i = 0x1234; byte *p; p = &i; return *p;
	// Little endian: 0x1234 -> 34 12. *p should be 0x34 (52).
	res := runByteTest(t, "int main() { int i = 0x1234; byte *p; p = &i; return *p; }")
	if res != 0x34 {
		t.Errorf("Expected 0x34 (low byte), got 0x%X", res)
	}

	// p[1] should be 0x12 (18).
	res = runByteTest(t, "int main() { int i = 0x1234; byte *p; p = &i; return p[1]; }")
	if res != 0x12 {
		t.Errorf("Expected 0x12 (high byte), got 0x%X", res)
	}
}

func TestStructWithBytes(t *testing.T) {
    // struct S { byte b1; byte b2; int i; };
    // struct S s; s.b1 = 1; s.b2 = 2; s.i = 0x1234; return s.b1 + s.b2;
    // offsets: b1=0, b2=1, i=2.
    // Total size 4 bytes.
	code := `
	struct S { byte b1; byte b2; int i; };
	int main() {
		struct S s;
		s.b1 = 1;
		s.b2 = 2;
		s.i = 3;
		return s.b1 + s.b2;
	}
	`
	res := runByteTest(t, code)
	if res != 3 {
		t.Errorf("Expected 3, got %d", res)
	}
}
