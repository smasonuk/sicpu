package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gocpu/pkg/compiler"
)

const testSource = `int x = 10;
int y = 20;
return x;
`

func main() {
	src := testSource
	baseDir := "."
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "read error:", err)
			os.Exit(1)
		}
		src = string(data)
		baseDir = filepath.Dir(os.Args[1])
	}

	// Preprocess
	var err error
	src, err = compiler.Preprocess(src, baseDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "preprocess error:", err)
		os.Exit(1)
	}

	fmt.Printf("Source:\n%s\n", src)

	// Lex
	tokens, err := compiler.Lex(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lex error:", err)
		os.Exit(1)
	}

	fmt.Printf("Tokens (%d)\n", len(tokens))
	for _, tok := range tokens {
		fmt.Println(" ", tok)
	}
	fmt.Println()

	// Parse
	stmts, err := compiler.Parse(tokens, src)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse error:", err)
		os.Exit(1)
	}

	fmt.Println("AST")
	for _, s := range stmts {
		fmt.Println(" ", s)
	}
	fmt.Println()

	// code Generation
	syms := compiler.NewSymbolTable()
	asm, err := compiler.Generate(stmts, syms)
	if err != nil {
		fmt.Fprintln(os.Stderr, "codegen error:", err)
		os.Exit(1)
	}

	fmt.Println("Generated Assembly")
	fmt.Print(asm)
	fmt.Println()
	fmt.Print(syms)
}
