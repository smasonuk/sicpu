package compiler

import (
	"fmt"
	"gocpu/pkg/asm"

	"os"
)

func Compile(src string, baseDir string) (*string, []byte, error) {

	// Preprocess
	var err error
	src, err = Preprocess(src, baseDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "preprocess error:", err)
		return nil, nil, err
	}

	// fmt.Printf("Source:\n%s\n", src)

	tokens, err := Lex(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lex error:", err)
		return nil, nil, err
	}

	stmts, err := Parse(tokens, src)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse error:", err)
		return nil, nil, err
	}

	syms := NewSymbolTable()
	assembly, err := Generate(stmts, syms)
	if err != nil {
		fmt.Fprintln(os.Stderr, "codegen error:", err)
		return nil, nil, err
	}

	// fmt.Println("Assembly:\n", assembly)

	machineCode, _, err := asm.Assemble(assembly)
	if err != nil {
		return &assembly, nil, fmt.Errorf("assembly error: %v", err)
	}

	return &assembly, machineCode, nil

}
