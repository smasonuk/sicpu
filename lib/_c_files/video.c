#include <stdio.c>
#include <sys.c>
#include <message.c>
#include <cam.c>



// Video Hardware Ports
int* VIDEO_FLIP_PORT = 0xFF06;
int* VIDEO_CTRL = 0xFF05;
int* PALETTE_IDX  = 0xFF07;
int* PALETTE_DATA = 0xFF08;
int TEXT_MODE = 1;
int GRAPHICS_MODE = 2;
int BUFFERED_MODE = 4;
int COLOR_8BPP_MODE = 8;
int* active_bank = 0xFF02;


// Image buffer: 128x128 = 16384 bytes, placed safely in mid-RAM (away from
// code at 0x0010 and stack at 0xFFFE).
#define IMAGE_BUFFER 0x4000
#define IMAGE_SIZE   16384


void set_active_bank(int bank) {
    *active_bank = bank;
}

// Swaps the active display bank to the specified bank (0-3)
void video_flip(int bank) {
    *VIDEO_FLIP_PORT = bank;
}

void change_video_mode_text() {
    int* res_mode = VIDEO_CTRL;
    *res_mode = TEXT_MODE;
}

void change_video_mode_graphics() {
    int* res_mode = VIDEO_CTRL;
    *res_mode = GRAPHICS_MODE;
}

void change_video_mode_both() {
    int* res_mode = VIDEO_CTRL;
    *res_mode = TEXT_MODE | GRAPHICS_MODE;
}

void enable_buffered_mode() {
    int* res_mode = VIDEO_CTRL;
    // *res_mode =  *res_mode |  BUFFERED_MODE;
    *res_mode =  *res_mode |  BUFFERED_MODE;
}

void change_video_mode_graphics_8bpp() {
    *VIDEO_CTRL = GRAPHICS_MODE | COLOR_8BPP_MODE;
}

void set_palette(int index, int rgb565) {
    *PALETTE_IDX = index;
    *PALETTE_DATA = rgb565;
}

//DOES this work?
int draw_pixel_8bpp(int x, int y, char color_index) {
    char* vram = 0xB600;
    *(vram + (y * 128) + x) = color_index;
    return 0;
}

int draw_pixel(int x, int y, char color) {
    int* gbase = 0xB600;   
    int pixel_index = y * 128 + x;
    int byte_index = pixel_index >> 1;         // which char in the bank
    int shift = (pixel_index & 1) << 2;        // bit offset within the char (0 for even pixels, 4 for odd pixels)
    int current = *(gbase + byte_index);
    int mask = 15 << shift;                    // 0xF in the right nibble position
    int cleared = current & ~mask;             // zero out the target nibble
    int colored = color << shift;              // place the new color in that nibble

    *(gbase + byte_index) = cleared | colored;
    return 0;
}


// // Sets the pixel at (x,y) to 'color' (0-255) using 8bpp mode.
void plot_8bpp(int x, int y, char color) {
    // VRAM starts at 0xB600. Each char is a pixel.
    char* vram = (char*)0xB600;
    vram[(y << 7) + x] = color; // y * 128 + x
}

// // Clears the entire graphics bank in one go using the CPU's FILL instruction.
void clear_8bpp(char color) {
    // Pack the color into both bytes of a word (16-bit)
    int pattern = (color << 8) | color;
    // 128*128 pixels = 16384 bytes = 8192 words.
    memset((int*)0xB600, 8192, pattern); 
}

// Enables 8bpp mode and sets up the palette for standard colors.
void init_8bpp() {
    change_video_mode_graphics_8bpp();
    enable_buffered_mode();
    set_active_bank(0);
}

int take_picture_and_send(char* send_to_address) {
    int CAM_CAPTURE_COMMAND = 1;

    int* cam = find_peripheral("CAMERA");
    if (cam == 0) {
        print("Camera not found!\n");
        return 1;
    }

    takepicture(IMAGE_BUFFER);

    send_message(send_to_address, IMAGE_BUFFER, IMAGE_SIZE);

    return 0;
}
