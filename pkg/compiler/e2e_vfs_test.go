package compiler

import (
	"gocpu/pkg/asm"
	"gocpu/pkg/cpu"
	"testing"
)

func runCodeWithVFS(t *testing.T, source string, setupVFS func(*cpu.CPU)) *cpu.CPU {
	// Preprocess
	processed, err := Preprocess(source, ".")
	if err != nil {
		t.Fatalf("Preprocess failed: %v", err)
	}

	// Lex -> Parse -> Generate
	tokens, err := Lex(processed)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}
	stmts, err := Parse(tokens, processed)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	syms := NewSymbolTable()
	assembly, err := Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Assemble
	machineCode, _, err := asm.Assemble(assembly)
	if err != nil {
		t.Fatalf("Assemble failed: %v\nAssembly:\n%s", err, assembly)
	}

	// Run
	vm := cpu.NewCPU()
	if setupVFS != nil {
		setupVFS(vm)
	}

	if len(machineCode) > len(vm.Memory) {
		t.Fatalf("Program too large")
	}
	copy(vm.Memory[:], machineCode)

	// Run with a limit to avoid infinite loops
	for i := 0; i < 10000; i++ {
		if vm.Halted {
			break
		}
		vm.Step()
	}

	return vm
}

func TestVFS_WriteRead(t *testing.T) {
	src := `
	#include <vfs.c>

	int main() {
		// Write "Hello" to test.txt
		char buf[6];
		buf[0] = 72; // H
		buf[1] = 101; // e
		buf[2] = 108; // l
		buf[3] = 108; // l
		buf[4] = 111; // o
		buf[5] = 0;

		int err = vfs_write("test.txt", buf, 5);
		if (err != 0) return err;

		// Read back
		char readBuf[10];
		err = vfs_read("test.txt", readBuf);
		if (err != 0) return err;

		return readBuf[0]; // Should be 'H' (72)
	}
	`

	vm := runCodeWithVFS(t, src, nil)

	if vm.Regs[0] != 72 {
		t.Errorf("Expected 'H' (72), got %d. VFS Status: %d", vm.Regs[0], vm.Read16(0xFF14))
	}

	// Verify file content in VFS
	data, err := vm.Disk.Read("test.txt")
	if err != nil {
		t.Fatalf("File not found in VFS: %v", err)
	}
	if string(data) != "Hello" {
		t.Errorf("Expected file content 'Hello', got %q", string(data))
	}
}

func TestVFS_Delete(t *testing.T) {
	src := `
	#include <vfs.c>

	int main() {
		int err = vfs_delete("existing.txt");
		return err;
	}
	`

	vm := runCodeWithVFS(t, src, func(vm *cpu.CPU) {
		vm.Disk.Write("existing.txt", []byte("content"))
	})

	if vm.Regs[0] != 0 {
		t.Errorf("Expected status 0 (success), got %d", vm.Regs[0])
	}

	_, err := vm.Disk.Read("existing.txt")
	if err == nil {
		t.Errorf("File should have been deleted")
	}
}
