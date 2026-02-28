#include <sys.c>
#include <video.c>

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

// Fills all 256 palette slots so that palette[i] represents 
// an equal mix of R, G, and B mapped to RGB565.
void setup_grayscale_palette() {
    int i = 0;
    while (i < 256) {
        // Drop lowest bits to fit 8-bit values into 5/6-bit RGB565 channels
        int r5 = i >> 3; 
        int g6 = i >> 2; 
        int b5 = i >> 3; 

        set_palette(i, (r5 << 11) | (g6 << 5) | b5);
        i = i + 1;
    }
}

int main() {
    init_8bpp();
    clear_8bpp(1);

    int* cam = find_peripheral("CAMERA");
    if (cam == 0) {
        print("Camera not found!\n");
        return 1;
    }

    // --- CONFIGURATION ---
    int use_grayscale = 1; // Change to 0 for colour

    if (use_grayscale == 1) {
        setup_grayscale_palette();
        cam[8] = 1; // offset 0x10: mode = Grayscale
    } else {
        setup_rgb332_palette();
        cam[8] = 0; // offset 0x10: mode = RGB332
    }
    // ---------------------

    set_active_bank(0);

    cam[1] = 0x8000;   // offset 0x02: buffer address
    cam[2] = 128;      // offset 0x04: width
    cam[3] = 128;      // offset 0x06: height

    // Trigger the capture
    cam[0] = 1;

    print("Camera capture complete. Displaying frame...\n");

    // Promote the back buffer to the display.
    video_flip(0);

    return 0;
}