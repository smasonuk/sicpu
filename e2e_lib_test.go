package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"gocpu/pkg/asm"
	"gocpu/pkg/compiler"
	"gocpu/pkg/cpu"
)

func TestFileIOApp(t *testing.T) {
	// 1. Read source
	srcPath := "_capps/file_io_test.c"
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("Failed to read source: %v", err)
	}
	src := string(srcBytes)

	// 2. Preprocess
	processed, err := compiler.Preprocess(src, "./_capps")
	if err != nil {
		t.Fatalf("Preprocess failed: %v", err)
	}

	// 3. Compile
	tokens, err := compiler.Lex(processed)
	if err != nil {
		t.Fatalf("Lex failed: %v", err)
	}
	stmts, err := compiler.Parse(tokens, processed)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	syms := compiler.NewSymbolTable()
	assembly, err := compiler.Generate(stmts, syms)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 4. Assemble
	machineCode, _, err := asm.Assemble(assembly)
	if err != nil {
		t.Fatalf("Assemble failed: %v\nAssembly:\n%s", err, assembly)
	}

	// 5. Run
	vm := cpu.NewCPU()
	if len(machineCode) > len(vm.Memory) {
		t.Fatalf("Program too large")
	}
	copy(vm.Memory[:], machineCode)

	var output bytes.Buffer
	vm.Output = &output

	// Run until halted
	for i := 0; i < 20000; i++ {
		if vm.Halted {
			break
		}
		vm.Step()
	}

	if !vm.Halted {
		t.Errorf("VM did not halt")
	}

	// 6. Verify Output
	outStr := output.String()
	expectedFragments := []string{
		"VFS Test Start",
		"Buffer before save: HELLO",
		"Save Success",
		"Clearing buffer...",
		"Buffer after clear: ",
		"Load Success",
		"Buffer after load: HELLO",
		"VFS Test Done",
	}

	for _, frag := range expectedFragments {
		if !strings.Contains(outStr, frag) {
			t.Errorf("Output missing %q. Got:\n%s", frag, outStr)
		}
	}

	// Check if file exists in VFS
	if _, err := vm.Disk.Read("TEST.TXT"); err != nil {
		t.Errorf("File TEST.TXT not found in VFS")
	}
}
