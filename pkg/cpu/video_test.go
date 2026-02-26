package cpu

import "testing"

// TestPaletteMMIO verifies the CLUT index/data MMIO ports (Ticket 1).
func TestPaletteMMIO(t *testing.T) {
	c := NewCPU()

	// Write index 5, then write color 0x1234
	c.Write16(0xFF07, 0x05)
	c.Write16(0xFF08, 0x1234)

	if c.Palette[5] != 0x1234 {
		t.Errorf("Palette[5]: expected 0x1234, got 0x%04X", c.Palette[5])
	}
	if got := c.Read16(0xFF08); got != 0x1234 {
		t.Errorf("Read16(0xFF08): expected 0x1234, got 0x%04X", got)
	}
	if got := c.Read16(0xFF07); got != 0x05 {
		t.Errorf("Read16(0xFF07): expected 0x05, got 0x%04X", got)
	}

	// Palette index should be masked to 0-255
	c.Write16(0xFF07, 0x1FF) // 511 → should clamp to 0xFF
	if c.PaletteIndex != 0xFF {
		t.Errorf("PaletteIndex mask: expected 0xFF, got 0x%04X", c.PaletteIndex)
	}

	// Default Pico-8 palette: index 0 should be black (0x0000)
	if c.Palette[0] != 0x0000 {
		t.Errorf("Palette[0] (Black): expected 0x0000, got 0x%04X", c.Palette[0])
	}
}

// TestColorMode8bpp verifies the 8bpp flag in 0xFF05 (Ticket 2).
func TestColorMode8bpp(t *testing.T) {
	c := NewCPU()

	// 0x0A = 0b0000_1010 → GraphicsEnabled (bit1) + ColorMode8bpp (bit3)
	c.Write16(0xFF05, 0x0A)

	if !c.GraphicsEnabled {
		t.Error("GraphicsEnabled: expected true")
	}
	if !c.ColorMode8bpp {
		t.Error("ColorMode8bpp: expected true")
	}

	got := c.ReadByte(0xFF05)
	if got != 0x0A {
		t.Errorf("ReadByte(0xFF05): expected 0x0A, got 0x%02X", got)
	}
}

// TestGetFramebufferRGBA_8bpp verifies 8bpp framebuffer decoding (Ticket 3).
func TestGetFramebufferRGBA_8bpp(t *testing.T) {
	c := NewCPU()
	c.ColorMode8bpp = true

	// Set a known color at palette index 1
	c.Palette[1] = 0xFFFF // full white in RGB565

	// Fill first 5 bytes of VRAM (bank 0) with index 1
	for i := 0; i < 5; i++ {
		c.GraphicsBanks[0][i] = 0x01
	}

	pixels := c.GetFramebufferRGBA()

	// RGB565 0xFFFF → r5=31, g6=63, b5=31 → r=(31<<3)|(31>>2)=0xFF, g=(63<<2)|(63>>4)=0xFF, b=0xFF
	for pix := 0; pix < 5; pix++ {
		base := pix * 4
		if pixels[base+0] != 0xFF || pixels[base+1] != 0xFF || pixels[base+2] != 0xFF || pixels[base+3] != 0xFF {
			t.Errorf("pixel %d: expected RGBA(0xFF,0xFF,0xFF,0xFF), got (%d,%d,%d,%d)",
				pix, pixels[base+0], pixels[base+1], pixels[base+2], pixels[base+3])
		}
	}
}

// TestGetFramebufferRGBA_4bpp verifies 4bpp framebuffer decoding (Ticket 3).
func TestGetFramebufferRGBA_4bpp(t *testing.T) {
	c := NewCPU()
	// 4bpp mode (default)

	// Set palette index 1 to a known color: RGB565 0xF800 = pure red
	c.Palette[1] = 0xF800

	// Encode 4 pixels of color index 1 into word 0 of VRAM (all nibbles = 1)
	// word = 0x1111 → LE bytes: lo=0x11, hi=0x11
	c.GraphicsBanks[0][0] = 0x11
	c.GraphicsBanks[0][1] = 0x11

	pixels := c.GetFramebufferRGBA()

	// RGB565 0xF800 → r5=31, g6=0, b5=0 → r=0xFF, g=0, b=0
	for pix := 0; pix < 4; pix++ {
		base := pix * 4
		if pixels[base+0] != 0xFF || pixels[base+1] != 0x00 || pixels[base+2] != 0x00 || pixels[base+3] != 0xFF {
			t.Errorf("pixel %d: expected RGBA(0xFF,0,0,0xFF), got (%d,%d,%d,%d)",
				pix, pixels[base+0], pixels[base+1], pixels[base+2], pixels[base+3])
		}
	}
}

// TestGetFramebufferImage verifies that GetFramebufferImage wraps the RGBA slice correctly.
func TestGetFramebufferImage(t *testing.T) {
	c := NewCPU()
	img := c.GetFramebufferImage()
	if img.Rect.Dx() != 128 || img.Rect.Dy() != 128 {
		t.Errorf("image size: expected 128x128, got %dx%d", img.Rect.Dx(), img.Rect.Dy())
	}
	if img.Stride != 128*4 {
		t.Errorf("image stride: expected %d, got %d", 128*4, img.Stride)
	}
}

// TestPaletteStateRoundtrip verifies that Palette and PaletteIndex survive a
// getState/restoreState round-trip (used by vfs_exec_wait).
func TestPaletteStateRoundtrip(t *testing.T) {
	c := NewCPU()
	c.Palette[7] = 0xABCD
	c.PaletteIndex = 7
	c.ColorMode8bpp = true

	state := c.getState()

	c2 := NewCPU()
	c2.restoreState(state)

	if c2.Palette[7] != 0xABCD {
		t.Errorf("Palette[7] after restore: expected 0xABCD, got 0x%04X", c2.Palette[7])
	}
	if c2.PaletteIndex != 7 {
		t.Errorf("PaletteIndex after restore: expected 7, got %d", c2.PaletteIndex)
	}
	if !c2.ColorMode8bpp {
		t.Error("ColorMode8bpp after restore: expected true")
	}
}
