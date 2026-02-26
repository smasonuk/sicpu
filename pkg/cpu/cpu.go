package cpu

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"

	"gocpu/pkg/vfs"
)

const (
	OpHLT  uint16 = 0x00
	OpNOP  uint16 = 0x01
	OpLDI  uint16 = 0x02
	OpMOV  uint16 = 0x03
	OpLD   uint16 = 0x04
	OpST   uint16 = 0x05
	OpADD  uint16 = 0x06
	OpSUB  uint16 = 0x07
	OpAND  uint16 = 0x08
	OpOR   uint16 = 0x09
	OpXOR  uint16 = 0x0A
	OpNOT  uint16 = 0x0B
	OpSHL  uint16 = 0x0C
	OpSHR  uint16 = 0x0D
	OpJMP  uint16 = 0x0E
	OpJZ   uint16 = 0x0F
	OpJNZ  uint16 = 0x10
	OpJN   uint16 = 0x11
	OpPUSH uint16 = 0x12
	OpPOP  uint16 = 0x13
	OpCALL uint16 = 0x14
	OpRET  uint16 = 0x15
	OpEI   uint16 = 0x16
	OpDI   uint16 = 0x17
	OpRETI uint16 = 0x18
	OpWFI  uint16 = 0x19
	OpLDSP uint16 = 0x1A
	OpSTSP uint16 = 0x1B
	OpMUL  uint16 = 0x1C
	OpDIV  uint16 = 0x1D
	OpFILL uint16 = 0x1E
	OpCOPY uint16 = 0x1F
	OpLDB  uint16 = 0x20
	OpSTB  uint16 = 0x21
	OpIDIV uint16 = 0x22
	OpJC   uint16 = 0x23
	OpJNC  uint16 = 0x24
)

const (
	RegA uint16 = 0
	RegB uint16 = 1
	RegC uint16 = 2
	RegD uint16 = 3
)

type CPU struct {
	Regs [8]uint16

	PC uint16
	SP uint16

	Z  bool
	N  bool
	C  bool
	IE bool

	Waiting bool

	InterruptPending bool

	Memory [65536]byte

	TextVRAM       [1024]uint16
	TextVRAM_Front [1024]uint16

	GraphicsBanks      [4][16384]byte
	GraphicsBanksFront [4][16384]byte
	CurrentBank        uint16
	DisplayBank        uint16

	TextResolutionMode uint16

	// GraphicsEnabled is set by writing bit 1 of MMIO register 0xFF05.
	GraphicsEnabled bool
	// TextOverlay controls whether the text VRAM layer is drawn on top of
	// bitmap graphics (bit 0 of 0xFF05). Defaults to true (text visible).
	TextOverlay bool
	// BufferedMode enables double buffering (bit 2 of 0xFF05).
	BufferedMode bool
	// ColorMode8bpp selects 8bpp unpacked VRAM mode (bit 3 of 0xFF05).
	ColorMode8bpp bool

	// Palette is the 256-entry Color Look-Up Table (CLUT) in RGB565 format.
	Palette [256]uint16
	// PaletteIndex is the currently selected palette entry for MMIO reads/writes.
	PaletteIndex uint16

	KeyBuffer []uint16

	Halted bool

	// Output is where MMIO writes (0xFF00, 0xFF01) are sent.
	// If nil, os.Stdout is used.
	Output io.Writer

	Disk        *vfs.VirtualDisk
	StoragePath string

	// VFS Directory Listing State
	VfsDirKeys  []string
	VfsDirIndex int

	// VFS MMIO parameter registers (0xFF11-0xFF15)
	vfsNamePtr  uint16
	vfsBufPtr   uint16
	vfsLength   uint16
	vfsStatus   uint16
	vfsFreeHigh uint16

	// MDU State
	mathA         uint16
	mathOp        uint16
	mathRes       uint16
	mathRemainder uint16

	CallDepth int

	Peripherals       [16]Peripheral
	PeripheralIntMask uint16
}

