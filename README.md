# gocpu

A custom 16-bit virtual CPU, assembler, and C-subset compiler written in Go.

## Features

- 16-bit address space (64 KB memory)
- 8 general-purpose registers: R0-R7
- Subroutine calls with a hardware stack
- Hardware interrupt support (`EI` / `DI` / `WFI` / `RETI`)
- Memory-Mapped I/O for console output, keyboard input, video, and a virtual file system
- Two-pass assembler with labels, `.ORG`, `.STRING`, and `.WORD`
- C-subset compiler with preprocessor (`#include`, `#define`), structs, arrays, pointers, and inline `asm()`
- Dead-function elimination optimizer
- Web IDE: assemble/compile and run programs in the browser, with single-step debugging
- Ebiten-based desktop app with text and bitmap graphics modes

---

## Memory Map

All addresses are **word** addresses (each word = 2 bytes). The CPU is word-addressed, so address `0x0001` refers to the second 16-bit word in memory.

| Memory Segment     | Word Range            | Byte Equivalent       | Size (words) | Owner                    |
|--------------------|-----------------------|-----------------------|--------------|--------------------------|
| **System/Vectors** | `0x0000` – `0x000F`   | `0x0000` – `0x001F`   | 16           | Compiler / Assembler     |
| **Code & Data**    | `0x0010` – `0x3FFF`   | `0x0020` – `0x7FFF`   | 16,368       | Compiler (dynamic)       |
| **Graphics VRAM**  | `0x4000` – `0x5FFF`   | `0x8000` – `0xBFFF`   | 8,192        | Hardware (`cpu.go`)      |
| **General RAM**    | `0x6000` – `0x77FF`   | `0xC000` – `0xEFFF`   | 6,144        | Free RAM                 |
| **Text VRAM**      | `0x7800` – `0x7BFF`   | `0xF000` – `0xF7FF`   | 1,024        | Hardware (`cpu.go`)      |
| **Upper RAM**      | `0x7C00` – `0x7F7F`   | `0xF800` – `0xFEFF`   | 896          | Free RAM (used by stack) |
| **MMIO**           | `0x7F80` – `0x7F8F`   | `0xFF00` – `0xFF1F`   | 16           | Hardware (`cpu.go`)      |
| **Stack (top)**    | `0x7F90` – `0x7FFF`   | `0xFF20` – `0xFFFF`   | 112          | SP grows down from top   |

**Interrupt vector:** The CPU jumps to address `0x0010` (byte `0x0020`) when an interrupt fires. Place your ISR there or use `.ORG 0x0010`.

---

## Quick Start — CLI

```bash
go build -o gocpu .

# Assemble a .asm file and write the binary
./gocpu -in example.asm -out example.bin

# Assemble and immediately run
./gocpu -in example.asm -run

# Compile a .c file and immediately run
./gocpu -in program.c -run

# Run an existing binary
./gocpu -run-bin example.bin

# Run with a persistent VFS storage directory
./gocpu -in program.c -run -storage ./mydata
```

### CLI flags

| Flag            | Description                                                  |
|-----------------|--------------------------------------------------------------|
| `-in <file>`    | Input file — `.asm` for assembler, `.c` for C compiler       |
| `-out <file>`   | Output binary path (default: same name as input with `.bin`) |
| `-run`          | Assemble/compile and run immediately                         |
| `-run-bin <file>` | Run an existing `.bin` file directly                       |
| `-storage <dir>`| Directory used as VFS backing store (persistent across runs) |

After a run completes, the CPU state is printed:

```
run complete (program.bin): PC=0x0010 SP=0xFFFE Z=false N=false R0=0x0007 R1=0x0000 R2=0x0000 R3=0x0000
```

---

## Desktop App

The desktop app runs the emulator with an Ebiten-based display (text and bitmap modes).

```bash
go run ./cmd/desktop
```

### Video Output

#### Text Mode

The text VRAM consists of 1,024 words starting at address `0xF000`. It renders as a 64-column × 16-row grid (default), or a 32-column × 32-row grid.

- `0xF000` → (col=0, row=0)
- `0xF001` → (col=1, row=0)
- `0xF040` → (col=0, row=1)  *(offset = 64 words per row)*

Write an ASCII code as a 16-bit word to the cell address to display a character.

