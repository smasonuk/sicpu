#include "../lib/sys.c"
#include "../lib/video.c"
#include "../lib/stdio.c"
#include "../lib/math.c"


int main() {
    init_8bpp();
    int angle = 0;
    int radius = 40;

    while (1) {
        // 1. Prepare the next frame (Back Buffer)
        clear_8bpp(1); // Dark Blue background

        // 2. Calculate coordinates
        int x = 64 + f_mul(radius, get_cos(angle));
        int y = 64 + f_mul(radius, get_sin(angle));

        // 3. Draw a "star" (3x3 cluster)
        plot_8bpp(x, y, 7);     // White center
        plot_8bpp(x+1, y, 10);  // Yellow neighbors
        plot_8bpp(x-1, y, 10);
        plot_8bpp(x, y+1, 10);
        plot_8bpp(x, y-1, 10);

        // 4. Swap buffers for flicker-free movement
        video_flip(0);
        
        angle = (angle + 2) & 255; // Increase speed

        // Simple delay loop (not precise, but good enough for demo)
        int delay = 1000;
        while (delay > 0) {
            delay = delay - 1;  
        }

    }
    return 0;
}