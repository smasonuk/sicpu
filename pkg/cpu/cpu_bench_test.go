package cpu

import (
	"io"
	"testing"
)

// newSilentCPU creates a CPU that discards all MMIO output.
func newSilentCPU() *CPU {
	c := NewCPU()
	c.Output = io.Discard
	return c
}

// loadWords writes a sequence of uint16 words into CPU memory in little-endian
// byte order starting at address 0.
func loadWords(c *CPU, words ...uint16) {
	for i, w := range words {
		c.Memory[i*2] = byte(w & 0xFF)
		c.Memory[i*2+1] = byte(w >> 8)
	}
}

// BenchmarkCPU_NOP measures the raw dispatch overhead of the Step loop by
// running a tight block of NOP instructions followed by HLT.
func BenchmarkCPU_NOP(b *testing.B) {
	const nopCount = 1000

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newSilentCPU()
		for j := 0; j < nopCount; j++ {
			w := EncodeInstruction(OpNOP, 0, 0, 0)
			c.Memory[j*2] = byte(w & 0xFF)
			c.Memory[j*2+1] = byte(w >> 8)
		}
		hlt := EncodeInstruction(OpHLT, 0, 0, 0)
		c.Memory[nopCount*2] = byte(hlt & 0xFF)
		c.Memory[nopCount*2+1] = byte(hlt >> 8)
		c.Run()
	}
}

// BenchmarkCPU_ALU_ADD measures ADD instruction throughput.
func BenchmarkCPU_ALU_ADD(b *testing.B) {
	const addCount = 1000

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newSilentCPU()
		for j := 0; j < addCount; j++ {
			w := EncodeInstruction(OpADD, RegA, RegB, 0)
			c.Memory[j*2] = byte(w & 0xFF)
			c.Memory[j*2+1] = byte(w >> 8)
		}
		hlt := EncodeInstruction(OpHLT, 0, 0, 0)
		c.Memory[addCount*2] = byte(hlt & 0xFF)
		c.Memory[addCount*2+1] = byte(hlt >> 8)
		c.Regs[RegA] = 1
		c.Regs[RegB] = 1
		c.Run()
	}
}

// BenchmarkCPU_ALU_MUL measures MUL instruction throughput.
func BenchmarkCPU_ALU_MUL(b *testing.B) {
	const mulCount = 1000

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newSilentCPU()
		for j := 0; j < mulCount; j++ {
			w := EncodeInstruction(OpMUL, RegA, RegB, 0)
			c.Memory[j*2] = byte(w & 0xFF)
			c.Memory[j*2+1] = byte(w >> 8)
		}
		hlt := EncodeInstruction(OpHLT, 0, 0, 0)
		c.Memory[mulCount*2] = byte(hlt & 0xFF)
		c.Memory[mulCount*2+1] = byte(hlt >> 8)
		c.Regs[RegA] = 3
		c.Regs[RegB] = 1 // multiply by 1 keeps value alive without overflow
		c.Run()
	}
}

// BenchmarkCPU_ALU_DIV measures DIV instruction throughput.
func BenchmarkCPU_ALU_DIV(b *testing.B) {
	const divCount = 1000

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newSilentCPU()
		for j := 0; j < divCount; j++ {
			w := EncodeInstruction(OpDIV, RegA, RegB, 0)
			c.Memory[j*2] = byte(w & 0xFF)
			c.Memory[j*2+1] = byte(w >> 8)
		}
		hlt := EncodeInstruction(OpHLT, 0, 0, 0)
		c.Memory[divCount*2] = byte(hlt & 0xFF)
		c.Memory[divCount*2+1] = byte(hlt >> 8)
		c.Regs[RegA] = 60000
		c.Regs[RegB] = 1
		c.Run()
	}
}

// BenchmarkCPU_Memory_LD measures LD (load from memory) throughput.
// R1 holds the address; R0 receives the loaded value.
func BenchmarkCPU_Memory_LD(b *testing.B) {
	const ldCount = 1000

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newSilentCPU()
		for j := 0; j < ldCount; j++ {
			w := EncodeInstruction(OpLD, RegA, RegB, 0)
			c.Memory[j*2] = byte(w & 0xFF)
			c.Memory[j*2+1] = byte(w >> 8)
		}
		hlt := EncodeInstruction(OpHLT, 0, 0, 0)
		c.Memory[ldCount*2] = byte(hlt & 0xFF)
		c.Memory[ldCount*2+1] = byte(hlt >> 8)
		c.Write16(0x2000, 0xABCD)
		c.Regs[RegB] = 0x2000
		c.Run()
	}
}

// BenchmarkCPU_Memory_ST measures ST (store to memory) throughput.
// R0 holds the destination address; R1 holds the value.
func BenchmarkCPU_Memory_ST(b *testing.B) {
	const stCount = 1000

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newSilentCPU()
		for j := 0; j < stCount; j++ {
			w := EncodeInstruction(OpST, RegA, RegB, 0)
			c.Memory[j*2] = byte(w & 0xFF)
			c.Memory[j*2+1] = byte(w >> 8)
		}
		hlt := EncodeInstruction(OpHLT, 0, 0, 0)
		c.Memory[stCount*2] = byte(hlt & 0xFF)
		c.Memory[stCount*2+1] = byte(hlt >> 8)
		c.Regs[RegA] = 0x2000
		c.Regs[RegB] = 0xBEEF
		c.Run()
	}
}

