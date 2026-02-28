// camera_view.c
//
// Captures a frame from the CAMERA peripheral and displays it.
//
// Format bridge: the camera writes 1 byte per pixel in RGB332 format
// (R:3 G:3 B:2 — same layout used by the peripheral's pixelColor()).
// 8-bpp video mode also uses 1 byte per pixel, but treats it as a
// palette index rather than a packed colour.
//
// Resolution: by pre-filling palette entry i with the RGB565 value
// that corresponds to RGB332 colour i, the camera byte can be used
// directly as a palette index — zero per-pixel conversion needed.
//
// Memory: the camera buffer is set to 0x8000 (VRAM start).
// CPU.WriteByte in the 0x8000-0xBFFF range routes straight into
// GraphicsBanks[CurrentBank], so the camera fills VRAM directly.

#include <sys.c>
#include <video.c>

// setup_rgb332_palette — fills all 256 palette slots so that
// palette[i] is the RGB565 colour for RGB332 packed value i.
//
// RGB332 bit layout:  [7:5]=R3  [4:2]=G3  [1:0]=B2
// RGB565 bit layout:  [15:11]=R5  [10:5]=G6  [4:0]=B5
//
// Channel scaling uses bit-replication (shifts only, no division):
//   R: 3→5 bits  r5 = (r3 << 2) | (r3 >> 1)
//   G: 3→6 bits  g6 = (g3 << 3) | g3
//   B: 2→5 bits  b5 = (b2 << 3) | (b2 << 1) | (b2 >> 1)
void setup_rgb332_palette() {
    int i = 0;
    while (i < 256) {
        int r3 = (i >> 5) & 7;
        int g3 = (i >> 2) & 7;
        int b2 = i & 3;

        int r5 = (r3 << 2) | (r3 >> 1);
        int g6 = (g3 << 3) | g3;
        int b5 = (b2 << 3) | (b2 << 1) | (b2 >> 1);

        set_palette(i, (r5 << 11) | (g6 << 5) | b5);
        i = i + 1;
    }
}

int main() {
    init_8bpp();
    clear_8bpp(1);

    // 1. Build the RGB332 → palette index mapping.
    setup_rgb332_palette();

    // 2. Enable 8-bpp buffered graphics mode and select bank 0.
    // change_video_mode_graphics_8bpp();
    // enable_buffered_mode();
    set_active_bank(0);

    // 3. Locate the camera by scanning expansion slots.
    int* cam = find_peripheral("CAMERA");
    if (cam == 0) {
        print("Camera not found!\n");
        return 1;
    }

    // 4. Aim the camera buffer at VRAM (0x8000).
    //    Each WriteByte to 0x8000-0xBFFF lands in GraphicsBanks[CurrentBank],
    //    so the capture fills the active bank directly — no extra copy step.
    cam[1] = 0x8000;   // offset 0x02: buffer address
    cam[2] = 128;       // offset 0x04: width
    cam[3] = 128;       // offset 0x06: height

    // 5. Trigger the capture (command = 1).
    cam[0] = 1;

    plot_8bpp(10, 10, 2); 

    print("Camera capture complete. Displaying frame...\n");

    // 6. Promote the back buffer to the display.
    video_flip(0);


    return 0;
}