type CPUState struct {
	Regs               [8]uint16
	PC, SP             uint16
	TextResolutionMode uint16
	CurrentBank        uint16
	DisplayBank        uint16
	Z, N, C, IE        bool
	Waiting            bool
	InterruptPending   bool
	GraphicsEnabled    bool
	TextOverlay        bool
	BufferedMode       bool
	ColorMode8bpp      bool
	Palette            [256]uint16
	PaletteIndex       uint16
	Memory             [65536]byte
	TextVRAM           [1024]uint16
	TextVRAM_Front     [1024]uint16
	GraphicsBanks      [4][16384]byte
	GraphicsBanksFront [4][16384]byte

	// MDU State
	MathA         uint16
	MathOp        uint16
	MathRes       uint16
	MathRemainder uint16
	PeripheralIntMask uint16
}

func (c *CPU) getState() CPUState {
	return CPUState{
		Regs:               c.Regs,
		PC:                 c.PC,
		SP:                 c.SP,
		TextResolutionMode: c.TextResolutionMode,
		CurrentBank:        c.CurrentBank,
		DisplayBank:        c.DisplayBank,
		Z:                  c.Z,
		N:                  c.N,
		C:                  c.C,
		IE:                 c.IE,
		Waiting:            c.Waiting,
		InterruptPending:   c.InterruptPending,
		GraphicsEnabled:    c.GraphicsEnabled,
		TextOverlay:        c.TextOverlay,
		BufferedMode:       c.BufferedMode,
		ColorMode8bpp:      c.ColorMode8bpp,
		Palette:            c.Palette,
		PaletteIndex:       c.PaletteIndex,
		Memory:             c.Memory,
		TextVRAM:           c.TextVRAM,
		TextVRAM_Front:     c.TextVRAM_Front,
		GraphicsBanks:      c.GraphicsBanks,
		GraphicsBanksFront: c.GraphicsBanksFront,
		MathA:              c.mathA,
		MathOp:             c.mathOp,
		MathRes:            c.mathRes,
		MathRemainder:      c.mathRemainder,
		PeripheralIntMask:  c.PeripheralIntMask,
	}
}

func (c *CPU) restoreState(state CPUState) {
	c.Regs = state.Regs
	c.PC = state.PC
	c.SP = state.SP
	c.TextResolutionMode = state.TextResolutionMode
	c.CurrentBank = state.CurrentBank
	c.DisplayBank = state.DisplayBank
	c.Z = state.Z
	c.N = state.N
	c.C = state.C
	c.IE = state.IE
	c.Waiting = state.Waiting
	c.InterruptPending = state.InterruptPending
	c.GraphicsEnabled = state.GraphicsEnabled
	c.TextOverlay = state.TextOverlay
	c.BufferedMode = state.BufferedMode
	c.ColorMode8bpp = state.ColorMode8bpp
	c.Palette = state.Palette
	c.PaletteIndex = state.PaletteIndex
	c.Memory = state.Memory
	c.TextVRAM = state.TextVRAM
	c.TextVRAM_Front = state.TextVRAM_Front
	c.GraphicsBanks = state.GraphicsBanks
	c.GraphicsBanksFront = state.GraphicsBanksFront
	c.mathA = state.MathA
	c.mathOp = state.MathOp
	c.mathRes = state.MathRes
	c.mathRemainder = state.MathRemainder
	c.PeripheralIntMask = state.PeripheralIntMask
}

func (c *CPU) MountPeripheral(slot uint8, p Peripheral) {
	if slot < 16 {
		c.Peripherals[slot] = p
	}
}

func (c *CPU) TriggerPeripheralInterrupt(slot uint8) {
	if slot < 16 {
		c.PeripheralIntMask |= (1 << slot)
		c.TriggerInterrupt()
	}
}

func (c *CPU) outputSink() io.Writer {
	if c.Output != nil {
		return c.Output
	}
	return os.Stdout
}