// BenchmarkCPU_FILL measures the FILL opcode (bulk memset).
// Each iteration fills 1000 words.
func BenchmarkCPU_FILL(b *testing.B) {
	// FILL Ra=dst, Rb=count, Rc=value  →  three-reg encoding
	// R0=0x2000 (dst), R1=1000 (count), R2=0xFFFF (fill value)
	fill := EncodeInstruction(OpFILL, RegA, RegB, RegC)
	hlt := EncodeInstruction(OpHLT, 0, 0, 0)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newSilentCPU()
		c.Memory[0] = byte(fill & 0xFF)
		c.Memory[1] = byte(fill >> 8)
		c.Memory[2] = byte(hlt & 0xFF)
		c.Memory[3] = byte(hlt >> 8)
		c.Regs[RegA] = 0x2000 // start address
		c.Regs[RegB] = 1000   // word count
		c.Regs[RegC] = 0xFFFF // fill value
		c.Run()
	}
}

// BenchmarkCPU_COPY measures the COPY opcode (bulk memcpy).
// Each iteration copies 1000 words from 0x1000 to 0x2000.
func BenchmarkCPU_COPY(b *testing.B) {
	// COPY Ra=src, Rb=dst, Rc=count
	cpy := EncodeInstruction(OpCOPY, RegA, RegB, RegC)
	hlt := EncodeInstruction(OpHLT, 0, 0, 0)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newSilentCPU()
		c.Memory[0] = byte(cpy & 0xFF)
		c.Memory[1] = byte(cpy >> 8)
		c.Memory[2] = byte(hlt & 0xFF)
		c.Memory[3] = byte(hlt >> 8)
		// Pre-fill source region with non-zero data
		for j := uint16(0); j < 1000; j++ {
			c.Write16(0x1000+j*2, j+1)
		}
		c.Regs[RegA] = 0x1000 // src
		c.Regs[RegB] = 0x2000 // dst
		c.Regs[RegC] = 1000   // count
		c.Run()
	}
}

// BenchmarkCPU_Call_Ret measures CALL + RET round-trip overhead.
// The "function" at 0x0800 immediately returns.
func BenchmarkCPU_Call_Ret(b *testing.B) {
	const callCount = 500

	// Layout (byte addresses):
	//   0x0000 .. : CALL 0x0800, CALL 0x0800, ...  (callCount × 4 bytes each)
	//   then HLT (2 bytes)
	//   0x0800: RET (2 bytes)

	funcAddr := uint16(0x0800)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newSilentCPU()
		pos := 0
		for j := 0; j < callCount; j++ {
			call := EncodeInstruction(OpCALL, 0, 0, 0)
			c.Memory[pos] = byte(call & 0xFF)
			c.Memory[pos+1] = byte(call >> 8)
			c.Memory[pos+2] = byte(funcAddr & 0xFF)
			c.Memory[pos+3] = byte(funcAddr >> 8)
			pos += 4
		}
		hlt := EncodeInstruction(OpHLT, 0, 0, 0)
		c.Memory[pos] = byte(hlt & 0xFF)
		c.Memory[pos+1] = byte(hlt >> 8)
		ret := EncodeInstruction(OpRET, 0, 0, 0)
		c.Memory[funcAddr] = byte(ret & 0xFF)
		c.Memory[funcAddr+1] = byte(ret >> 8)
		c.Run()
	}
}

// BenchmarkCPU_Fibonacci measures CPU execution of the iterative Fibonacci
// program. Machine code is pre-built once; only the CPU loop is timed.
//
// Computes fib(20) = 6765 iteratively.
func BenchmarkCPU_Fibonacci(b *testing.B) {
	prog := buildFibProgram(20)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := newSilentCPU()
		copy(c.Memory[:], prog)
		c.Run()
	}
}

// buildFibProgram returns a byte slice that computes fib(n) iteratively,
// leaving the result in R1 (the 'a' register).
func buildFibProgram(n uint16) []byte {
	var prog []byte

	emitWord := func(w uint16) {
		prog = append(prog, byte(w&0xFF), byte(w>>8))
	}

	// Setup: R0=n, R1=0, R2=1
	emitWord(EncodeInstruction(OpLDI, RegA, 0, 0))
	emitWord(n)
	emitWord(EncodeInstruction(OpLDI, RegB, 0, 0))
	emitWord(0)
	emitWord(EncodeInstruction(OpLDI, RegC, 0, 0))
	emitWord(1)

	loopAddr := uint16(len(prog))

	// R3 = 0 + R0 (test R0 == 0)
	emitWord(EncodeInstruction(OpLDI, RegD, 0, 0))
	emitWord(0)
	emitWord(EncodeInstruction(OpADD, RegD, RegA, 0))

	// JZ done (target patched below)
	doneJZPos := len(prog)
	emitWord(EncodeInstruction(OpJZ, 0, 0, 0))
	emitWord(0) // placeholder

	emitWord(EncodeInstruction(OpMOV, RegD, RegC, 0)) // R3 = b
	emitWord(EncodeInstruction(OpADD, RegC, RegB, 0)) // b = b + a
	emitWord(EncodeInstruction(OpMOV, RegB, RegD, 0)) // a = old b
	emitWord(EncodeInstruction(OpLDI, RegD, 0, 0))
	emitWord(1)
	emitWord(EncodeInstruction(OpSUB, RegA, RegD, 0)) // n--
	emitWord(EncodeInstruction(OpJMP, 0, 0, 0))
	emitWord(loopAddr)

	doneAddr := uint16(len(prog))
	emitWord(EncodeInstruction(OpHLT, 0, 0, 0))

	// Patch the JZ target (little-endian at doneJZPos+2)
	prog[doneJZPos+2] = byte(doneAddr & 0xFF)
	prog[doneJZPos+3] = byte(doneAddr >> 8)

	return prog
}
