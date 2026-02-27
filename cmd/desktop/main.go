package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/draw"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"gocpu/pkg/compiler"
	"gocpu/pkg/cpu"
	"gocpu/pkg/grid"
	"gocpu/pkg/peripherals"
	"gocpu/pkg/utils"
)

type Game struct {
	vm          *cpu.CPU
	graphicsImg *ebiten.Image // reused 128×128 bitmap canvas
}

func loadImage(fileName string) (*image.RGBA, error) {
	imgFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer imgFile.Close()

	img, _, err := image.Decode(imgFile)
	if err != nil {
		return nil, err
	}

	rgbaImg, ok := img.(*image.RGBA)
	if !ok {
		rgbaImg = image.NewRGBA(img.Bounds())
		draw.Draw(rgbaImg, img.Bounds(), img, image.Point{}, draw.Src)
	}

	return rgbaImg, nil
}

func (g *Game) Update() error {
	for _, r := range ebiten.AppendInputChars(nil) {
		g.vm.PushKey(uint16(r))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.vm.PushKey(10) // ASCII newline
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		g.vm.PushKey(8) // ASCII backspace
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		if err := g.vm.HibernateToFile("save_state.zip"); err != nil {
			fmt.Printf("[Hibernate] Save failed: %v\n", err)
		} else {
			fmt.Println("[Hibernate] State saved to save_state.zip")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		if err := g.vm.RestoreFromFile("save_state.zip"); err != nil {
			fmt.Printf("[Hibernate] Load failed: %v\n", err)
		} else {
			fmt.Println("[Hibernate] State restored from save_state.zip")
		}
	}

	// Run at a fixed, maximum clock speed (e.g. ~60 MHz at 60fps)
	for i := 0; i < 10000; i++ {
		// Break early if the program finishes or goes to sleep!
		if g.vm.Halted || g.vm.Waiting {
			break
		}
		g.vm.Step()
	}

	return nil
}

func (g *Game) drawBitmap(screen *ebiten.Image) {
	if g.graphicsImg == nil {
		g.graphicsImg = ebiten.NewImage(128, 128)
	}

	// If the allocation STILL failed, don't try to draw this frame
	if g.graphicsImg == nil {
		return
	}

	pixels := g.vm.GetFramebufferRGBA()
	g.graphicsImg.WritePixels(pixels)

	// Scale the 128×128 image to fill the 256×256 logical screen.
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(g.graphicsImg, op)
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.vm.GraphicsEnabled {
		g.drawBitmap(screen)
		if !g.vm.TextOverlay {
			return // text hidden in pure graphics mode
		}
	}

	// Text layer
	mode := g.vm.TextResolutionMode
	var cols, charWidth, charHeight int
	if mode == 1 {
		cols = 64
		charWidth = 8
		charHeight = 12
	} else {
		cols = 32
		charWidth = 16
		charHeight = 16
	}

	sourceVRAM := g.vm.TextVRAM
	if g.vm.BufferedMode {
		sourceVRAM = g.vm.TextVRAM_Front
	}

	for i, charCode := range sourceVRAM {
		if charCode == 0 {
			continue
		}
		x, y := grid.GetGridCoords(i, cols)
		px := x * charWidth
		py := y * charHeight
		msg := fmt.Sprintf("%c", charCode)
		ebitenutil.DebugPrintAt(screen, msg, px, py)
	}
}

// startDiskSyncer flushes the VFS to disk every interval while stop is open.
func startDiskSyncer(vm *cpu.CPU, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if vm.Disk.Dirty {
				if err := vm.Disk.PersistTo(vm.StoragePath); err == nil {
					vm.Disk.Dirty = false
				}
			}
		case <-stop:
			return
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if g.vm.GraphicsEnabled {
		// 128×128 bitmap rendered at 2× scale
		return 256, 256
	}
	if g.vm.TextResolutionMode == 1 {
		// Mode 1: 64 columns × 8px, 16 rows × 12px
		return 512, 192
	}
	// Mode 0: 32 columns × 16px, 32 rows × 16px
	return 512, 512
}

const storagePath = "gocpu_vfs"

func main() {
	filename := os.Args[1]
	showAsm := false
	if len(os.Args) > 2 {
		for _, arg := range os.Args[2:] {
			showAsm = arg == "--show-asm"
		}
	}

	cameraImage, err := loadImage("./frame_1.png")
	if err != nil {
		log.Fatalf("Failed to load demo image: %v", err)
		os.Exit(1)
	}

	fullPath, baseDir, err := utils.GetPathInfo(filename)
	sourceBytes, err := os.ReadFile(fullPath)
	if err != nil {
		log.Fatalf("Failed to read source file: %v", err)
	}
	demoSource := string(sourceBytes)

	// 4. Run Game
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(512, 512)
	ebiten.SetWindowTitle("GoCPU Desktop")

	asm, mc, err := compiler.Compile(demoSource, baseDir) // TODO: handle errors
	if err != nil {
		log.Fatalf("Compilation failed: %v", err)
	}
	machineCode := mc
	if showAsm {
		print("Generated Assembly:\n", *asm, "\n")
	}

	capFunc := func() *image.RGBA {
		return cameraImage
	}

	// Register peripheral factories for hibernation restore.
	cpu.RegisterPeripheral(peripherals.MessagePeripheralType, func(c *cpu.CPU, slot uint8) cpu.Peripheral {
		return peripherals.NewMessageSender(c, slot)
	})
	cpu.RegisterPeripheral(peripherals.CameraPeripheralType, func(c *cpu.CPU, slot uint8) cpu.Peripheral {
		return peripherals.NewCameraPeripheral(c, slot, capFunc)
	})

	// 3. Initialize CPU (loads any previously saved VFS files from storagePath)
	vm := cpu.NewCPU(storagePath)
	vm.MountPeripheral(0, peripherals.NewMessageSender(vm, 0))
	vm.MountPeripheral(1, peripherals.NewCameraPeripheral(vm, 1, capFunc))

	if len(machineCode) > len(vm.Memory) {
		log.Fatalf("Program too large for memory")
	}
	copy(vm.Memory[:], machineCode)

	// Start background disk syncer (flushes dirty VFS to host every 3 s)
	stopSyncer := make(chan struct{})
	go startDiskSyncer(vm, 3*time.Second, stopSyncer)

	game := &Game{vm: vm}
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

	// Graceful shutdown: stop syncer and do a final flush
	close(stopSyncer)
	if vm.Disk.Dirty {
		_ = vm.Disk.PersistTo(vm.StoragePath)
	}
}
