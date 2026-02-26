#include "../../lib/sys.c"
#include "../../lib/video.c"
#include "../../lib/stdio.c"



int main() {
    change_video_mode_graphics();

    // Draw a cycling color gradient: color = (x + y) & 15
    byte y = 0;
    while (y < 128) {
        byte x = 0;
        while (x < 128) {
            byte color = (x + y) & 15;
            draw_pixel(x, y, color);
            x = x + 1;
        }
        y = y + 1;
    }
    return 0;
}