// pico8Palette contains the RGB565 equivalents of the Pico-8-inspired 16-color palette.
var pico8Palette = [16]uint16{
	0x0000, // 0  Black       {0x00, 0x00, 0x00}
	0x194A, // 1  Dark Blue   {0x1D, 0x2B, 0x53}
	0x792A, // 2  Dark Purple {0x7E, 0x25, 0x53}
	0x042A, // 3  Dark Green  {0x00, 0x87, 0x51}
	0xAA86, // 4  Brown       {0xAB, 0x52, 0x36}
	0x5AA9, // 5  Dark Gray   {0x5F, 0x57, 0x4F}
	0xC618, // 6  Light Gray  {0xC2, 0xC3, 0xC7}
	0xFF9D, // 7  White       {0xFF, 0xF1, 0xE8}
	0xF809, // 8  Red         {0xFF, 0x00, 0x4D}
	0xFD00, // 9  Orange      {0xFF, 0xA3, 0x00}
	0xFF64, // 10 Yellow      {0xFF, 0xEC, 0x27}
	0x0726, // 11 Green       {0x00, 0xE4, 0x36}
	0x2D7F, // 12 Blue        {0x29, 0xAD, 0xFF}
	0x83B3, // 13 Indigo      {0x83, 0x76, 0x9C}
	0xFBB5, // 14 Pink        {0xFF, 0x77, 0xA8}
	0xFE75, // 15 Peach       {0xFF, 0xCC, 0xAA}
}

// NewCPU creates a new CPU instance. An optional storagePath may be provided;
// if non-empty, existing files from that directory are loaded into the VFS on startup.
func NewCPU(storagePath ...string) *CPU {
	c := &CPU{
		SP:          0xFFFE,
		TextOverlay: true,
		Disk:        vfs.NewVirtualDisk(),
	}
	for i, v := range pico8Palette {
		c.Palette[i] = v
	}
	if len(storagePath) > 0 && storagePath[0] != "" {
		c.StoragePath = storagePath[0]
		_ = c.Disk.LoadFrom(storagePath[0]) // best-effort bootstrap; ignore errors on first run
	}
	return c
}

func (c *CPU) reg(idx uint16) *uint16 {
	if idx < 8 {
		return &c.Regs[idx]
	}
	return &c.Regs[0] // Fallback
}

func (c *CPU) updateFlags(result uint16) {
	c.Z = result == 0
	c.N = (result & 0x8000) != 0
}

func (c *CPU) TriggerInterrupt() {
	c.InterruptPending = true
}

func (c *CPU) PushKey(val uint16) {
	c.KeyBuffer = append(c.KeyBuffer, val)
	c.TriggerInterrupt()
}

// Read16 reads a little-endian uint16 from addr and addr+1.
// MMIO registers (0xFF00-0xFF1F) are read from dedicated struct fields.
func (c *CPU) Read16(addr uint16) uint16 {
	if addr >= 0xFC00 && addr <= 0xFCFF {
		slot := uint8((addr - 0xFC00) / 16)
		offset := (addr - 0xFC00) % 16
		if c.Peripherals[slot] != nil {
			return c.Peripherals[slot].Read16(offset)
		}
		return 0
	}

	switch addr {
	case 0xFF09:
		return c.PeripheralIntMask
	case 0xFF04: // Keyboard buffer – return whole key code atomically
		if len(c.KeyBuffer) > 0 {
			val := c.KeyBuffer[0]
			c.KeyBuffer = c.KeyBuffer[1:]
			return val
		}
		return 0
	case 0xFF07:
		return c.PaletteIndex
	case 0xFF08:
		return c.Palette[c.PaletteIndex]
	case 0xFF11:
		return c.vfsNamePtr
	case 0xFF12:
		return c.vfsBufPtr
	case 0xFF13:
		return c.vfsLength
	case 0xFF14:
		return c.vfsStatus
	case 0xFF15:
		return c.vfsFreeHigh
	case 0xFF22:
		return c.mathRes
	case 0xFF24:
		return c.mathRemainder
	}
	lo := uint16(c.ReadByte(addr))
	hi := uint16(c.ReadByte(addr + 1))
	return lo | (hi << 8)
}

// Write16 writes a little-endian uint16 to addr and addr+1.
// MMIO registers occupy 0xFF00-0xFF1F; addresses above that (0xFF20+) are normal RAM
// and are used for the stack (which starts at 0xFFFE and grows down).
func (c *CPU) Write16(addr uint16, val uint16) {
	if addr >= 0xFC00 && addr <= 0xFCFF {
		slot := uint8((addr - 0xFC00) / 16)
		offset := (addr - 0xFC00) % 16
		if c.Peripherals[slot] != nil {
			c.Peripherals[slot].Write16(offset, val)
		}
		return
	}

	if addr >= 0xFF00 && addr <= 0xFF2F {
		c.handleMMIOWrite16(addr, val)
		return
	}
	c.WriteByte(addr, byte(val&0xFF))
	c.WriteByte(addr+1, byte(val>>8))
}

