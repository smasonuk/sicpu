package cpu

import (
	"image"
	"image/png"
	"os"
)

// rgb565ToRGBA converts an RGB565 color to four RGBA bytes using accurate bit-expansion.
func rgb565ToRGBA(val uint16) (r, g, b, a byte) {
	r5 := byte((val >> 11) & 0x1F)
	g6 := byte((val >> 5) & 0x3F)
	b5 := byte(val & 0x1F)
	r = (r5 << 3) | (r5 >> 2)
	g = (g6 << 2) | (g6 >> 4)
	b = (b5 << 3) | (b5 >> 2)
	a = 0xFF
	return
}

// GetFramebufferRGBA decodes the current graphics bank into a 128×128 RGBA8888
// byte slice (length 128*128*4 = 65536). It respects BufferedMode, DisplayBank,
// CurrentBank, and ColorMode8bpp.
func (c *CPU) GetFramebufferRGBA() []byte {
	var bankData *[16384]byte
	if c.BufferedMode {
		bankData = &c.GraphicsBanksFront[c.DisplayBank]
	} else {
		bankData = &c.GraphicsBanks[c.CurrentBank]
	}

	pixels := make([]byte, 128*128*4)

	if c.ColorMode8bpp {
		// 8bpp: one byte per pixel, 16384 pixels total
		for i := 0; i < 16384; i++ {
			colorIdx := bankData[i]
			r, g, b, a := rgb565ToRGBA(c.Palette[colorIdx])
			pixels[i*4+0] = r
			pixels[i*4+1] = g
			pixels[i*4+2] = b
			pixels[i*4+3] = a
		}
	} else {
		// 4bpp: one word (2 bytes) holds 4 nibbles → 4 pixels
		for wordIdx := 0; wordIdx < 4096; wordIdx++ {
			lo := uint16(bankData[wordIdx*2])
			hi := uint16(bankData[wordIdx*2+1])
			word := lo | (hi << 8)
			for sub := 0; sub < 4; sub++ {
				colorIdx := (word >> (sub * 4)) & 0xF
				r, g, b, a := rgb565ToRGBA(c.Palette[colorIdx])
				pixelIdx := (wordIdx*4 + sub) * 4
				pixels[pixelIdx+0] = r
				pixels[pixelIdx+1] = g
				pixels[pixelIdx+2] = b
				pixels[pixelIdx+3] = a
			}
		}
	}

	return pixels
}

// GetFramebufferImage returns the current graphics bank as an *image.RGBA.
func (c *CPU) GetFramebufferImage() *image.RGBA {
	pix := c.GetFramebufferRGBA()
	return &image.RGBA{
		Pix:    pix,
		Stride: 128 * 4,
		Rect:   image.Rect(0, 0, 128, 128),
	}
}

// SaveScreenshot encodes the current framebuffer as a PNG and writes it to filename.
func (c *CPU) SaveScreenshot(filename string) error {
	img := c.GetFramebufferImage()
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
