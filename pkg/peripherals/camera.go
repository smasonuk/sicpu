package peripherals

import (
	"gocpu/pkg/cpu"
)

// CameraPeripheral generates a test image with geometric shapes on a dark blue/black background.
// Image is stored in CPU memory as 1 byte per pixel in RGB332 format (R:3 G:3 B:2), row-major.
//
// Registers:
//
//	0x00 (W): Command — write 1 to capture a frame into the buffer
//	0x02 (R/W): Buffer address — where in RAM to write pixel data
//	0x04 (R/W): Image width  (default 128)
//	0x06 (R/W): Image height (default 128)
//	0x08-0x0E (R): Peripheral ID — "CAMERA"
type CameraPeripheral struct {
	c    *cpu.CPU
	slot uint8

	bufAddr uint16
	width   uint16
	height  uint16
}

func NewCameraPeripheral(c *cpu.CPU, slot uint8) *CameraPeripheral {
	return &CameraPeripheral{
		c:      c,
		slot:   slot,
		width:  128,
		height: 128,
	}
}

func (cam *CameraPeripheral) Read16(offset uint16) uint16 {
	if offset >= 0x08 && offset <= 0x0E {
		return cpu.EncodePeripheralName("CAMERA", offset)
	}
	switch offset {
	case 0x00:
		return 0
	case 0x02:
		return cam.bufAddr
	case 0x04:
		return cam.width
	case 0x06:
		return cam.height
	}
	return 0
}

func (cam *CameraPeripheral) Write16(offset uint16, val uint16) {
	switch offset {
	case 0x00:
		if val == 1 {
			cam.capture()
		}
	case 0x02:
		cam.bufAddr = val
	case 0x04:
		cam.width = val
	case 0x06:
		cam.height = val
	}
}

func (cam *CameraPeripheral) Type() string { return "CameraPeripheral" }

func (cam *CameraPeripheral) Step() {}

// capture renders a test image into CPU RAM at bufAddr.
// Each pixel is 1 byte in RGB332 format: bits[7:5]=R, bits[4:2]=G, bits[1:0]=B.
func (cam *CameraPeripheral) capture() {
	w := int(cam.width)
	h := int(cam.height)
	addr := cam.bufAddr

	for y := range h {
		for x := range w {
			cam.c.WriteByte(addr, cam.pixelColor(x, y, w, h))
			addr++
		}
	}

	cam.c.TriggerPeripheralInterrupt(cam.slot)
}

// rgb332 packs 8-bit R, G, B into a single RGB332 byte: bits[7:5]=R, bits[4:2]=G, bits[1:0]=B.
func rgb332(r, g, b uint8) uint8 {
	return (r & 0xE0) | ((g >> 3) & 0x1C) | (b >> 6)
}

// pixelColor returns a single RGB332 byte for the pixel at (x, y) in an image of size (w, h).
// Shapes drawn (in order, later shapes paint over earlier ones):
//   - Background: vertical gradient from black (top) to dark blue (bottom)
//   - Red square:     top-left  corner
//   - Yellow square:  top-right corner
//   - Green square:   bottom-right corner
//   - Cyan circle:    image centre (filled)
//   - Magenta circle: lower-left quadrant (filled)
func (cam *CameraPeripheral) pixelColor(x, y, w, h int) uint8 {
	// Background gradient: black → navy blue
	blue := uint8(y * 80 / h)
	bg := rgb332(0, 0, blue)

	// Red square — top-left
	sqSize := max(w/8, 4)
	if x >= 4 && x < 4+sqSize && y >= 4 && y < 4+sqSize {
		return rgb332(220, 30, 30)
	}

	// Yellow square — top-right
	if x >= w-4-sqSize && x < w-4 && y >= 4 && y < 4+sqSize {
		return rgb332(220, 200, 20)
	}

	// Green square — bottom-right
	if x >= w-4-sqSize && x < w-4 && y >= h-4-sqSize && y < h-4 {
		return rgb332(30, 200, 60)
	}

	// Cyan filled circle — image centre
	cx, cy := w/2, h/2
	cr := max(h/5, 4)
	dx, dy := x-cx, y-cy
	distSq := dx*dx + dy*dy
	if distSq <= cr*cr {
		return rgb332(30, 200, 220)
	}
	// Thin white ring just outside the cyan circle
	if distSq <= (cr+2)*(cr+2) {
		return rgb332(240, 240, 240)
	}

	// Magenta filled circle — lower-left quadrant
	cx2, cy2 := w/4, h*3/4
	cr2 := max(h/6, 3)
	dx2, dy2 := x-cx2, y-cy2
	if dx2*dx2+dy2*dy2 <= cr2*cr2 {
		return rgb332(200, 30, 200)
	}

	return bg
}