// ReadByte reads a single byte from addr, with MMIO and VRAM interception.
func (c *CPU) ReadByte(addr uint16) byte {
	// Expansion Bus: 0xFC00-0xFCFF
	if addr >= 0xFC00 && addr <= 0xFCFF {
		val := c.Read16(addr & 0xFFFE)
		if addr%2 == 0 {
			return byte(val & 0xFF)
		}
		return byte(val >> 8)
	}

	// Text VRAM: 0xF000-0xF7FF (1024 uint16 cells × 2 bytes each)
	if addr >= 0xF000 && addr <= 0xF7FF {
		offset := addr - 0xF000
		wordIndex := offset / 2
		if offset%2 == 0 {
			return byte(c.TextVRAM[wordIndex] & 0xFF)
		}
		return byte(c.TextVRAM[wordIndex] >> 8)
	}
	// Graphics banks: 0x8000-0xBFFF
	if addr >= 0x8000 && addr <= 0xBFFF {
		return c.GraphicsBanks[c.CurrentBank][addr-0x8000]
	}
	// MMIO reads
	if addr == 0xFF03 {
		return byte(c.TextResolutionMode)
	}
	if addr == 0xFF04 {
		if len(c.KeyBuffer) > 0 {
			val := c.KeyBuffer[0]
			c.KeyBuffer = c.KeyBuffer[1:]
			return byte(val)
		}
		return 0
	}
	if addr == 0xFF05 {
		var v byte
		if c.TextOverlay {
			v |= 0x01
		}
		if c.GraphicsEnabled {
			v |= 0x02
		}
		if c.BufferedMode {
			v |= 0x04
		}
		if c.ColorMode8bpp {
			v |= 0x08
		}
		return v
	}
	return c.Memory[addr]
}

// WriteByte writes a single byte to addr, with MMIO and VRAM interception.
func (c *CPU) WriteByte(addr uint16, val byte) {
	// Expansion Bus: 0xFC00-0xFCFF
	if addr >= 0xFC00 && addr <= 0xFCFF {
		wordAddr := addr & 0xFFFE
		current := c.Read16(wordAddr)
		var newVal uint16
		if addr%2 == 0 {
			newVal = (current & 0xFF00) | uint16(val)
		} else {
			newVal = (current & 0x00FF) | (uint16(val) << 8)
		}
		c.Write16(wordAddr, newVal)
		return
	}

	// Text VRAM: 0xF000-0xF7FF
	if addr >= 0xF000 && addr <= 0xF7FF {
		offset := addr - 0xF000
		wordIndex := offset / 2
		if offset%2 == 0 {
			c.TextVRAM[wordIndex] = (c.TextVRAM[wordIndex] & 0xFF00) | uint16(val)
		} else {
			c.TextVRAM[wordIndex] = (c.TextVRAM[wordIndex] & 0x00FF) | (uint16(val) << 8)
		}
		return
	}
	// Graphics banks: 0x8000-0xBFFF
	if addr >= 0x8000 && addr <= 0xBFFF {
		c.GraphicsBanks[c.CurrentBank][addr-0x8000] = val
		return
	}
	// MMIO byte writes (for completeness; 16-bit MMIO handled via handleMMIOWrite16)
	// Only 0xFF00-0xFF1F are MMIO; above that is normal RAM (used by the stack).
	if addr >= 0xFF00 && addr <= 0xFF2F {
		c.handleMMIOWrite16(addr, uint16(val))
		return
	}
	c.Memory[addr] = val
}

