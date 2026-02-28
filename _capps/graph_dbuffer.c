// TODO: this needs to be updated to match the new MMIO register layout and video mode changes. Since we moved to byte mode, the way we write to VRAM has changed (2 pixels per word instead of 4). Also, the video control register has changed and the video_flip() function may have a different signature. This code is currently not working and needs to be revised to match the new hardware design.

// MMIO Registers
int* video_ctrl = 0xFF05;
int* active_bank = 0xFF02;
int* vram_base  = 0xB600;

void main() {

    // Set VIDEO_CONTROL: 
    // Bit 0 (Text = 1) + Bit 1 (Graphics = 2) + Bit 2 (Buffered Mode = 4) -> 7
    *video_ctrl = 7;

    // Game State
    int y = 0;
    int dir = 1;
    int current_page = 0;

    // Main Game Loop
    while (1 == 1) {
        
        // 1. Point the CPU to the "hidden" bank (our back buffer)
        *active_bank = current_page;

        // 2. Draw the frame (128x128 screen = 4096 words)
        int i = 0;
        while (i < 4096) {
            // 128 pixels wide / 4 pixels per word = 32 words per line
            int current_row = i / 32; 
            
            if (current_row == y) {
                // Draw a Red line (Palette color 8: 0x8888 packs 4 red pixels)
                *(vram_base + i) = 0x8888; 
            } else {
                // Fill background with Dark Blue (Palette color 1: 0x1111)
                *(vram_base + i) = 0x1111; 
            }
            i = i + 1;
        }

        // 3. Instantly swap the hidden bank to the physical display!
        video_flip(current_page);

        // 4. Update the bouncing logic for the next frame
        y = y + dir;
        if (y == 127) {
            dir = -1; // Bounce up
        }
        if (y == 0) {
            dir = 1;     // Bounce down
        }

        // 5. Swap our target bank for the NEXT frame (Ping-Pong: 0 -> 1 -> 0 -> 1)
        if (current_page == 0) {
            current_page = 1;
        } else {
            current_page = 0;
        }

        // 6. Simple delay loop so it doesn't move too fast to see
        int delay = 0;
        while (delay < 500) {
            delay = delay + 1;
        }
    }
}