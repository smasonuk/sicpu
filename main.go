//go:build !js

package main

import (
	"flag"
	"fmt"
	"gocpu/pkg/asm"
	"gocpu/pkg/compiler"
	"gocpu/pkg/cpu"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	inPath := flag.String("in", "", "input assembly file path")
	outPath := flag.String("out", "", "output binary file path (default: input with .bin extension)")
	runProgram := flag.Bool("run", false, "run the generated binary file on the virtual CPU")
	runBinPath := flag.String("run-bin", "", "run an existing binary file on the virtual CPU")
	storagePath := flag.String("storage", "", "storage path for VFS")
	flag.Parse()

	if *runProgram && *runBinPath != "" {
		fmt.Fprintln(os.Stderr, "use either -run or -run-bin, not both")
		os.Exit(2)
	}

	assembledOutput := ""
	if *inPath != "" {
		source, err := os.ReadFile(*inPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read input file %q: %v\n", *inPath, err)
			os.Exit(1)
		}

		var code []byte
		if strings.HasSuffix(*inPath, ".c") {
			_, code, err = compiler.Compile(string(source), filepath.Dir(*inPath))
			if err != nil {
				fmt.Fprintf(os.Stderr, "compilation failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			code, _, err = asm.Assemble(string(source))
			if err != nil {
				fmt.Fprintf(os.Stderr, "assembly failed: %v\n", err)
				os.Exit(1)
			}
		}

		output := *outPath
		if output == "" {
			output = defaultOutputPath(*inPath)
		}

		if err := writeBinary(output, code); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write binary file %q: %v\n", output, err)
			os.Exit(1)
		}

		fmt.Printf("assembled %d bytes -> %s\n", len(code), output)
		assembledOutput = output
	}

	if *inPath == "" && *runBinPath == "" && !*runProgram {
		fmt.Fprintln(os.Stderr, "nothing to do: provide -in to assemble, -run to run assembled output, or -run-bin <file> to run an existing binary")
		flag.Usage()
		os.Exit(2)
	}

	runTarget := ""
	switch {
	case *runBinPath != "":
		runTarget = *runBinPath
	case *runProgram:
		if assembledOutput == "" {
			fmt.Fprintln(os.Stderr, "-run requires -in, or use -run-bin <file>")
			os.Exit(2)
		}
		runTarget = assembledOutput
	default:
		return
	}

	if err := runBinary(runTarget, *storagePath); err != nil {
		fmt.Fprintf(os.Stderr, "run failed for %q: %v\n", runTarget, err)
		os.Exit(1)
	}
}

func defaultOutputPath(inPath string) string {
	ext := filepath.Ext(inPath)
	if ext == "" {
		return inPath + ".bin"
	}
	return strings.TrimSuffix(inPath, ext) + ".bin"
}

func writeBinary(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func readBinary(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func runBinary(path string, storagePath string) error {
	loadedBytes, err := readBinary(path)
	if err != nil {
		return err
	}

	vm := cpu.NewCPU(storagePath)
	if len(loadedBytes) > len(vm.Memory) {
		return fmt.Errorf("program too large for memory: %d bytes > %d bytes", len(loadedBytes), len(vm.Memory))
	}

	copy(vm.Memory[:], loadedBytes)
	vm.Run()

	fmt.Printf(
		"run complete (%s): PC=0x%04X SP=0x%04X Z=%t N=%t R0=0x%04X R1=0x%04X R2=0x%04X R3=0x%04X\n",
		path,
		vm.PC,
		vm.SP,
		vm.Z,
		vm.N,
		vm.Regs[cpu.RegA],
		vm.Regs[cpu.RegB],
		vm.Regs[cpu.RegC],
		vm.Regs[cpu.RegD],
	)

	return nil
}
