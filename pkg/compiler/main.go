// Package compiler provides a C-subset lexer, parser, and code generator
// that targets the GoCPU 16-bit assembly language.
//
// Pipeline: C source → Lex → Parse → Generate → GoCPU assembly text
package compiler