#### Bitmap Mode

128×128 pixels, 16 colors (Pico-8-inspired palette), 4-bit packed per nibble.

- **VRAM base:** `0x8000` – `0x9FFF` (4-bit packed: 4 pixels per word)
- **8bpp unpacked mode:** each word stores one 8-bit color index; enable with `VIDEO_CTRL` bit 3
- **Color Depth:** 4-bit (16 colors) in default mode, 8-bit (256 colors) in 8bpp mode
- **Pixel layout (4bpp):**
  - Bits 0–3: pixel `n`
  - Bits 4–7: pixel `n+1`
  - Bits 8–11: pixel `n+2`
  - Bits 12–15: pixel `n+3`
- **Banking:** 4 graphics banks available. Write bank index (0–3) to `0xFF02` to select which bank the CPU writes to.
- **Double buffering:** Enable with bit 2 of `0xFF05`, then call `video_flip(bank)` to swap back→front.

---

## Instruction Set

### Encoding

Every instruction is one 16-bit word:

```
 15  14  13  12  11 | 10   9 | 8   7 | 6 … 0
       opcode       |  regA  |  regB | (unused)
```

Instructions that take an immediate value (label or literal) store it in the **next** word (2 words total).

### Registers

| Name | Index | Notes                                              |
|------|-------|----------------------------------------------------|
| R0   | 0     | General purpose; also holds the return value       |
| R1   | 1     | General purpose                                    |
| R2   | 2     | General purpose; used as frame pointer by compiler |
| R3   | 3     | General purpose; scratch register for compiler     |
| PC   | —     | Program counter                                    |
| SP   | —     | Stack pointer; initialised to `0xFFFE`, grows down |

**Flags:**
- **Z** — Zero: set when an arithmetic/logic result is 0
- **N** — Negative: set when bit 15 of the result is 1 (signed negative)
- **C** — Carry/Borrow: set by `ADD` on unsigned overflow, set by `SUB` when the result borrows

### Instruction Reference

#### No operands

| Mnemonic | Opcode | Description                                         |
|----------|--------|-----------------------------------------------------|
| `HLT`    | 0x00   | Halt execution                                      |
| `NOP`    | 0x01   | No operation                                        |
| `RET`    | 0x15   | Return from subroutine: pop PC from stack           |
| `EI`     | 0x16   | Enable interrupts                                   |
| `DI`     | 0x17   | Disable interrupts                                  |
| `RETI`   | 0x18   | Return from interrupt handler: pop PC, re-enable interrupts |
| `WFI`    | 0x19   | Wait for interrupt (halts until one fires)          |

#### One register

| Mnemonic     | Opcode | Description                                   |
|--------------|--------|-----------------------------------------------|
| `NOT Rn`     | 0x0B   | `Rn = ~Rn`; sets Z, N                         |
| `PUSH Rn`    | 0x12   | Push `Rn` onto the stack                      |
| `POP Rn`     | 0x13   | Pop top of stack into `Rn`                    |
| `LDSP Rn`    | 0x1A   | `Rn = SP` - Copies the current value of the Stack Pointer into a general-purpose register |
| `STSP Rn`    | 0x1B   | `SP = Rn` - Replaces the value in the Stack Pointer with the value from a general-purpose register.|

#### Two registers

