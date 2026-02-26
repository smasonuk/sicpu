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