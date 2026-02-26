# Development Log

## Session: Implement Compiler Intrinsics and String Literals

### 1. `print(ptr)` Compiler Intrinsic

**Objective:**
Extend the C compiler to support a built-in `print` function that takes a pointer to a null-terminated string and writes it to the standard output (`0xFF00`).

**Changes:**
- **Codegen (`compiler/codegen.go`):**
  - Added a case for `"print"` in `genExpr`.
  - Implemented an assembly loop that:
    - Loads the character from the address in `R1`.
    - Checks for null termination (0).
    - Writes the character to `0xFF00`.
    - Increments the pointer.
    - Loops until null is encountered.

- **Documentation (`README.md`):**
  - Updated the "C-subset compiler" section to list `print(int* ptr)` as a supported intrinsic.

- **Testing:**
  - Added `TestGenerate_PrintIntrinsic` to `compiler/codegen_test.go`.
  - Verified end-to-end functionality with a custom test case.

### 2. String Literal Support

**Objective:**
Allow string literals (e.g., `"Hello World"`) to be used directly in C code, automatically handling storage allocation and pooling.

**Changes:**
- **Lexer (`compiler/lexer.go`, `compiler/token.go`):**
  - Added `STRING` token type.
  - Implemented `scanString()` to handle double-quoted strings with escape sequences (`\n`, `\t`, `\"`, `\\`).

- **Parser (`compiler/parser.go`, `compiler/ast.go`):**
  - Added `StringLiteral` struct to the AST.
  - Updated `parsePrimary()` to produce `StringLiteral` nodes when encountering `STRING` tokens.

- **Codegen (`compiler/codegen.go`):**
  - Added `stringPool map[string]string` to `CodeGen` struct to map string values to unique labels (`S0`, `S1`, etc.).
  - Updated `genExpr` to handle `StringLiteral`:
    - Checks if the string exists in the pool; if not, assigns a new label.
    - Emits `LDI R0, LABEL` to load the string's address.
  - Updated `Generate` to emit all pooled strings as `.STRING` directives at the end of the assembly output (after the final `HLT`).

- **Testing:**
  - Created `compiler/string_literal_test.go` covering:
    - Basic string literals.
    - String pooling (reusing labels for identical strings).
    - Escape sequence handling.

---

## Session: Implement `print_packed` Intrinsic and Two-Register Shifts

### 1. `print_packed(ptr)` Compiler Intrinsic

**Objective:**
Add a `print_packed` intrinsic for printing packed 8-bit strings, where two ASCII characters are stored per 16-bit word (low byte = first character, high byte = second character).

**Changes:**
- **Codegen (`compiler/codegen.go`):**
  - Added a case for `"print_packed"` in `genExpr`.
  - Implemented an assembly loop that:
    - Loads a word from the pointer in `R1`.
    - Checks for a null word (both bytes zero) and exits.
    - Extracts the low byte, checks for null, and writes it to `0xFF00`.
    - Extracts the high byte (via `SHR R0, R2` where R2=8), checks for null, and writes it to `0xFF00`.
    - Increments the pointer by 1 word and loops.

### 2. Two-Register Shift Instructions

**Objective:**
Update `SHL` and `SHR` to accept two register operands so the shift amount is specified by a register value.

**Changes:**
- **CPU (`pkg/cpu/cpu.go`):** `OpSHL` and `OpSHR` now use `regA << regB` / `regA >> regB`.
- **Assembler (`pkg/asm/asm.go`):** Moved `SHL`/`SHR` to `twoRegisterOps` (syntax: `SHL Ra, Rb`).
- **Documentation (`README.md`):** Moved `SHL`/`SHR` from the "One register" table to the "Two registers" table with updated descriptions.

---

## Session: Implement Standard Library String Functions

### 1. String Utilities in `lib/stdio.c`

**Objective:**
Expand the C standard library with common string manipulation functions to support more complex applications.

**Changes:**
- **Standard Library (`lib/stdio.c`):**
  - Implemented `strlen(byte* str)`: Calculates string length.
  - Implemented `strcpy(byte* dest, byte* src)`: Copies a string.
  - Implemented `strcmp(byte* s1, byte* s2)`: Compares two strings lexicographically.
  - Implemented `strcat(byte* dest, byte* src)`: Concatenates two strings.
  - Implemented `reverse(byte* s)`: Reverses a string in place.
  - Implemented `itoa(int n, byte* s)`: Converts an integer to its string representation (handles negative numbers).

- **Testing:**
  - Created `_capps/test_stdio_strings.c` to verify all new functions.
  - Verified correctness with `go run cmd/console/main.go`.