| Mnemonic        | Opcode | Description                                                      |
|-----------------|--------|------------------------------------------------------------------|
| `MOV Ra, Rb`    | 0x03   | `Ra = Rb`                                                        |
| `LD  Ra, [Rb]`  | 0x04   | `Ra = Memory[Rb]` — load word from address in Rb                 |
| `ST  [Ra], Rb`  | 0x05   | `Memory[Ra] = Rb` — store word; triggers MMIO if `Ra` is in range |
| `ADD Ra, Rb`    | 0x06   | `Ra = Ra + Rb`; sets Z, N, C                                     |
| `SUB Ra, Rb`    | 0x07   | `Ra = Ra − Rb`; sets Z, N, C                                     |
| `AND Ra, Rb`    | 0x08   | `Ra = Ra & Rb`; sets Z, N                                        |
| `OR  Ra, Rb`    | 0x09   | `Ra = Ra \| Rb`; sets Z, N                                       |
| `XOR Ra, Rb`    | 0x0A   | `Ra = Ra ^ Rb`; sets Z, N                                        |
| `SHL Ra, Rb`    | 0x0C   | `Ra = Ra << Rb`; sets Z, N                                       |
| `SHR Ra, Rb`    | 0x0D   | `Ra = Ra >> Rb` (logical); sets Z, N                             |
| `MUL Ra, Rb`    | 0x1C   | `Ra = Ra * Rb`; sets Z, N                                        |
| `DIV Ra, Rb`    | 0x1D   | `Ra = Ra / Rb` (unsigned); sets Z, N; `Ra = 0` if `Rb = 0`      |
| `LDB Ra, [Rb]`  | 0x20   | `Ra = Memory[Rb]` — load **byte** (zero-extended to 16 bits)     |
| `STB [Ra], Rb`  | 0x21   | `Memory[Ra] = Rb & 0xFF` — store low **byte** only               |
| `IDIV Ra, Rb`   | 0x22   | `Ra = Ra / Rb` (signed two's-complement); sets Z, N; `Ra = 0` if `Rb = 0` |

#### Three registers

| Mnemonic          | Opcode | Description                                                |
|-------------------|--------|------------------------------------------------------------|
| `FILL Ra, Rb, Rc` | 0x1E   | Hardware memset: fill `Rb` words at address `Ra` with `Rc` |
| `COPY Ra, Rb, Rc` | 0x1F   | Hardware memcpy: copy `Rc` words from address `Ra` to `Rb` |

#### Register + immediate (2 words)

| Mnemonic      | Opcode | Description                                  |
|---------------|--------|----------------------------------------------|
| `LDI Ra, imm` | 0x02   | `Ra = imm` — load a 16-bit immediate or label address |

#### Immediate only — branches and calls (2 words)

| Mnemonic       | Opcode | Condition                        |
|----------------|--------|----------------------------------|
| `JMP target`   | 0x0E   | Unconditional jump               |
| `JZ  target`   | 0x0F   | Jump if Z set (result was zero)  |
| `JNZ target`   | 0x10   | Jump if Z clear                  |
| `JN  target`   | 0x11   | Jump if N set (signed negative)  |
| `JC  target`   | 0x23   | Jump if C set (unsigned overflow / borrow) |
| `CALL target`  | 0x14   | Push next PC onto stack, then jump |

---

## Memory-Mapped I/O

All MMIO ports occupy `0xFF00`–`0xFF1F`. Use `ST [Rport], Rdata` to write and `LD Rdst, [Rport]` to read.

### Console

| Address  | R/W   | Description                                               |
|----------|-------|-----------------------------------------------------------|
| `0xFF00` | Write | Output the low byte of the register value as an ASCII character |
| `0xFF01` | Write | Output the register value as a signed decimal integer     |

### Video

| Address  | R/W        | Description                                                                   |
|----------|------------|-------------------------------------------------------------------------------|
| `0xFF02` | Write      | Set active GPU write bank (0–3)                                               |
| `0xFF03` | Read/Write | Text resolution mode: `0` = 32×32, `1` = 64×16                               |
| `0xFF05` | Read/Write | Video control flags (see below)                                               |
| `0xFF06` | Write      | Video flip: copy back-buffer bank N to front; write bank index (0–3)          |
| `0xFF07` | Read/Write | Palette index register (0–15 in 4bpp, 0–255 in 8bpp)                         |
| `0xFF08` | Read/Write | Palette data register — write RGB565 colour for the selected palette index    |

**`0xFF05` Video Control bits:**

| Bit | Mask | Name             | Description                                    |
|-----|------|------------------|------------------------------------------------|
| 0   | 0x01 | Text Overlay     | Enable text layer on top of graphics           |
| 1   | 0x02 | Graphics Enable  | Enable bitmap graphics layer                   |
| 2   | 0x04 | Buffered Mode    | Enable double-buffering (requires `video_flip`) |
| 3   | 0x08 | 8bpp Mode        | Each VRAM word stores one 8-bit colour index   |

### Keyboard

| Address  | R/W  | Description                                             |
|----------|------|---------------------------------------------------------|
| `0xFF04` | Read | Pop the oldest keycode from the keyboard buffer; returns 0 if empty |

### Virtual File System

| Address  | R/W        | Description                                               |
|----------|------------|-----------------------------------------------------------|
| `0xFF10` | Write      | VFS command trigger (see command table below)             |
| `0xFF11` | Read/Write | VFS filename pointer — address of a null-terminated string in RAM |
| `0xFF12` | Read/Write | VFS buffer pointer — address of the data buffer in RAM    |
| `0xFF13` | Read/Write | VFS size/length (words)                                   |
| `0xFF14` | Read       | VFS status code (see status table below)                  |
| `0xFF15` | Read       | VFS free-space high word (32-bit result with `0xFF13`)    |

**VFS commands (`0xFF10`):**

| Value | Name        | Description                                                                      |
|-------|-------------|----------------------------------------------------------------------------------|
| 1     | Read        | Load file named by `0xFF11` into buffer at `0xFF12`                              |
| 2     | Write       | Save `0xFF13` words from buffer `0xFF12` to file named by `0xFF11`               |
| 3     | Size        | Get file size in words; result written to `0xFF13`                               |
| 4     | Delete      | Delete the file named by `0xFF11`                                                |
| 5     | List        | Return next filename into buffer at `0xFF12`; call repeatedly until status = 5   |
| 6     | FreeSpace   | Return remaining capacity: low word in `0xFF13`, high word in `0xFF15`           |
| 7     | GetMeta     | Write 12 uint16 values (created/modified timestamps) to buffer at `0xFF12`       |
| 8     | ExecWait    | Load and run binary named by `0xFF11`; resume when it halts                      |

**VFS status codes (`0xFF14`):**

| Value | Name         | Meaning                             |
|-------|--------------|-------------------------------------|
| 0     | Success      | Command completed successfully      |
| 1     | NotFound     | File does not exist                 |
| 2     | Full         | Disk quota exceeded                 |
| 3     | InvalidName  | Filename failed validation          |
| 4     | OutOfBounds  | Buffer address out of valid RAM     |
| 5     | DirEnd       | No more files (end of List command) |

**Filename rules:** case-sensitive, matches `^[a-zA-Z0-9_]{1,12}(\.[a-zA-Z0-9]{1,3})?$`, max 16 characters.

**Total capacity:** 1.44 MB (737,280 words).

---

## Peripherals and Expansion Bus

The CPU supports up to 16 peripheral devices connected via the **Expansion Bus**, which occupies the MMIO address range `0xFC00`–`0xFCFF`. Each slot is 16 bytes wide.

### Address Mapping

| Slot | Address Range | Size (bytes) | Description                |
|------|---------------|--------------|----------------------------|
| 0    | `0xFC00`–`0xFC0F` | 16    | Peripheral slot 0           |
| 1    | `0xFC10`–`0xFC1F` | 16    | Peripheral slot 1           |
| ...  | ...           | ...          | ...                         |
| 15   | `0xFCF0`–`0xFCFF` | 16    | Peripheral slot 15          |

Each slot can hold a peripheral implementing the `Peripheral` interface.

### Peripheral Interface

All peripherals must implement the following interface (in Go):

```go
type Peripheral interface {
    // Read16(offset uint16) uint16
    // Reads a 16-bit value at the given offset (0–15) within the slot.
    
    // Write16(offset uint16, val uint16)
    // Writes a 16-bit value at the given offset (0–15) within the slot.
    
    // Step()
    // Called on every CPU step. Allows asynchronous hardware behavior.
}
```

### Using Peripherals from Assembly/C

Peripherals are accessed like any other MMIO device via `LD`/`ST` instructions:

```asm
; Read from peripheral in slot 2, offset 0x04
LD  R0, [0xFC24]     ; 0xFC20 + 0x04

; Write to peripheral in slot 2, offset 0x00
LDI R1, 42
ST  [0xFC20], R1     ; 0xFC20 + 0x00

; Byte access
LDB R0, [0xFC21]     ; Read low byte from slot 2, offset 0x00
STB [0xFC21], R1     ; Write low byte to slot 2, offset 0x00
```

### Built-in Peripherals

#### 1. DMA Tester (`DMATester`)

A simple **Direct Memory Access** (DMA) controller for testing hardware transfers.

**Registers (offsets within slot):**

| Offset | R/W        | Description                                       |
|--------|------------|---------------------------------------------------|
| 0x00   | Write      | Command register: write `1` to trigger DMA         |
| 0x02   | Read/Write | Target address (destination)                       |
| 0x04   | Read/Write | Transfer length (in bytes)                         |

**Example (Assembly):**

```asm
; Mount DMA in slot 1 and transfer 16 bytes to address 0x1000
LDI R0, 0x1000      ; target address
ST  [0xFC12], R0    ; write to slot 1, offset 0x02

LDI R0, 16          ; length
ST  [0xFC14], R0    ; write to slot 1, offset 0x04

LDI R0, 1           ; trigger command
ST  [0xFC10], R0    ; write to slot 1, offset 0x00 (fires interrupt)
```

The DMA peripheral fills the target region with `0xAA` bytes and triggers a **peripheral interrupt** (slot 1, bit mask `0x0002`) when complete.

#### 2. Message Peripheral (`MessagePeripheral`)

A peripheral that sends formatted messages to the host output. Useful for inter-process communication or debugging.

**Registers (offsets within slot):**

| Offset | R/W        | Description                                       |
|--------|------------|---------------------------------------------------|
| 0x00   | Write      | Control: write `1` to send message                 |
| 0x02   | Read/Write | Target address (null-terminated recipient string)  |
| 0x04   | Read/Write | Body/data address (message content)                |
| 0x06   | Read/Write | Body length (in bytes)                             |

**Example (Assembly):**

```asm
; Send message to "system" with body "hello"

; Write target string at 0x1000
.ORG 0x1000
TARGET: .STRING "system"

; Write message body at 0x1010
.ORG 0x1010
BODY: .STRING "hello"

; Configure and send
.ORG 0x0100
    LDI R0, 0x1000      ; target address
    ST  [0xFC02], R0

    LDI R0, 0x1010      ; body address
    ST  [0xFC04], R0

    LDI R0, 5           ; body length
    ST  [0xFC06], R0

    LDI R0, 1           ; send
    ST  [0xFC00], R0
```

Output (host stdout):
```
[Message HW] To: system | Body: hello
```

### Interrupts from Peripherals

Peripherals can trigger interrupts by calling `cpu.TriggerPeripheralInterrupt(slot)`.

**From Go:**

```go
// In your custom Peripheral implementation:
func (p *MyPeripheral) someMethod() {
    // ... do work ...
    p.cpu.TriggerPeripheralInterrupt(p.slot)
}
```

**From assembly, handling a peripheral interrupt:**

```asm
JMP MAIN

; Interrupt vector
.ORG 0x0010
ISR:
    ; R0 = read interrupt mask register
    LD  R0, [0xFF09]
    
    ; Check if slot 1 fired (bit 1)
    LDI R1, 0x0002
    AND R0, R1
    JZ  ISR_END
    
    ; Handle slot 1 interrupt...
    ; ... (call handler, etc.)
    
    ; Clear interrupt flag
    LDI R1, 0x0002
    ST  [0xFF09], R1    ; write-to-clear
    
ISR_END:
    RETI

MAIN:
    EI                  ; enable interrupts
    LOOP:
        WFI             ; wait for interrupt
        JMP LOOP
```

**Interrupt Mask Register (MMIO `0xFF09`):**

| Bit | Value | Slot |
|-----|-------|------|
| 0   | 0x0001 | Slot 0 |
| 1   | 0x0002 | Slot 1 |
| 2   | 0x0004 | Slot 2 |
| ... | ...   | ...   |
| 15  | 0x8000 | Slot 15 |

Writing a bit value to `0xFF09` clears the corresponding interrupt flag.

### Implementing a Custom Peripheral

Create a struct that implements the `Peripheral` interface:

```go
package myapp

import (
    "gocpu/pkg/cpu"
)

type MyPeripheral struct {
    c    *cpu.CPU
    slot uint8
    
    // Your registers
    statusReg uint16
    dataReg   uint16
}

func NewMyPeripheral(c *cpu.CPU, slot uint8) *MyPeripheral {
    return &MyPeripheral{
        c:    c,
        slot: slot,
    }
}

func (p *MyPeripheral) Read16(offset uint16) uint16 {
    switch offset {
    case 0x00:
        return p.statusReg
    case 0x02:
        return p.dataReg
    }
    return 0
}

func (p *MyPeripheral) Write16(offset uint16, val uint16) {
    switch offset {
    case 0x00:
        p.statusReg = val
        if val == 1 {
            p.doSomething()
        }
    case 0x02:
        p.dataReg = val
    }
}

func (p *MyPeripheral) Step() {
    // Called every CPU cycle; use for async behavior
}

func (p *MyPeripheral) doSomething() {
    // Perform work...
    
    // Trigger an interrupt when done
    p.c.TriggerPeripheralInterrupt(p.slot)
}
```

Then mount it:

```go
c := cpu.NewCPU()
myPeripheral := NewMyPeripheral(c, 3)
c.MountPeripheral(3, myPeripheral)
```

---

## Assembler Directives

| Directive          | Description                                                                 |
|--------------------|-----------------------------------------------------------------------------|
| `.ORG addr`        | Set the current address counter to `addr` (decimal or hex; cannot go backward) |
| `.STRING "text"`   | Emit each character as a 16-bit word, null-terminated (supports `\n`, `\t`, `\\`, `\"`) |
| `.WORD value`      | Emit a single 16-bit word literal                                           |

Labels end with `:` and may appear on their own line or before an instruction. Labels are **case-insensitive**.

Comments begin with `;` or `//` and run to end of line.

### Example

```asm
    JMP MAIN

.ORG 0x0010           ; interrupt vector
ISR:
    RETI

MAIN:
    LDI R0, MSG       ; R0 = address of string
    CALL PRINT
    HLT

PRINT:                ; print null-terminated string pointed to by R0
PRINT_LOOP:
    LD  R1, [R0]
    JZ  PRINT_DONE
    ST  [0xFF00], R1  ; write character to console
    LDI R3, 1
    ADD R0, R3
    JMP PRINT_LOOP
PRINT_DONE:
    RET

MSG:
    .STRING "Hello, World!\n"
```

---

## Interrupt Handling

The CPU jumps to address `0x0010` when an interrupt fires (and interrupts are enabled via `EI`).

```asm
    JMP MAIN          ; skip over the handler

.ORG 0x0010
ISR:
    ; ... save registers, handle event ...
    RETI              ; restore PC and re-enable interrupts

MAIN:
    EI
LOOP:
    WFI               ; sleep until interrupted
    JMP LOOP
```

From Go host code, call `cpu.TriggerInterrupt()` to fire a software interrupt. The ISR must be named `isr` in C code to be treated as a root by the dead-function eliminator.

---

## C-Subset Compiler

The `pkg/compiler` package implements a C dialect that compiles to GoCPU assembly. It is exposed in the Web IDE via the **C** tab and as a standalone CLI tool.

### Compiler CLI

```bash
# Compile a file and print tokens / AST / generated assembly
go run ./cmd/ccompiler prog.c

# Compile via the main CLI (produces a .bin)
./gocpu -in prog.c -out prog.bin
./gocpu -in prog.c -run
```

### Preprocessor

The preprocessor runs before lexing and handles:

```c
// Include another file (searched relative to current file, then CWD)
#include "lib/stdio.c"

// Define a constant (supports nested defines)
#define MAX_SIZE 256
#define BUFFER_WORDS (MAX_SIZE / 2)

int buf[BUFFER_WORDS];  // becomes int buf[128]
```

- `#include "file"` — replaces the directive with the contents of `file`; circular includes are detected and rejected
- `#define NAME VALUE` — performs word-boundary text substitution across the rest of the source (skipping string literals); defines expand transitively

### Supported Syntax

```c
//  Types 
int x = 10;           // signed 16-bit integer
unsigned y = 50000;   // unsigned 16-bit integer
unsigned int z = 0xFFF0u; // u/U suffix forces unsigned literal
byte b = 255;         // 8-bit value (stored in 16-bit word; upper byte ignored)

//  Structs 
struct Point {
    int x;
    int y;
};
struct Point pt;
pt.x = 10;
pt.y = 20;

//  Arrays 
int arr[10];           // array of 10 ints
int arr2[] = {1,2,3};  // size inferred (3)
arr[0] = 5;

//  Pointers 
int* p = &x;           // address-of
int v = *p;            // dereference
*p = 99;               // dereference-assign
byte* bp = 0x8000;     // raw address cast to byte pointer

//  Arithmetic 
int a = x + y;
int b = x - y;
int c = x * y;
int d = x / y;   // IDIV (signed) for int, DIV (unsigned) for unsigned int
int r = x % y;
x++;  x--;
x += 5;  x -= 2;  x *= 3;  x /= 2;

//  Bitwise 
int bits = x & y;   // AND
int bits = x | y;   // OR
int bits = x ^ y;   // XOR
int bits = ~x;      // NOT
int bits = x << 2;  // shift left
int bits = x >> 1;  // shift right (logical)

//  Comparison 
int eq = x == y;
int ne = x != y;
int lt = x < y;   // signed: uses JN;  unsigned: uses JC
int gt = x > y;

//  Logical (short-circuit) 
int a = x && y;
int o = x || y;
int n = !x;

//  Control flow 
if (x == 10) { y = 1; } else { y = 0; }

while (x > 0) { x--; }

for (int i = 0; i < 10; i++) { arr[i] = i; }

switch (x) {
    case 1: y = 10; break;
    case 2: y = 20; break;
    default: y = 0;
}

break;     // exit nearest for/while/switch
continue;  // jump to post-step of nearest for/while

//  Functions 
int add(int a, int b) { return a + b; }
void log(int val) { print_int(val); return; }  // void: return; is optional

//  Inline assembly 
asm("NOP");
asm("LDI R0, 42");

//  Type casts 
byte lo = (byte)x;   // truncate to 8 bits
int  w  = (int)b;    // widen byte to int
```

### Types

| Type           | Width  | Division                         | Less-than comparison |
|----------------|--------|----------------------------------|----------------------|
| `int`          | 16-bit | `IDIV` (signed two's-complement) | `JN` (sign flag)     |
| `unsigned`     | 16-bit | `DIV` (unsigned)                 | `JC` (carry flag)    |
| `unsigned int` | 16-bit | `DIV` (unsigned)                 | `JC` (carry flag)    |
| `byte`         | 8-bit  | —                                | —                    |

Integer literals are **signed** by default. Append `u` or `U` to force unsigned (e.g. `65535u`, `0xFFFFu`). When either operand of a compile-time constant fold is unsigned, the entire expression is folded as unsigned.

### Calling Convention

- Parameters are pushed right-to-left onto the stack.
- R2 is the **frame pointer** (FP); locals are at negative offsets from FP.
- R0 holds the **return value**.
- R1, R3 are caller-saved scratch registers.
- The compiler emits a function prologue (`PUSH R2; MOV R2, SP`) and epilogue (`MOV SP, R2; POP R2; RET`).

### Optimizer

The compiler runs a **dead function elimination** pass after parsing:

- Reachability is seeded from `main` and `isr` (if present), plus any functions called from global initializers.
- All transitively unreachable functions are removed from the AST before code generation, reducing binary size.
- Built-in intrinsics (`print`, `enable_interrupts`, etc.) are always treated as external and are never pruned.

---

## Standard Library

The `lib/` directory contains C source files you include with `#include`. None of them require a pre-compile step.

### `lib/stdio.c` — Console I/O and strings

```c
#include "lib/stdio.c"

print("Hello\n");            // write null-terminated string to 0xFF00
print_int(42);               // write decimal integer to 0xFF01

int n = strlen(str);         // length of null-terminated byte string
strcpy(dest, src);           // copy string (null-terminated)
strcmp(s1, s2);              // 0 if equal, <0 / >0 otherwise
strcat(dest, src);           // append src to end of dest
reverse(str);                // reverse string in place
itoa(n, buf);                // convert integer to decimal string
print_array(arr, len);       // print an int array as "[1, 2, 3]"
```

### `lib/sys.c` — System intrinsics

```c
#include "lib/sys.c"

enable_interrupts();         // EI
disable_interrupts();        // DI
wait_for_interrupt();        // WFI

memset(dest, count, val);    // hardware FILL: write 'val' to 'count' words at 'dest'
memcpy(dest, src, count);    // hardware COPY: copy 'count' words from 'src' to 'dest'
```

### `lib/video.c` — Video control

```c
#include "lib/video.c"

video_flip(0);               // flip back-buffer bank 0 to front (requires buffered mode)
set_active_bank(1);          // select GPU write bank 1

change_video_mode_text();           // text layer only (bit 0)
change_video_mode_graphics();       // graphics layer only (bit 1)
change_video_mode_both();           // text + graphics
change_video_mode_graphics_8bpp();  // 8bpp unpacked mode
enable_buffered_mode();             // enable double-buffering

set_palette(index, rgb565);  // set palette entry to an RGB565 colour

// Draw a 4bpp pixel at (x, y) with colour index 0–15
draw_pixel(x, y, color);

// Draw an 8bpp pixel at (x, y) with colour index 0–255
draw_pixel_8bpp(x, y, color_index);
```

### `lib/vfs.c` — Virtual file system

```c
#include "lib/vfs.c"

// Read a file into buffer (buffer must be large enough)
int status = vfs_read(filename_ptr, buffer_ptr);

// Write 'length' words from buffer to a file
int status = vfs_write(filename_ptr, buffer_ptr, length);

// Get the size of a file in words (-1 if not found)
int size = vfs_size(filename_ptr);

// Delete a file
int status = vfs_delete(filename_ptr);

// Load and run a binary from VFS; resume when it halts
int status = vfs_exec_wait(filename_ptr);
```

Return value is a VFS status code (0 = success; see the MMIO section for the full table).

---

## Project Layout

```
gocpu/
├ pkg/
│   ├ cpu/                # CPU struct, Step(), Run(), MMIO, video, VFS
│   ├ asm/                # two-pass assembler
│   └ grid/               # VRAM grid utilities
├ cmd/
│   ├ desktop/            # Ebiten-based desktop app
│   └ ccompiler/          # debug CLI: print tokens, AST, and generated assembly
├ lib/
│   ├ stdio.c             # print, strlen, strcpy, strcmp, strcat, reverse, itoa
│   ├ sys.c               # enable_interrupts, disable_interrupts, memset, memcpy
│   ├ video.c             # video_flip, draw_pixel, set_palette, mode helpers
│   └ vfs.c               # vfs_read, vfs_write, vfs_size, vfs_delete, vfs_exec_wait
├ pkg/compiler/
│   ├ token.go            # token types and TokenType constants
│   ├ lexer.go            # tokeniser (produces []Token from source string)
│   ├ ast.go              # AST node types
│   ├ parser.go           # recursive-descent parser; error messages include source snippet
│   ├ codegen.go          # assembly code generator
│   ├ optimize.go         # dead function elimination pass
│   ├ preprocessor.go     # #include and #define expansion
│   ├ symtable.go         # symbol table (globals, locals, params, structs)
│   └ compile.go          # Compile() top-level entry point
├ main.go                 # CLI entry point (native builds only)
├ wasm_wrapper.go         # WebAssembly entry point
├ Makefile
├ example.asm             # Hello World with subroutines
├ example_interrupts.asm
└ gocpu-web/              # React + Vite browser IDE
    ├ src/                # TypeScript/React source
    └ public/             # built WASM + wasm_exec.js (generated by make build)
```

---

## Testing

### Unit Tests

```bash
go test ./...
```

Tests cover the CPU, assembler, compiler (lexer, parser, codegen), and the VFS. Run a specific package:

```bash
go test ./pkg/compiler/...   # compiler tests only
go test ./pkg/cpu/...        # CPU and VFS tests
go test ./pkg/asm/...        # assembler tests
```

### End-to-End Web IDE Verification

The `verify_app.py` script uses [Playwright](https://playwright.dev/) to launch a headless browser, navigate to the Web IDE, and verify key UI elements.

```bash
# 1. Install Playwright
pip install playwright
playwright install

# 2. Start the dev server in one terminal
make dev

# 3. Run the verifier in another terminal
python verify_app.py
# Saves a screenshot to gocpu-web/screenshot.png
```

---

## Makefile Targets

| Target       | Description                                                                 |
|--------------|-----------------------------------------------------------------------------|
| `make`       | Alias for `make build`                                                      |
| `make build` | Compile `main.wasm` and copy `wasm_exec.js` into `gocpu-web/public/`       |
| `make dev`   | Build WASM and start the Vite dev server                                    |
| `make clean` | Remove `gocpu-web/public/main.wasm` and `gocpu-web/public/wasm_exec.js`    |
| `make test`  | Run `go test ./...` for all packages                                        |
