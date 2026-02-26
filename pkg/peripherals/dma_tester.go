package peripherals

import (
	"gocpu/pkg/cpu"
)

type DMATester struct {
	c    *cpu.CPU
	slot uint8

	targetAddr uint16
	length     uint16
}

func NewDMATester(c *cpu.CPU, slot uint8) *DMATester {
	return &DMATester{
		c:    c,
		slot: slot,
	}
}

func (d *DMATester) Read16(offset uint16) uint16 {
	switch offset {
	case 0x00: // Command/Status
		return 0 // Always return 0 for status for now
	case 0x02:
		return d.targetAddr
	case 0x04:
		return d.length
	}
	return 0
}

func (d *DMATester) Write16(offset uint16, val uint16) {
	switch offset {
	case 0x00: // Command
		if val == 1 {
			d.performDMA()
		}
	case 0x02:
		d.targetAddr = val
	case 0x04:
		d.length = val
	}
}

func (d *DMATester) performDMA() {
	// Allocate a slice of bytes matching the transfer length (fill it with a test pattern, e.g., 0xAA).
	// Iterate over the length and call c.cpu.WriteByte(targetAddr + i, val).
	// Call c.cpu.TriggerPeripheralInterrupt(c.slot).

	for i := uint16(0); i < d.length; i++ {
		d.c.WriteByte(d.targetAddr+i, 0xAA)
	}

	// Clear status register (conceptually done)

	d.c.TriggerPeripheralInterrupt(d.slot)
}

func (d *DMATester) Step() {
	// No background task for this tester
}
