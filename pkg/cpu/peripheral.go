package cpu

// Peripheral is the interface all expansion bus peripherals must implement.
type Peripheral interface {
	Read16(offset uint16) uint16
	Write16(offset uint16, val uint16)
	Step()
	Type() string
}

// StatefulPeripheral is an optional interface peripherals may implement to
// persist their internal state across hibernation cycles.
type StatefulPeripheral interface {
	SaveState() []byte
	LoadState(data []byte) error
}

// PeripheralFactory is a constructor that creates a Peripheral for a given CPU
// and expansion-bus slot index.
type PeripheralFactory func(c *CPU, slot uint8) Peripheral

var peripheralRegistry = make(map[string]PeripheralFactory)

// RegisterPeripheral registers a factory under the given name so that
// RestoreFromBytes can re-instantiate peripherals by type string.
func RegisterPeripheral(name string, factory PeripheralFactory) {
	peripheralRegistry[name] = factory
}

// EncodePeripheralName takes a string (up to 7 chars) and returns
// the correct 16-bit word for the given MMIO offset (0x08-0x0E).
func EncodePeripheralName(name string, offset uint16) uint16 {
	charIdx := (int(offset) - 0x08) / 2
	stringIdx := charIdx * 2

	var lo, hi byte
	if stringIdx < len(name) {
		lo = name[stringIdx]
	}
	if stringIdx+1 < len(name) {
		hi = name[stringIdx+1]
	}
	return uint16(lo) | (uint16(hi) << 8)
}
