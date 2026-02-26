package cpu

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"gocpu/pkg/vfs"
)

// humanReadableState is the JSON-serializable snapshot of CPU control state.
type humanReadableState struct {
	Regs               [8]uint16      `json:"regs"`
	PC                 uint16         `json:"pc"`
	SP                 uint16         `json:"sp"`
	Z                  bool           `json:"z"`
	N                  bool           `json:"n"`
	C                  bool           `json:"c"`
	IE                 bool           `json:"ie"`
	Waiting            bool           `json:"waiting"`
	Halted             bool           `json:"halted"`
	InterruptPending   bool           `json:"interrupt_pending"`
	CallDepth          int            `json:"call_depth"`
	PeripheralIntMask  uint16         `json:"peripheral_int_mask"`
	GraphicsEnabled    bool           `json:"graphics_enabled"`
	TextOverlay        bool           `json:"text_overlay"`
	BufferedMode       bool           `json:"buffered_mode"`
	ColorMode8bpp      bool           `json:"color_mode_8bpp"`
	TextResolutionMode uint16         `json:"text_resolution_mode"`
	CurrentBank        uint16         `json:"current_bank"`
	DisplayBank        uint16         `json:"display_bank"`
	Palette            [256]uint16    `json:"palette"`
	PaletteIndex       uint16         `json:"palette_index"`
	MountedPeripherals map[int]string `json:"mounted_peripherals"`
}