func (c *CPU) handleMMIOWrite16(addr uint16, val uint16) {
	switch addr {
	case 0xFF00:
		fmt.Fprintf(c.outputSink(), "%c", val)
	case 0xFF01:
		// TODO: not sure we need a special case for 0xFF01 vs 0xFF00
		fmt.Fprintf(c.outputSink(), "%d", val)
	case 0xFF02:
		c.CurrentBank = val & 0x03
	case 0xFF03:
		c.TextResolutionMode = val & 0x01
	case 0xFF05:
		c.TextOverlay = (val & 0x01) != 0
		c.GraphicsEnabled = (val & 0x02) != 0
		c.BufferedMode = (val & 0x04) != 0
		c.ColorMode8bpp = (val & 0x08) != 0
	case 0xFF06:
		bankToFlip := val & 0x03
		c.DisplayBank = bankToFlip
		copy(c.GraphicsBanksFront[bankToFlip][:], c.GraphicsBanks[bankToFlip][:])
		copy(c.TextVRAM_Front[:], c.TextVRAM[:])
	case 0xFF07:
		c.PaletteIndex = val & 0xFF
	case 0xFF08:
		c.Palette[c.PaletteIndex] = val
	case 0xFF09:
		c.PeripheralIntMask &= ^val
	case 0xFF10:
		c.handleVFSCommand(val)
	case 0xFF11:
		c.vfsNamePtr = val
	case 0xFF12:
		c.vfsBufPtr = val
	case 0xFF13:
		c.vfsLength = val
	case 0xFF14:
		c.vfsStatus = val
	case 0xFF15:
		c.vfsFreeHigh = val
	case 0xFF20:
		c.mathA = val
	case 0xFF23:
		c.mathOp = val
	case 0xFF21:
		// Trigger Calculation
		if c.mathOp == 0 { // Multiplication Q8.8
			// Use 32-bit math to prevent overflow
			res32 := int32(int16(c.mathA)) * int32(int16(val))
			// Result is (A*B) >> 8
			c.mathRes = uint16(res32 >> 8)
		} else { // Division Q8.8
			if val == 0 {
				c.mathRes = 0xFFFF // Division by zero error state
			} else {
				// (A << 8) / B
				dividend := int32(int16(c.mathA)) << 8
				divisor := int32(int16(val))
				c.mathRes = uint16(dividend / divisor)
				c.mathRemainder = uint16(dividend % divisor)
			}
		}
	}
}

// ReadMem reads a 16-bit value from addr (for backward compatibility).
// Deprecated: use Read16 directly.
func (c *CPU) ReadMem(addr uint16) uint16 {
	return c.Read16(addr)
}

// WriteMem writes a 16-bit value to addr (for backward compatibility).
// Deprecated: use Write16 directly.
func (c *CPU) WriteMem(addr uint16, val uint16) {
	c.Write16(addr, val)
}

func (c *CPU) ReadStringFromRAM(ptr uint16) (string, error) {
	var chars []byte
	for i := uint16(0); i < 17; i++ {
		if int(ptr)+int(i) >= len(c.Memory) {
			return "", errors.New("memory access out of bounds")
		}
		val := c.Memory[ptr+i]
		if val == 0 {
			return string(chars), nil
		}
		chars = append(chars, val)
	}
	return "", errors.New("string too long or missing null terminator")
}

func (c *CPU) writeStringToRAM(ptr uint16, s string) error {
	for i := 0; i < len(s); i++ {
		if int(ptr)+i >= len(c.Memory) {
			return errors.New("memory access out of bounds")
		}
		c.Memory[ptr+uint16(i)] = s[i]
	}
	if int(ptr)+len(s) >= len(c.Memory) {
		return errors.New("memory access out of bounds")
	}
	c.Memory[ptr+uint16(len(s))] = 0
	return nil
}

func (c *CPU) copyToRAM(dst uint16, src []byte) error {
	if int(dst)+len(src) > len(c.Memory) {
		return errors.New("memory access out of bounds")
	}
	copy(c.Memory[dst:], src)
	return nil
}

func (c *CPU) copyFromRAM(src uint16, length uint16) ([]byte, error) {
	if int(src)+int(length) > len(c.Memory) {
		return nil, errors.New("memory access out of bounds")
	}
	data := make([]byte, length)
	copy(data, c.Memory[src:src+length])
	return data, nil
}

