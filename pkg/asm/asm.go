package asm

import (
	"fmt"
	"gocpu/pkg/cpu"
	"strconv"
	"strings"
	"unicode"
)

var zeroOperandOps = map[string]uint16{
	"HLT":  cpu.OpHLT,
	"NOP":  cpu.OpNOP,
	"RET":  cpu.OpRET,
	"EI":   cpu.OpEI,
	"DI":   cpu.OpDI,
	"RETI": cpu.OpRETI,
	"WFI":  cpu.OpWFI,
}

var oneRegisterOps = map[string]uint16{
	"NOT":  cpu.OpNOT,
	"PUSH": cpu.OpPUSH,
	"POP":  cpu.OpPOP,
	"LDSP": cpu.OpLDSP,
	"STSP": cpu.OpSTSP,
}

var twoRegisterOps = map[string]uint16{
	"MOV":  cpu.OpMOV,
	"LD":   cpu.OpLD,
	"ST":   cpu.OpST,
	"ADD":  cpu.OpADD,
	"SUB":  cpu.OpSUB,
	"AND":  cpu.OpAND,
	"OR":   cpu.OpOR,
	"XOR":  cpu.OpXOR,
	"MUL":  cpu.OpMUL,
	"DIV":  cpu.OpDIV,
	"IDIV": cpu.OpIDIV,
	"SHL":  cpu.OpSHL,
	"SHR":  cpu.OpSHR,
	"LDB":  cpu.OpLDB,
	"STB":  cpu.OpSTB,
}

var threeRegisterOps = map[string]uint16{
	"FILL": cpu.OpFILL,
	"COPY": cpu.OpCOPY,
}

var regAndImmediateOps = map[string]uint16{
	"LDI": cpu.OpLDI,
}

var immediateOnlyOps = map[string]uint16{
	"JMP":  cpu.OpJMP,
	"JZ":   cpu.OpJZ,
	"JNZ":  cpu.OpJNZ,
	"JN":   cpu.OpJN,
	"JC":   cpu.OpJC,
	"JNC":  cpu.OpJNC,
	"CALL": cpu.OpCALL,
}

type Assembler struct {
	labels map[string]uint16
}

type parsedLine struct {
	lineNo   int
	labels   []string
	mnemonic string
	operands []string
}

func NewAssembler() *Assembler {
	return &Assembler{
		labels: make(map[string]uint16),
	}
}

func Assemble(code string) ([]byte, map[uint16]int, error) {
	return NewAssembler().Assemble(code)
}

func (a *Assembler) Assemble(code string) ([]byte, map[uint16]int, error) {
	lines := strings.Split(code, "\n")

	if err := a.pass1(lines); err != nil {
		return nil, nil, err
	}

	return a.pass2(lines)
}

func (a *Assembler) pass1(lines []string) error {
	var address uint32

	for i, raw := range lines {
		lineNo := i + 1
		p, err := parseLine(raw, lineNo)
		if err != nil {
			return err
		}

		for _, lbl := range p.labels {
			if address > 0xFFFF {
				return fmt.Errorf("label '%s' on line %d points past addressable memory", lbl, lineNo)
			}
			key := normalizeLabel(lbl)
			if _, exists := a.labels[key]; exists {
				return fmt.Errorf("duplicate label '%s' on line %d", lbl, lineNo)
			}
			a.labels[key] = uint16(address)
		}

		if p.mnemonic == "" {
			continue
		}

		if p.mnemonic == ".STRING" {
			if len(p.operands) != 1 {
				return fmt.Errorf(".STRING expects exactly one string operand on line %d", lineNo)
			}
			// 1 byte per character + 1 null byte
			length := uint32(len(p.operands[0]) + 1)
			if address+length > 65536 {
				return fmt.Errorf("program too large near line %d", lineNo)
			}
			address += length
			continue
		}

		if p.mnemonic == ".PSTRING" {
			if len(p.operands) != 1 {
				return fmt.Errorf(".PSTRING expects exactly one string operand on line %d", lineNo)
			}
			// Each pair of characters packs into one uint16 word (2 bytes), plus null word (2 bytes).
			runes := []rune(p.operands[0])
			length := uint32((len(runes)/2+1)*2 + 2)
			if address+length > 65536 {
				return fmt.Errorf("program too large near line %d", lineNo)
			}
			address += length
			continue
		}

		if p.mnemonic == ".ORG" {
			if len(p.operands) != 1 {
				return fmt.Errorf(".ORG expects exactly one operand on line %d", lineNo)
			}
			target, err := strconv.ParseUint(p.operands[0], 0, 32)
			if err != nil {
				return fmt.Errorf("invalid .ORG value on line %d: %s", lineNo, p.operands[0])
			}
			if target > 0xFFFF {
				return fmt.Errorf(".ORG out of range on line %d: %s", lineNo, p.operands[0])
			}
			if uint32(target) < address {
				return fmt.Errorf("cannot move origin backward on line %d", lineNo)
			}
			address = uint32(target)
			continue
		}

		if p.mnemonic == ".WORD" {
			if len(p.operands) != 1 {
				return fmt.Errorf(".WORD expects exactly one operand on line %d", lineNo)
			}
			if address+2 > 65536 {
				return fmt.Errorf("program too large near line %d", lineNo)
			}
			address += 2
			continue
		}

		length, ok := instructionLength(p.mnemonic)
		if !ok {
			return fmt.Errorf("unknown instruction on line %d: %s", lineNo, p.mnemonic)
		}

		if address+uint32(length) > 65536 {
			return fmt.Errorf("program too large near line %d", lineNo)
		}
		address += uint32(length)
	}

	return nil
}