// vfsFileDescriptor holds per-file metadata for the VFS snapshot.
type vfsFileDescriptor struct {
	Name     string    `json:"name"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Size     int       `json:"size"`
}

// vfsMetadata is the JSON envelope for all VFS file descriptors.
type vfsMetadata struct {
	Files []vfsFileDescriptor `json:"files"`
}

// HibernateToBytes serialises the complete VM state into an in-memory ZIP archive
// and returns the raw bytes.
func (c *CPU) HibernateToBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// ── 1. cpu_state.json ──────────────────────────────────────────────────
	state := humanReadableState{
		Regs:               c.Regs,
		PC:                 c.PC,
		SP:                 c.SP,
		Z:                  c.Z,
		N:                  c.N,
		C:                  c.C,
		IE:                 c.IE,
		Waiting:            c.Waiting,
		Halted:             c.Halted,
		InterruptPending:   c.InterruptPending,
		CallDepth:          c.CallDepth,
		PeripheralIntMask:  c.PeripheralIntMask,
		GraphicsEnabled:    c.GraphicsEnabled,
		TextOverlay:        c.TextOverlay,
		BufferedMode:       c.BufferedMode,
		ColorMode8bpp:      c.ColorMode8bpp,
		TextResolutionMode: c.TextResolutionMode,
		CurrentBank:        c.CurrentBank,
		DisplayBank:        c.DisplayBank,
		Palette:            c.Palette,
		PaletteIndex:       c.PaletteIndex,
		MountedPeripherals: make(map[int]string),
	}

	for i, p := range c.Peripherals {
		if p != nil {
			state.MountedPeripherals[i] = p.Type()
		}
	}

	jsonData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal cpu_state: %w", err)
	}
	if err := writeZipEntry(zw, "cpu_state.json", jsonData); err != nil {
		return nil, err
	}

	// ── 2. memory.bin ──────────────────────────────────────────────────────
	if err := writeZipEntry(zw, "memory.bin", c.Memory[:]); err != nil {
		return nil, err
	}

	// ── 3. Graphics banks ──────────────────────────────────────────────────
	for i := 0; i < 4; i++ {
		if err := writeZipEntry(zw, fmt.Sprintf("graphics_bank_%d.bin", i), c.GraphicsBanks[i][:]); err != nil {
			return nil, err
		}
		if err := writeZipEntry(zw, fmt.Sprintf("graphics_bank_front_%d.bin", i), c.GraphicsBanksFront[i][:]); err != nil {
			return nil, err
		}
	}

	// ── 4. Text VRAM (uint16 arrays → little-endian bytes) ─────────────────
	textVRAMBytes := uint16SliceToLE(c.TextVRAM[:])
	if err := writeZipEntry(zw, "text_vram.bin", textVRAMBytes); err != nil {
		return nil, err
	}
	textVRAMFrontBytes := uint16SliceToLE(c.TextVRAM_Front[:])
	if err := writeZipEntry(zw, "text_vram_front.bin", textVRAMFrontBytes); err != nil {
		return nil, err
	}

	// ── 5. VFS snapshot ────────────────────────────────────────────────────
	if c.Disk != nil {
		c.Disk.Mu.RLock()

		meta := vfsMetadata{}
		for name, entry := range c.Disk.Files {
			meta.Files = append(meta.Files, vfsFileDescriptor{
				Name:     name,
				Created:  entry.Created,
				Modified: entry.Modified,
				Size:     len(entry.Data),
			})
		}

		metaJSON, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			c.Disk.Mu.RUnlock()
			return nil, fmt.Errorf("marshal vfs_metadata: %w", err)
		}
		if err := writeZipEntry(zw, "vfs_metadata.json", metaJSON); err != nil {
			c.Disk.Mu.RUnlock()
			return nil, err
		}

		for name, entry := range c.Disk.Files {
			if err := writeZipEntry(zw, "vfs/"+name, entry.Data); err != nil {
				c.Disk.Mu.RUnlock()
				return nil, err
			}
		}

		c.Disk.Mu.RUnlock()
	}

	// ── 6. Peripheral state bins ───────────────────────────────────────────
	for i, p := range c.Peripherals {
		if p == nil {
			continue
		}
		if sp, ok := p.(StatefulPeripheral); ok {
			data := sp.SaveState()
			if err := writeZipEntry(zw, fmt.Sprintf("peripheral_%d.bin", i), data); err != nil {
				return nil, err
			}
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}
	return buf.Bytes(), nil
}

// RestoreFromBytes deserialises a ZIP archive produced by HibernateToBytes and
// applies the saved state to the CPU.
func (c *CPU) RestoreFromBytes(data []byte) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}

	fileMap := make(map[string]*zip.File, len(r.File))
	for _, f := range r.File {
		fileMap[f.Name] = f
	}

	// ── 1. cpu_state.json ──────────────────────────────────────────────────
	jsonData, err := readZipEntry(fileMap, "cpu_state.json")
	if err != nil {
		return err
	}
	var state humanReadableState
	if err := json.Unmarshal(jsonData, &state); err != nil {
		return fmt.Errorf("unmarshal cpu_state: %w", err)
	}

	c.Regs = state.Regs
	c.PC = state.PC
	c.SP = state.SP
	c.Z = state.Z
	c.N = state.N
	c.C = state.C
	c.IE = state.IE
	c.Waiting = state.Waiting
	c.Halted = state.Halted
	c.InterruptPending = state.InterruptPending
	c.CallDepth = state.CallDepth
	c.PeripheralIntMask = state.PeripheralIntMask
	c.GraphicsEnabled = state.GraphicsEnabled
	c.TextOverlay = state.TextOverlay
	c.BufferedMode = state.BufferedMode
	c.ColorMode8bpp = state.ColorMode8bpp
	c.TextResolutionMode = state.TextResolutionMode
	c.CurrentBank = state.CurrentBank
	c.DisplayBank = state.DisplayBank
	c.Palette = state.Palette
	c.PaletteIndex = state.PaletteIndex

	// ── 2. memory.bin ──────────────────────────────────────────────────────
	if memData, err := readZipEntry(fileMap, "memory.bin"); err == nil {
		copy(c.Memory[:], memData)
	}

	// ── 3. Graphics banks ──────────────────────────────────────────────────
	for i := 0; i < 4; i++ {
		if d, err := readZipEntry(fileMap, fmt.Sprintf("graphics_bank_%d.bin", i)); err == nil {
			copy(c.GraphicsBanks[i][:], d)
		}
		if d, err := readZipEntry(fileMap, fmt.Sprintf("graphics_bank_front_%d.bin", i)); err == nil {
			copy(c.GraphicsBanksFront[i][:], d)
		}
	}

	// ── 4. Text VRAM ───────────────────────────────────────────────────────
	if raw, err := readZipEntry(fileMap, "text_vram.bin"); err == nil {
		leToUint16Slice(raw, c.TextVRAM[:])
	}
	if raw, err := readZipEntry(fileMap, "text_vram_front.bin"); err == nil {
		leToUint16Slice(raw, c.TextVRAM_Front[:])
	}

	// ── 5. VFS ─────────────────────────────────────────────────────────────
	if metaJSON, err := readZipEntry(fileMap, "vfs_metadata.json"); err == nil {
		var meta vfsMetadata
		if err := json.Unmarshal(metaJSON, &meta); err != nil {
			return fmt.Errorf("unmarshal vfs_metadata: %w", err)
		}

		metaLookup := make(map[string]vfsFileDescriptor, len(meta.Files))
		for _, fd := range meta.Files {
			metaLookup[fd.Name] = fd
		}

		if c.Disk == nil {
			c.Disk = vfs.NewVirtualDisk()
		}

		c.Disk.Mu.Lock()
		c.Disk.Files = make(map[string]*vfs.FileEntry)
		c.Disk.DirtyFiles = make(map[string]bool)
		c.Disk.UsedBytes = 0

		for fname, fd := range metaLookup {
			fileData, err := readZipEntry(fileMap, "vfs/"+fname)
			if err != nil {
				c.Disk.Mu.Unlock()
				return fmt.Errorf("restore vfs file %q: %w", fname, err)
			}
			c.Disk.Files[fname] = &vfs.FileEntry{
				Data:     fileData,
				Created:  fd.Created,
				Modified: fd.Modified,
			}
			c.Disk.DirtyFiles[fname] = true
			c.Disk.UsedBytes += len(fileData)
		}
		c.Disk.Dirty = true
		c.Disk.Mu.Unlock()
	}

	// ── 6. Peripherals ─────────────────────────────────────────────────────
	for slot, typeName := range state.MountedPeripherals {
		factory, ok := peripheralRegistry[typeName]
		if !ok {
			continue
		}
		p := factory(c, uint8(slot))
		c.Peripherals[slot] = p

		if sp, ok := p.(StatefulPeripheral); ok {
			if binData, err := readZipEntry(fileMap, fmt.Sprintf("peripheral_%d.bin", slot)); err == nil {
				if err := sp.LoadState(binData); err != nil {
					return fmt.Errorf("load peripheral %d state: %w", slot, err)
				}
			}
		}
	}

	return nil
}

// HibernateToFile writes the hibernation archive to the given file path.
func (c *CPU) HibernateToFile(path string) error {
	data, err := c.HibernateToBytes()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// RestoreFromFile reads a hibernation archive from the given file path and
// restores the VM state.
func (c *CPU) RestoreFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return c.RestoreFromBytes(data)
}

// ── helpers ────────────────────────────────────────────────────────────────

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("create zip entry %q: %w", name, err)
	}
	_, err = w.Write(data)
	return err
}

func readZipEntry(fileMap map[string]*zip.File, name string) ([]byte, error) {
	f, ok := fileMap[name]
	if !ok {
		return nil, fmt.Errorf("zip entry %q not found", name)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open zip entry %q: %w", name, err)
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func uint16SliceToLE(src []uint16) []byte {
	out := make([]byte, len(src)*2)
	for i, v := range src {
		binary.LittleEndian.PutUint16(out[i*2:], v)
	}
	return out
}

func leToUint16Slice(src []byte, dst []uint16) {
	for i := range dst {
		if i*2+1 < len(src) {
			dst[i] = binary.LittleEndian.Uint16(src[i*2:])
		}
	}
}
