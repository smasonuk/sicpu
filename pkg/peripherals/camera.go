package peripherals

import (
	"gocpu/pkg/cpu"
)

const CameraPeripheralType = "CameraPeripheral"

type CameraPeripheral struct {
	c    *cpu.CPU
	slot uint8

	bufAddr uint16
	width   uint16
	height  uint16
	mode    uint16 // 0 = RGB332, 1 = Grayscale
}

func NewCameraPeripheral(c *cpu.CPU, slot uint8) *CameraPeripheral {
	return &CameraPeripheral{
		c:      c,
		slot:   slot,
		width:  128,
		height: 128,
		mode:   0,
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
	case 0x10:
		return cam.mode
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
	case 0x10:
		cam.mode = val
	}
}

func (cam *CameraPeripheral) Type() string { return CameraPeripheralType }

func (cam *CameraPeripheral) Step() {}

func (cam *CameraPeripheral) capture() {
	w := int(cam.width)
	h := int(cam.height)
	addr := cam.bufAddr

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cam.c.WriteByte(addr, cam.pixelColor(x, y, w, h))
			addr++
		}
	}

	cam.c.TriggerPeripheralInterrupt(cam.slot)
}

// packColor dynamically returns either RGB332 or 8-bit Grayscale depending on the mode register.
func (cam *CameraPeripheral) packColor(r, g, b uint8) uint8 {
	if cam.mode == 1 {
		// Grayscale luminance formula: Y = 0.299R + 0.587G + 0.114B
		// We use fast integer approximation: (R*77 + G*150 + B*29) / 256
		// Maximum possible value is (255*77 + 255*150 + 255*29) = 65280, which safely fits in a uint16.
		y := (uint16(r)*77 + uint16(g)*150 + uint16(b)*29) >> 8
		return uint8(y)
	}
	// Default: RGB332
	return (r & 0xE0) | ((g >> 3) & 0x1C) | (b >> 6)
}

func (cam *CameraPeripheral) pixelColor(x, y, w, h int) uint8 {
	// Background gradient: black → navy blue
	blue := uint8(y * 80 / h)
	bg := cam.packColor(0, 0, blue)

	// Red square — top-left
	sqSize := max(w/8, 4)
	if x >= 4 && x < 4+sqSize && y >= 4 && y < 4+sqSize {
		return cam.packColor(220, 30, 30)
	}

	// Yellow square — top-right
	if x >= w-4-sqSize && x < w-4 && y >= 4 && y < 4+sqSize {
		return cam.packColor(220, 200, 20)
	}

	// Green square — bottom-right
	if x >= w-4-sqSize && x < w-4 && y >= h-4-sqSize && y < h-4 {
		return cam.packColor(30, 200, 60)
	}

	// Cyan filled circle — image centre
	cx, cy := w/2, h/2
	cr := max(h/5, 4)
	dx, dy := x-cx, y-cy
	distSq := dx*dx + dy*dy
	if distSq <= cr*cr {
		return cam.packColor(30, 200, 220)
	}
	// Thin white ring just outside the cyan circle
	if distSq <= (cr+2)*(cr+2) {
		return cam.packColor(240, 240, 240)
	}

	// Magenta filled circle — lower-left quadrant
	cx2, cy2 := w/4, h*3/4
	cr2 := max(h/6, 3)
	dx2, dy2 := x-cx2, y-cy2
	if dx2*dx2+dy2*dy2 <= cr2*cr2 {
		return cam.packColor(200, 30, 200)
	}

	return bg
}