func (a *Assembler) pass2(lines []string) ([]byte, map[uint16]int, error) {
	program := make([]byte, 0)
	sourceMap := make(map[uint16]int)

	for i, raw := range lines {
		lineNo := i + 1
		p, err := parseLine(raw, lineNo)
		if err != nil {
			return nil, nil, err
		}

		if p.mnemonic == "" {
			continue
		}

		sourceMap[uint16(len(program))] = lineNo

		if p.mnemonic == ".STRING" {
			if len(p.operands) != 1 {
				return nil, nil, fmt.Errorf(".STRING expects exactly one string operand on line %d", lineNo)
			}
			// Emit 1 byte per character + null byte
			for _, r := range p.operands[0] {
				program = append(program, byte(r))
			}
			program = append(program, 0x00)
			continue
		}

		if p.mnemonic == ".PSTRING" {
			if len(p.operands) != 1 {
				return nil, nil, fmt.Errorf(".PSTRING expects exactly one string operand on line %d", lineNo)
			}
			runes := []rune(p.operands[0])
			for i := 0; i < len(runes); i += 2 {
				char1 := uint16(runes[i])
				var char2 uint16
				if i+1 < len(runes) {
					char2 = uint16(runes[i+1])
				}
				word := char1 | (char2 << 8)
				program = append(program, byte(word&0xFF), byte(word>>8))
			}
			// Null terminator word (2 bytes)
			program = append(program, 0x00, 0x00)
			continue
		}

		mnemonic := p.mnemonic
		ops := p.operands

		if mnemonic == ".ORG" {
			if len(ops) != 1 {
				return nil, nil, fmt.Errorf(".ORG expects exactly one operand on line %d", lineNo)
			}
			target, err := strconv.ParseUint(ops[0], 0, 32)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid .ORG value on line %d: %s", lineNo, ops[0])
			}
			if target > 0xFFFF {
				return nil, nil, fmt.Errorf(".ORG out of range on line %d: %s", lineNo, ops[0])
			}
			padding := int(target) - len(program)
			if padding < 0 {
				return nil, nil, fmt.Errorf("cannot move origin backward on line %d", lineNo)
			}
			if padding > 0 {
				program = append(program, make([]byte, padding)...)
			}
			continue
		}

		if mnemonic == ".WORD" {
			if len(ops) != 1 {
				return nil, nil, fmt.Errorf(".WORD expects exactly one operand on line %d", lineNo)
			}
			val, err := a.parseImmediate(ops[0], lineNo)
			if err != nil {
				return nil, nil, err
			}
			program = append(program, byte(val&0xFF), byte(val>>8))
			continue
		}

		if opcode, ok := zeroOperandOps[mnemonic]; ok {
			if len(ops) != 0 {
				return nil, nil, fmt.Errorf("%s expects 0 operands on line %d", mnemonic, lineNo)
			}
			instr := cpu.EncodeInstruction(opcode, 0, 0, 0)
			program = append(program, byte(instr&0xFF), byte(instr>>8))
			continue
		}

		if opcode, ok := oneRegisterOps[mnemonic]; ok {
			if len(ops) != 1 {
				return nil, nil, fmt.Errorf("%s expects 1 operand on line %d", mnemonic, lineNo)
			}
			regA, err := parseRegister(ops[0], lineNo)
			if err != nil {
				return nil, nil, err
			}
			instr := cpu.EncodeInstruction(opcode, regA, 0, 0)
			program = append(program, byte(instr&0xFF), byte(instr>>8))
			continue
		}

		if opcode, ok := twoRegisterOps[mnemonic]; ok {
			if len(ops) != 2 {
				return nil, nil, fmt.Errorf("%s expects 2 operands on line %d", mnemonic, lineNo)
			}
			regA, err := parseRegister(ops[0], lineNo)
			if err != nil {
				return nil, nil, err
			}
			regB, err := parseRegister(ops[1], lineNo)
			if err != nil {
				return nil, nil, err
			}
			instr := cpu.EncodeInstruction(opcode, regA, regB, 0)
			program = append(program, byte(instr&0xFF), byte(instr>>8))
			continue
		}

		if opcode, ok := threeRegisterOps[mnemonic]; ok {
			if len(ops) != 3 {
				return nil, nil, fmt.Errorf("%s expects 3 operands on line %d", mnemonic, lineNo)
			}
			regA, err := parseRegister(ops[0], lineNo)
			if err != nil {
				return nil, nil, err
			}
			regB, err := parseRegister(ops[1], lineNo)
			if err != nil {
				return nil, nil, err
			}
			regC, err := parseRegister(ops[2], lineNo)
			if err != nil {
				return nil, nil, err
			}
			instr := cpu.EncodeInstruction(opcode, regA, regB, regC)
			program = append(program, byte(instr&0xFF), byte(instr>>8))
			continue
		}

		if opcode, ok := regAndImmediateOps[mnemonic]; ok {
			if len(ops) != 2 {
				return nil, nil, fmt.Errorf("%s expects 2 operands on line %d", mnemonic, lineNo)
			}
			regA, err := parseRegister(ops[0], lineNo)
			if err != nil {
				return nil, nil, err
			}
			imm, err := a.parseImmediate(ops[1], lineNo)
			if err != nil {
				return nil, nil, err
			}
			instr := cpu.EncodeInstruction(opcode, regA, 0, 0)
			program = append(program, byte(instr&0xFF), byte(instr>>8))
			program = append(program, byte(imm&0xFF), byte(imm>>8))
			continue
		}

		if opcode, ok := immediateOnlyOps[mnemonic]; ok {
			if len(ops) != 1 {
				return nil, nil, fmt.Errorf("%s expects 1 operand on line %d", mnemonic, lineNo)
			}
			imm, err := a.parseImmediate(ops[0], lineNo)
			if err != nil {
				return nil, nil, err
			}
			instr := cpu.EncodeInstruction(opcode, 0, 0, 0)
			program = append(program, byte(instr&0xFF), byte(instr>>8))
			program = append(program, byte(imm&0xFF), byte(imm>>8))
			continue
		}

		return nil, nil, fmt.Errorf("unknown instruction on line %d: %s", lineNo, mnemonic)
	}

	return program, sourceMap, nil
}