func (c *CPU) handleVFSCommand(val uint16) {
	// Read VFS parameter registers from dedicated struct fields.
	filenamePtr := c.vfsNamePtr
	bufferPtr := c.vfsBufPtr

	switch val {
	case 1: // Read
		filename, err := c.ReadStringFromRAM(filenamePtr)
		if err != nil {
			c.vfsStatus = 3 // Invalid Name
			return
		}
		data, err := c.Disk.Read(filename)
		if err != nil {
			if errors.Is(err, vfs.ErrFileNotFound) {
				c.vfsStatus = 1
			} else {
				c.vfsStatus = 3
			}
			return
		}
		err = c.copyToRAM(bufferPtr, data)
		if err != nil {
			c.vfsStatus = 4 // Out of Bounds
			return
		}
		c.vfsStatus = 0 // Success

	case 2: // Write
		filename, err := c.ReadStringFromRAM(filenamePtr)
		if err != nil {
			c.vfsStatus = 3 // Invalid Name
			return
		}
		data, err := c.copyFromRAM(bufferPtr, c.vfsLength)
		if err != nil {
			c.vfsStatus = 4 // Out of Bounds
			return
		}
		err = c.Disk.Write(filename, data)
		if err != nil {
			if errors.Is(err, vfs.ErrQuotaExceeded) {
				c.vfsStatus = 2
			} else {
				c.vfsStatus = 3
			}
			return
		}
		c.vfsStatus = 0 // Success

	case 3: // Get Size
		filename, err := c.ReadStringFromRAM(filenamePtr)
		if err != nil {
			c.vfsStatus = 3 // Invalid Name
			return
		}
		size, err := c.Disk.Size(filename)
		if err != nil {
			if errors.Is(err, vfs.ErrFileNotFound) {
				c.vfsStatus = 1
			} else {
				c.vfsStatus = 3
			}
			return
		}
		c.vfsLength = uint16(size)
		c.vfsStatus = 0 // Success

	case 4: // Delete
		filename, err := c.ReadStringFromRAM(filenamePtr)
		if err != nil {
			c.vfsStatus = 3 // Invalid Name
			return
		}
		err = c.Disk.Delete(filename)
		if err != nil {
			if errors.Is(err, vfs.ErrFileNotFound) {
				c.vfsStatus = 1
			} else {
				c.vfsStatus = 3
			}
			return
		}
		c.vfsStatus = 0 // Success

	case 5: // List
		if c.VfsDirKeys == nil {
			c.VfsDirKeys = c.Disk.List()
			c.VfsDirIndex = 0
		}

		if c.VfsDirIndex >= len(c.VfsDirKeys) {
			c.vfsStatus = 5    // DirEnd
			c.VfsDirKeys = nil // Reset
			return
		}

		filename := c.VfsDirKeys[c.VfsDirIndex]
		err := c.writeStringToRAM(bufferPtr, filename)
		if err != nil {
			c.vfsStatus = 4 // Out of bounds
			return
		}

		c.VfsDirIndex++
		c.vfsStatus = 0 // Success

	case 6: // FreeSpace
		free := c.Disk.FreeSpace()
		c.vfsLength = uint16(free & 0xFFFF)
		c.vfsFreeHigh = uint16((free >> 16) & 0xFFFF)
		c.vfsStatus = 0 // Success

	case 7: // GetMeta
		filename, err := c.ReadStringFromRAM(filenamePtr)
		if err != nil {
			c.vfsStatus = 3 // Invalid Name
			return
		}
		created, modified, err := c.Disk.GetMeta(filename)
		if err != nil {
			c.vfsStatus = 1 // Not Found
			return
		}

		// 12 uint16 values = 24 bytes: [C_Year, C_Month, C_Day, C_Hour, C_Min, C_Sec, M_Year, M_Month, M_Day, M_Hour, M_Min, M_Sec]
		meta := []uint16{
			uint16(created.Year()), uint16(created.Month()), uint16(created.Day()),
			uint16(created.Hour()), uint16(created.Minute()), uint16(created.Second()),
			uint16(modified.Year()), uint16(modified.Month()), uint16(modified.Day()),
			uint16(modified.Hour()), uint16(modified.Minute()), uint16(modified.Second()),
		}

		// Write as bytes (LE uint16 pairs)
		metaBytes := make([]byte, len(meta)*2)
		for i, v := range meta {
			metaBytes[i*2] = byte(v & 0xFF)
			metaBytes[i*2+1] = byte(v >> 8)
		}
		err = c.copyToRAM(bufferPtr, metaBytes)
		if err != nil {
			c.vfsStatus = 4 // Out of Bounds
			return
		}
		c.vfsStatus = 0 // Success

	case 8: // ExecWait
		filename, err := c.ReadStringFromRAM(filenamePtr)
		if err != nil {
			c.vfsStatus = 3 // Invalid Name
			return
		}

		// 1. Read the target executable
		binData, err := c.Disk.Read(filename)
		if err != nil {
			c.vfsStatus = 1 // Not Found
			return
		}

		// 2. Generate swap filename
		swapName := fmt.Sprintf(".swap_%d.sys", c.CallDepth)

		// 3. Serialize current state
		state := c.getState()
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(state); err != nil {
			c.vfsStatus = 4 // Treat as internal error
			return
		}

		// 4. Write swap file
		if err := c.Disk.Write(swapName, buf.Bytes()); err != nil {
			if errors.Is(err, vfs.ErrQuotaExceeded) {
				c.vfsStatus = 2
			} else {
				c.vfsStatus = 3
			}
			return
		}

		// 5. Context Switch
		c.CallDepth++

		// Clear Memory
		c.Memory = [65536]byte{}

		// Load binary
		if len(binData) > len(c.Memory) {
			c.vfsStatus = 4 // Out of Bounds
			return
		}
		copy(c.Memory[:], binData)

		// Reset Registers
		c.PC = 0
		c.SP = 0xFFFE
		c.Z = false
		c.N = false
		c.C = false
		c.IE = false
		c.Waiting = false
		c.InterruptPending = false

		// Reset Video
		c.TextResolutionMode = 0
		c.GraphicsEnabled = false
		c.TextOverlay = true
		c.BufferedMode = false

		c.vfsStatus = 0 // Success
	}
}

