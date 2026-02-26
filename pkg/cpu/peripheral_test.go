package cpu

import "testing"

func TestEncodePeripheralName(t *testing.T) {
	name := "MSGSNDR"

	// Offset 0x08: M, S
	expected := uint16('M') | (uint16('S') << 8)
	actual := EncodePeripheralName(name, 0x08)
	if actual != expected {
		t.Errorf("Offset 0x08: Expected 0x%04X, got 0x%04X", expected, actual)
	}

	// Offset 0x0A: G, S
	expected = uint16('G') | (uint16('S') << 8)
	actual = EncodePeripheralName(name, 0x0A)
	if actual != expected {
		t.Errorf("Offset 0x0A: Expected 0x%04X, got 0x%04X", expected, actual)
	}

	// Offset 0x0C: N, D
	expected = uint16('N') | (uint16('D') << 8)
	actual = EncodePeripheralName(name, 0x0C)
	if actual != expected {
		t.Errorf("Offset 0x0C: Expected 0x%04X, got 0x%04X", expected, actual)
	}

	// Offset 0x0E: R, \0
	expected = uint16('R')
	actual = EncodePeripheralName(name, 0x0E)
	if actual != expected {
		t.Errorf("Offset 0x0E: Expected 0x%04X, got 0x%04X", expected, actual)
	}
}