func parseLine(raw string, lineNo int) (parsedLine, error) {
	p := parsedLine{lineNo: lineNo}

	// 1. Check if the raw line contains the .STRING or .PSTRING directive (case-insensitive)
	upperRaw := strings.ToUpper(raw)
	pstringIdx := strings.Index(upperRaw, ".PSTRING")
	stringIdx := strings.Index(upperRaw, ".STRING")

	// Prefer .PSTRING if found (it's longer and would also match .STRING substring check)
	var directiveIdx int
	var directiveName string
	if pstringIdx != -1 {
		directiveIdx = pstringIdx
		directiveName = ".PSTRING"
	} else if stringIdx != -1 {
		directiveIdx = stringIdx
		directiveName = ".STRING"
	}

	if directiveName != "" {
		// 2. Extract any labels BEFORE the directive.
		preDirective := raw[:directiveIdx]
		if colonIdx := strings.Index(preDirective, ":"); colonIdx != -1 {
			label := strings.TrimSpace(preDirective[:colonIdx])
			if label != "" {
				p.labels = append(p.labels, label)
			}
		}

		// 3. Extract the quoted content from the ORIGINAL raw line to preserve spaces
		opening := strings.Index(raw, "\"")
		closing := strings.LastIndex(raw, "\"")
		if opening != -1 && closing != -1 && opening != closing {
			p.mnemonic = directiveName
			content := raw[opening+1 : closing]
			if unquoted, err := strconv.Unquote(`"` + content + `"`); err == nil {
				p.operands = []string{unquoted}
			} else {
				p.operands = []string{content}
			}
			return p, nil
		}
		return p, fmt.Errorf("invalid string literal on line %d", lineNo)
	}

	line := strings.TrimSpace(raw)
	if line == "" {
		return p, nil
	}

	for {
		colon := strings.IndexByte(line, ':')
		if colon <= 0 {
			break
		}

		beforeColon := strings.TrimSpace(line[:colon])
		if beforeColon == "" {
			return p, fmt.Errorf("invalid label on line %d", lineNo)
		}

		if strings.ContainsAny(beforeColon, " \t") {
			break
		}

		if !isIdentifier(beforeColon) {
			return p, fmt.Errorf("invalid label '%s' on line %d", beforeColon, lineNo)
		}

		p.labels = append(p.labels, beforeColon)
		line = strings.TrimSpace(line[colon+1:])
		if line == "" {
			return p, nil
		}
	}

	line = stripComments(line)
	line = strings.TrimSpace(line)
	if line == "" {
		return p, nil
	}

	line = normalizeInstructionText(line)
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return p, nil
	}

	p.mnemonic = strings.ToUpper(fields[0])
	if len(fields) > 1 {
		p.operands = fields[1:]
	}

	if strings.EqualFold(p.mnemonic, ".ORG") {
		p.mnemonic = ".ORG"
		if len(p.operands) != 1 {
			return p, fmt.Errorf(".ORG expects exactly one operand on line %d", lineNo)
		}
	}

	return p, nil
}