func (c *CPU) Step() {
	if c.Halted {
		return
	}

	for _, p := range c.Peripherals {
		if p != nil {
			p.Step()
		}
	}

	if c.InterruptPending && c.IE {
		c.InterruptPending = false
		c.IE = false
		c.Waiting = false
		c.SP -= 2
		c.Write16(c.SP, c.PC)
		c.PC = 0x0010
	}

	if c.Waiting {
		return
	}

	instr := c.Read16(c.PC)
	c.PC += 2

	opcode := (instr >> 10) & 0x3F
	regA := (instr >> 7) & 0x07
	regB := (instr >> 4) & 0x07

	switch opcode {
	case OpHLT:
		if c.CallDepth > 0 {
			c.CallDepth--
			swapName := fmt.Sprintf(".swap_%d.sys", c.CallDepth)

			// Read swap file
			swapData, err := c.Disk.Read(swapName)
			if err != nil {
				// Fatal error if swap is missing
				c.Halted = true
				return
			}

			// Deserialize
			var state CPUState
			buf := bytes.NewBuffer(swapData)
			dec := gob.NewDecoder(buf)
			if err := dec.Decode(&state); err != nil {
				c.Halted = true
				return
			}

			c.restoreState(state)

			// Cleanup
			_ = c.Disk.Delete(swapName)

			c.Halted = false
		} else {
			c.Halted = true
		}

	case OpNOP:
		// No operation.

	case OpJNC:
		target := c.Read16(c.PC)
		c.PC += 2
		if !c.C { // Jump if Carry flag is false (No Carry)
			c.PC = target
		}

	case OpLDI:
		imm := c.Read16(c.PC)
		c.PC += 2
		*c.reg(regA) = imm

	case OpMOV:
		*c.reg(regA) = *c.reg(regB)

	case OpLD:
		addr := *c.reg(regB)
		*c.reg(regA) = c.Read16(addr)

	case OpADD:
		valA := uint32(*c.reg(regA))
		valB := uint32(*c.reg(regB))
		res32 := valA + valB
		result := uint16(res32)
		c.C = res32 > 0xFFFF
		*c.reg(regA) = result
		c.updateFlags(result)

	case OpSUB:
		valA := *c.reg(regA)
		valB := *c.reg(regB)
		result := valA - valB
		c.C = valA < valB
		*c.reg(regA) = result
		c.updateFlags(result)

	case OpAND:
		result := *c.reg(regA) & *c.reg(regB)
		*c.reg(regA) = result
		c.updateFlags(result)

	case OpOR:
		result := *c.reg(regA) | *c.reg(regB)
		*c.reg(regA) = result
		c.updateFlags(result)

	case OpXOR:
		result := *c.reg(regA) ^ *c.reg(regB)
		*c.reg(regA) = result
		c.updateFlags(result)

	case OpNOT:
		result := ^*c.reg(regA)
		*c.reg(regA) = result
		c.updateFlags(result)

	case OpSHL:
		result := *c.reg(regA) << *c.reg(regB)
		*c.reg(regA) = result
		c.updateFlags(result)

	case OpSHR:
		result := *c.reg(regA) >> *c.reg(regB)
		*c.reg(regA) = result
		c.updateFlags(result)

	case OpJMP:
		target := c.Read16(c.PC)
		c.PC += 2
		c.PC = target

	case OpJZ:
		target := c.Read16(c.PC)
		c.PC += 2
		if c.Z {
			c.PC = target
		}

	case OpJNZ:
		target := c.Read16(c.PC)
		c.PC += 2
		if !c.Z {
			c.PC = target
		}

	case OpJN:
		target := c.Read16(c.PC)
		c.PC += 2
		if c.N {
			c.PC = target
		}

	case OpJC:
		target := c.Read16(c.PC)
		c.PC += 2
		if c.C {
			c.PC = target
		}

	case OpPUSH:
		c.SP -= 2
		c.Write16(c.SP, *c.reg(regA))

	case OpPOP:
		*c.reg(regA) = c.Read16(c.SP)
		c.SP += 2

	case OpCALL:
		target := c.Read16(c.PC)
		c.PC += 2
		c.SP -= 2
		c.Write16(c.SP, c.PC)
		c.PC = target

	case OpRET:
		c.PC = c.Read16(c.SP)
		c.SP += 2

	case OpEI:
		c.IE = true

	case OpDI:
		c.IE = false

	case OpRETI:
		c.PC = c.Read16(c.SP)
		c.SP += 2
		c.IE = true

	case OpWFI:
		c.Waiting = true

	case OpLDSP:
		*c.reg(regA) = c.SP

	case OpSTSP:
		c.SP = *c.reg(regA)

	case OpMUL:
		result := *c.reg(regA) * *c.reg(regB)
		*c.reg(regA) = result
		c.updateFlags(result)

	case OpDIV:
		divisor := *c.reg(regB)
		if divisor == 0 {
			*c.reg(regA) = 0
			c.updateFlags(0)
		} else {
			result := *c.reg(regA) / divisor
			*c.reg(regA) = result
			c.updateFlags(result)
		}

	case OpIDIV:
		divisor := int16(*c.reg(regB))
		if divisor == 0 {
			*c.reg(regA) = 0
			c.updateFlags(0)
		} else {
			result := int16(*c.reg(regA)) / divisor
			*c.reg(regA) = uint16(result)
			c.updateFlags(uint16(result))
		}

	case OpST:
		addr := *c.reg(regA)
		val := *c.reg(regB)
		c.Write16(addr, val)

	case OpFILL:
		regC := (instr >> 1) & 0x07
		startAddr := *c.reg(regA)
		count := *c.reg(regB)
		val := *c.reg(regC)
		for i := uint16(0); i < count; i++ {
			c.Write16(startAddr+i*2, val)
		}

	case OpCOPY:
		regC := (instr >> 1) & 0x07
		srcAddr := *c.reg(regA)
		dstAddr := *c.reg(regB)
		count := *c.reg(regC)

		if srcAddr < dstAddr && srcAddr+count*2 > dstAddr {
			// Overlap, copy backwards
			for i := count; i > 0; i-- {
				val := c.Read16(srcAddr + (i-1)*2)
				c.Write16(dstAddr+(i-1)*2, val)
			}
		} else {
			// No overlap or dst < src, copy forwards
			for i := uint16(0); i < count; i++ {
				val := c.Read16(srcAddr + i*2)
				c.Write16(dstAddr+i*2, val)
			}
		}

	case OpLDB:
		addr := *c.reg(regB)
		*c.reg(regA) = uint16(c.ReadByte(addr))

	case OpSTB:
		addr := *c.reg(regA)
		c.WriteByte(addr, byte(*c.reg(regB)&0xFF))
	}
}

func (c *CPU) Run() {
	for !c.Halted {
		c.Step()
	}
}

func EncodeInstruction(opcode, regA, regB, regC uint16) uint16 {
	return (opcode << 10) | ((regA & 0x07) << 7) | ((regB & 0x07) << 4) | ((regC & 0x07) << 1)
}

func (c *CPU) RunUntilDone() {
	for {
		if c.Halted || c.Waiting {
			break
		}
		c.Step()
	}
}
