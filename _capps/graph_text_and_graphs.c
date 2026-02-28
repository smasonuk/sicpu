// --- GoCPU Double-Buffering & Sync Test ---

int main() {
    // 1. Define MMIO Pointers
    int* video_ctrl = 0xFF05;
    int* active_bank = 0xFF02;
    int* vram_base = 0xB600;
    int* text_base = 0xF600;

    // 2. Enable Text (1) + Graphics (2) + Buffered Mode (4) = 7
    *video_ctrl = 7;

    // 3. Initialize Variables
    int scanline = 0;
    int page = 0;

    while (1 == 1) {
        // --- STEP 1: PREPARE BACK BUFFER ---
        // Point CPU to the bank we aren't showing yet
        *active_bank = page;

        // --- STEP 2: DRAW GRAPHICS (Radar Sweep) ---
        int i = 0;
        while (i < 4096) {
            int row = i / 32; 
            
            if (row == scanline) {
                *(vram_base + i) = 0xBBBB; // Bright Green (Color 11)
            } else {
                *(vram_base + i) = 0x1111; // Dark Blue Background (Color 1)
            }
            i = i + 1;
        }

        // --- STEP 3: UPDATE TEXT LAYER (Status Message) ---
        // Writes to Text VRAM are buffered until video_flip()
        *(text_base + 0) = 83; // 'S'
        *(text_base + 1) = 89; // 'Y'
        *(text_base + 2) = 83; // 'S'
        *(text_base + 3) = 32; // ' '
        *(text_base + 4) = 48 + page; // Current Bank Index (ASCII)

        // --- STEP 4: THE TARGETED FLIP ---
        // This copies 'page' to front and latches Text VRAM
        video_flip(page);

        // --- STEP 5: LOGIC UPDATE ---
        scanline = scanline + 1;
        if (scanline == 128) {
            scanline = 0;
        }

        // Toggle page (0 -> 1 -> 0)
        if (page == 0) { page = 1; } 
        else { page = 0; }

        // Short delay to make the sweep visible
        // int d = 0;
        // while (d < 100) { d = d + 1; }

        print("done");
    }
    
    return 0;
}