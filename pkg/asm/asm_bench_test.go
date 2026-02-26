package asm

import "testing"

// Registers: R0, R1, R2, R3 (no RA/RB aliases in this assembler).
// LD Rdst, Rsrc  — loads from address in Rsrc into Rdst
// ST Raddr, Rval — stores Rval into address in Raddr

// smallProgram is a ~20-instruction counter loop.
const smallProgram = `
    LDI R0, 10
    LDI R1, 0
loop:
    ADD R1, R0
    LDI R2, 1
    SUB R0, R2
    JNZ loop
    HLT
`

// mediumProgram is a ~60-instruction program with multiple subroutines,
// labels, and a .STRING directive.
const mediumProgram = `
    JMP main

abs_fn:
    LDI R1, 0
    SUB R1, R0
    JN  abs_neg
    RET
abs_neg:
    NOT R0
    LDI R1, 1
    ADD R0, R1
    RET

double_fn:
    ADD R0, R0
    RET

triple_fn:
    PUSH R0
    CALL double_fn
    POP  R1
    ADD  R0, R1
    RET

count_down:
    LDI R1, 1
cd_loop:
    LDI R2, 0
    ADD R2, R0
    JZ  cd_done
    SUB R0, R1
    JMP cd_loop
cd_done:
    RET

main:
    LDI R0, 7
    CALL abs_fn
    PUSH R0

    LDI R0, 5
    CALL triple_fn
    PUSH R0

    LDI R0, 12
    CALL count_down

    POP R1
    POP R2
    ADD R1, R2
    LDI R3, 0x3000
    ST  R3, R1
    HLT

greeting:
    .STRING "Hello, World!"
`

// largeProgram is a ~300-instruction assembly program representative of
// typical compiler output: multiple functions, nested loops, memory access.
const largeProgram = `
    JMP main

; ---- iterative fibonacci: R0=n, returns fib(n) in R0 ----
fib:
    LDI R1, 0
    LDI R2, 1
fib_loop:
    LDI R3, 0
    ADD R3, R0
    JZ  fib_done
    MOV R3, R2
    ADD R2, R1
    MOV R1, R3
    LDI R3, 1
    SUB R0, R3
    JMP fib_loop
fib_done:
    MOV R0, R1
    RET

; ---- sum array at 0x1000, length in R1, result in R0 ----
sum_array:
    LDI R2, 0x1000
    LDI R0, 0
sum_loop:
    LDI R3, 0
    ADD R3, R1
    JZ  sum_done
    LD  R3, R2
    ADD R0, R3
    LDI R3, 1
    ADD R2, R3
    SUB R1, R3
    JMP sum_loop
sum_done:
    RET

; ---- fill 0x2000..0x2000+R1 with R0 ----
fill_array:
    LDI R2, 0x2000
fill_loop:
    LDI R3, 0
    ADD R3, R1
    JZ  fill_done
    ST  R2, R0
    LDI R3, 1
    ADD R2, R3
    SUB R1, R3
    JMP fill_loop
fill_done:
    RET

; ---- copy 0x1000->0x2000, R1 words ----
copy_array:
    LDI R2, 0x1000
    LDI R3, 0x2000
copy_loop:
    LDI R0, 0
    ADD R0, R1
    JZ  copy_done
    LD  R0, R2
    ST  R3, R0
    LDI R0, 1
    ADD R2, R0
    ADD R3, R0
    SUB R1, R0
    JMP copy_loop
copy_done:
    RET

; ---- simple bubble sort of 0x1000, R1 words ----
bubble_sort:
    PUSH R1
    LDI R3, 1
    SUB R1, R3
bs_outer:
    LDI R3, 0
    ADD R3, R1
    JZ  bs_done
    LDI R2, 0x1000
    PUSH R1
bs_inner:
    POP  R1
    LDI R3, 0
    ADD R3, R1
    PUSH R1
    JZ  bs_inner_done
    LD   R0, R2
    LDI  R3, 1
    ADD  R3, R2
    PUSH R0
    LD   R0, R3
    POP  R1
    SUB  R0, R1
    JN   bs_no_swap
    LD   R0, R2
    PUSH R0
    LDI  R3, 1
    ADD  R3, R2
    LD   R0, R3
    ST   R2, R0
    POP  R0
    LDI  R3, 1
    ADD  R3, R2
    ST   R3, R0
bs_no_swap:
    POP  R1
    LDI  R3, 1
    SUB  R1, R3
    PUSH R1
    LDI  R3, 1
    ADD  R2, R3
    JMP  bs_inner
bs_inner_done:
    POP  R1
    POP  R1
    LDI  R3, 1
    SUB  R1, R3
    JMP  bs_outer
bs_done:
    POP R1
    RET

; ---- multiply R0 * R1, result in R0 ----
multiply:
    LDI R2, 0
    LDI R3, 0
mul_loop:
    ADD R3, R1
    JZ  mul_done
    ADD R2, R0
    LDI R0, 1
    SUB R3, R0
    JMP mul_loop
mul_done:
    MOV R0, R2
    RET

; ---- absolute value of R0 ----
abs_fn:
    LDI R1, 0
    SUB R1, R0
    JN  abs_neg
    RET
abs_neg:
    NOT R0
    LDI R1, 1
    ADD R0, R1
    RET

main:
    LDI R0, 0x1000
    LDI R1, 16
    LDI R2, 1
init_loop:
    LDI R3, 0
    ADD R3, R1
    JZ  init_done
    ST  R0, R2
    LDI R3, 1
    ADD R0, R3
    ADD R2, R3
    SUB R1, R3
    JMP init_loop
init_done:

    LDI R0, 10
    CALL fib
    PUSH R0

    LDI R1, 16
    CALL sum_array
    PUSH R0

    LDI R0, 0xABCD
    LDI R1, 32
    CALL fill_array

    LDI R1, 16
    CALL copy_array

    LDI R1, 16
    CALL bubble_sort

    LDI R0, 7
    LDI R1, 6
    CALL multiply
    CALL abs_fn
    PUSH R0

    POP R2
    POP R1
    POP R0
    ADD R0, R1
    ADD R0, R2
    LDI R1, 0x3000
    ST  R1, R0
    HLT

msg:
    .STRING "Benchmark complete"
`

func BenchmarkAssemble_Small(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, err := Assemble(smallProgram)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAssemble_Medium(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, err := Assemble(mediumProgram)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAssemble_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, err := Assemble(largeProgram)
		if err != nil {
			b.Fatal(err)
		}
	}
}
