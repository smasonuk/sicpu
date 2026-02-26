package cpu

type Peripheral interface {
    Read16(offset uint16) uint16
    Write16(offset uint16, val uint16)
    Step()
}
