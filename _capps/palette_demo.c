#include "../../lib/sys.c"
#include "../../lib/video.c"

int main() {
    // Build a 4-segment rainbow palette across all 256 indices.
    //
    //   0 – 63  : black  → red      (r rises 0→31, g=0,  b=0)
    //  64 – 127 : red    → yellow   (r=31,  g rises 0→63, b=0)
    // 128 – 191 : yellow → green    (r falls 31→0, g=63, b=0)
    // 192 – 255 : green  → blue     (r=0,  g falls 63→0, b rises 0→31)
    int i = 0;
    while (i < 64) {
        // 0-63: black → red
        set_palette(i, (i >> 1) << 11);

        // 64-127: red → yellow (green channel 0→63)
        set_palette(i + 64, (31 << 11) | (i << 5));

        // 128-191: yellow → green (red channel 31→0)
        set_palette(i + 128, ((31 - (i >> 1)) << 11) | (63 << 5));

        // 192-255: green → blue (green 63→0, blue 0→31)
        set_palette(i + 192, ((63 - i) << 5) | (i >> 1));

        i = i + 1;
    }

    // Switch to 8bpp mode – one byte per pixel, 256 palette entries available.
    change_video_mode_graphics_8bpp();
    enable_buffered_mode();
    set_active_bank(0); 

    // Draw a plasma pattern.
    // The x*y term curves the isolines so the bands aren't straight diagonals.
    int y = 0;
    while (y < 128) {
        int x = 0;
        while (x < 128) {
            int color = (x + y + ((x * y) >> 7)) & 255;
            draw_pixel_8bpp(x, y, color);
            x = x + 1;
        }
        y = y + 1;
    }

    video_flip(0); 

    return 0;
}
