package cpu

type Peripheral interface {
    Read16(offset uint16) uint16
    Write16(offset uint16, val uint16)
    Step()
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
