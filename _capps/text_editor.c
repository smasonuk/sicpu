// Global cursor position for text output
int cursor_x = 0;
int cursor_y = 0;

// Set the display resolution mode via MMIO config port.
// mode=0: 32-column, mode=1: 64-column
int set_resolution(int mode) {
    int* config_port = 0xFF03;
    *config_port = mode;
    return 0;
}

// Write character c to VRAM at grid position (x, y).
// Reads the current column count from the config port to compute the offset.
int print_at(int x, int y, int c) {
    int* vram = 0xF000;
    int* config_port = 0xFF03;
    int cols = 32;
    if (*config_port == 1) { cols = 64; }
    *(vram + (y * cols) + x) = c;
    return 0;
}

// Read the last keypress from the keyboard MMIO port.
int get_key() {
    int* kb = 0xFF04;
    return *kb;
}

// Interrupt service routine: handles keyboard input.
// key=10 (newline): move cursor to next line
// key=8  (backspace): erase previous character and retreat cursor
// other: print character and advance cursor, wrapping at column 64
int isr() {
    int key = get_key();
    if (key != 0) {
        if (key == 10) {
            cursor_x = 0;
            cursor_y = cursor_y + 1;
        } else if (key == 8) {
            if (cursor_x > 0) {
                cursor_x = cursor_x - 1;
                print_at(cursor_x, cursor_y, 0); // erase cell
            }
        } else {
            print_at(cursor_x, cursor_y, key);
            cursor_x = cursor_x + 1;
            if (cursor_x == 64) { // wrap at end of line
                cursor_x = 0;
                cursor_y = cursor_y + 1;
            }
        }
    }
    return 0;
}

int main() {
    set_resolution(1); // 64-column mode
 
    enable_interrupts();

    // Spin forever; all work is done in the ISR
    while (1 == 1) {
        wait_for_interrupt();
    }
    return 0;
}