func stripComments(line string) string {
	semicolon := strings.Index(line, ";")
	doubleSlash := strings.Index(line, "//")

	cut := -1
	if semicolon >= 0 {
		cut = semicolon
	}
	if doubleSlash >= 0 && (cut == -1 || doubleSlash < cut) {
		cut = doubleSlash
	}
	if cut >= 0 {
		return line[:cut]
	}
	return line
}

func normalizeInstructionText(line string) string {
	replacer := strings.NewReplacer(",", " ", "[", " ", "]", " ")
	return replacer.Replace(line)
}

func parseRegister(token string, lineNo int) (uint16, error) {
	switch strings.ToUpper(token) {
	case "R0":
		return 0, nil
	case "R1":
		return 1, nil
	case "R2":
		return 2, nil
	case "R3":
		return 3, nil
	case "R4":
		return 4, nil
	case "R5":
		return 5, nil
	case "R6":
		return 6, nil
	case "R7":
		return 7, nil
	default:
		return 0, fmt.Errorf("invalid register '%s' on line %d", token, lineNo)
	}
}

func (a *Assembler) parseImmediate(token string, lineNo int) (uint16, error) {
	if value, err := strconv.ParseUint(token, 0, 32); err == nil {
		if value > 0xFFFF {
			return 0, fmt.Errorf("immediate out of range on line %d: %s", lineNo, token)
		}
		return uint16(value), nil
	}

	label := normalizeLabel(token)
	if addr, ok := a.labels[label]; ok {
		return addr, nil
	}

	if isIdentifier(token) {
		return 0, fmt.Errorf("undefined label '%s' on line %d", token, lineNo)
	}

	return 0, fmt.Errorf("invalid immediate '%s' on line %d", token, lineNo)
}

// instructionLength returns the byte length of an instruction.
// All instructions are 2 bytes; instructions with an immediate are 4 bytes.
func instructionLength(mnemonic string) (uint16, bool) {
	mnemonic = strings.ToUpper(mnemonic)

	if _, ok := zeroOperandOps[mnemonic]; ok {
		return 2, true
	}
	if _, ok := oneRegisterOps[mnemonic]; ok {
		return 2, true
	}
	if _, ok := twoRegisterOps[mnemonic]; ok {
		return 2, true
	}
	if _, ok := threeRegisterOps[mnemonic]; ok {
		return 2, true
	}
	if _, ok := regAndImmediateOps[mnemonic]; ok {
		return 4, true
	}
	if _, ok := immediateOnlyOps[mnemonic]; ok {
		return 4, true
	}
	return 0, false
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}

	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
			continue
		}

		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}

	return true
}

func normalizeLabel(label string) string {
	return strings.ToUpper(label)
}
