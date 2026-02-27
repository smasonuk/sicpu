#include "stdio.c"

int* MSG_CMD = 0xFC00; //writing into slot 0
int* MSG_TO  = 0xFC02;
int* MSG_BODY = 0xFC04;
int* MSG_LEN = 0xFC06;


void enable_interrupts() {
    asm("EI");
}

void disable_interrupts() {
    asm("DI");
}

void wait_for_interrupt() {
    asm("WFI");
}

void memset(int* dest, int count, int val) {

    asm("MOV R0, R4");   // R0 = dest
    asm("MOV R1, R5");   // R1 = count
    asm("MOV R3, R6");   // R2 = val (safe now, R2 is on stack)
    asm("FILL R0, R1, R3");
}

void memcpy(int* dest, int* src, int count) {
    // The compiler uses R4, R5, R6 for these args.
    // We use R0, R1, R3 as "scratch" registers because 
    // the compiler expects R0-R3 to be clobbered anyway.
    
    asm("MOV R0, R5");   // src  -> R0
    asm("MOV R1, R4");   // dest -> R1
    asm("MOV R3, R6");   // count -> R3
    asm("COPY R0, R1, R3"); 
    // We didn't touch R2 (the Frame Pointer), so the 
    // function will return safely!
}

// Scans the 16 expansion slots (0xFC00 - 0xFCFF).
// Returns the base address of the peripheral if found, or 0 if not found.
int* find_peripheral(char* target_name) {
    for (int slot = 0; slot < 16; slot++) {
        int* base_addr = (int*)(0xFC00 + (slot * 16));
        char* name_ptr = (char*)(0xFC00 + (slot * 16) + 8);
        
        if (strcmp(name_ptr, target_name) == 0) {
            return base_addr;
        }
    }
    return 0;
}


void send_msg(int slot, char* to, char* body, int len) {
    int* SLOT_BASE = 0xFC00 + (slot * 16);

    *MSG_TO = to;
    *MSG_BODY = body;
    *MSG_LEN = len;
    *MSG_CMD = 1;
}