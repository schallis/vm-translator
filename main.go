package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// The line struct stores information about the lines we are translating
type Instruction struct {
	raw string

	// computed values (by NewLine constructor)
	stripped        string
	empty           bool     // default: false
	translatedLines []string // The resulting translations

	// Parsed values
	operation string // push, pop, `function`
	segment   string
	value     int
}

// Constructor for the Instruction type
func NewInstruction(rawline string) Instruction {
	line := Instruction{
		raw: rawline,
	}
	line.clean()

	return line
}

// Add a translated ASM code lines to our instruction (can also be a comment)
func (l *Instruction) outputLines(lines ...string) {
	l.translatedLines = append(l.translatedLines, lines...)
}

func (l *Instruction) clean() {
	// Strip trailing comments
	before, _, _ := strings.Cut(l.raw, "//")

	// Check for empty line
	if len(before) == 0 {
		l.empty = true
	} else {
		l.stripped = before
	}
}

func validateOperation(operation string) bool {
	switch operation {
	case "push":
	case "pop":
	case "add":
	case "sub":
	default:
		return false // Not one of allowed operation
		// "eq",
		// "lt",
		// "gt",
		// "neg",
		// "or",
		// "not",
		// "and",
	}
	return true
}

func validateSegment(segment string) bool {
	switch segment {
	case "local":
	case "constant":
	case "static":
	case "pointer":
	case "this":
	case "that":
	case "temp":
	case "argument":
	default:
		return false // Not one of allowed segments
	}
	return true
}

// Parse instruction, tokenize and validate tokens
func (l *Instruction) parse() error {
	if l.empty {
		return nil
	}

	// Should be either 1 or 3 tokens separated by spaces
	tokens_w_empty := strings.Split(l.stripped, " ")

	// Multiple spaces will result in empty tokens, so eliminate those
	tokens := filterBlanks(tokens_w_empty)
	num_t := len(tokens)

	l.operation = tokens[0]
	if ok := validateOperation(l.operation); !ok {
		return fmt.Errorf("undefined operation type %v", l.operation)
	}

	switch num_t {
	case 1:
		// is a function, operation already captured
	case 3:
		// is a push or pop
		l.segment = tokens[1]
		if ok := validateSegment(l.segment); !ok {
			return fmt.Errorf("undefined segment type %v", l.segment)
		}

		val, err := strconv.ParseInt(tokens[2], 10, 16)
		if err != nil {
			return fmt.Errorf("invalid value %v got err %v", tokens[2], err)
		}
		l.value = int(val)
	default:
		return fmt.Errorf("invalid instruction, has %v tokens", num_t)
	}

	return nil
}

func (instr *Instruction) Translate() {
	/*
		RAM[0]		SP points to next topmost location in stack
		RAM[1]		LCL points to base of `local` segment
		RAM[2]		ARG points to base of `argument` segment
		RAM[3]		THIS points to base of `this` segment
		RAM[4]		THAT points to base of `that` segment
		RAM[5-12] 	Holds contents of `temp` segment, 8 values
		RAM[13-15]	Can be used by VM as general purpose
		RAM[256]	Start of global stack
	*/
	segmentMap := map[string]string{
		"local":    "LCL",
		"argument": "ARG",
		"this":     "THIS",
		"that":     "THAT",
	}

	switch instr.operation {
	case "push":
		switch instr.segment {
		case "local", "argument", "this", "that":
			// e.g. push local 2
			instr.outputLines(
				// *addr=LCL+2
				// Compute the address and store in @addr
				fmt.Sprintf("@%d", instr.value),
				"D=A",
				fmt.Sprintf("@%v", segmentMap[instr.segment]),
				"A=M",
				"D=D+A",
				// *SP=*addr
				"A=D",
				"D=M",
				"@SP",
				"A=M",
				"M=D",
				// SP++
				"@SP",
				"M=M+1",
			)
		case "constant":
			// e.g. push constant 17
			instr.outputLines(
				// *SP=17
				// Assign our value to our SP location
				fmt.Sprintf("@%d", instr.value),
				"D=A",
				"@SP",
				"A=M",
				"M=D",
				// SP++
				// Increment the SP
				"@SP",
				"M=M+1",
			)
		case "temp":
			// addr=5+i, *SP=*addr, SP++
			instr.outputLines(
				// addr=5+i
				fmt.Sprintf("@%d", instr.value+5),
				"D=M",
				// *SP=*addr
				"@SP",
				"A=M",
				"M=D",
				// SP++
				"@SP",
				"M=M+1",
			)
		case "static":
			// Translate `static i` into  `@Foo.i` in Foo.vm
			instr.outputLines("// UNDEF")
		case "pointer":
			// pointer 0/1 -> *SP=THIS/THAT, SP++
			thisthat := "THIS"
			if instr.value == 1 {
				thisthat = "THAT"
			}

			instr.outputLines(
				// THIS/THAT=*SP
				fmt.Sprintf("@%v", thisthat),
				"D=M",
				"@SP",
				"A=M",
				"D=M",
				// SP++
				"@SP",
				"M=M+1",
			)
		}

	case "pop":
		switch instr.segment {
		case "local", "argument", "this", "that":
			// All of these segments are processed the same way
			// e.g. pop local i
			// addr=LCL+i, SP--, *addr=*SP
			segCode := segmentMap[instr.segment]
			instr.outputLines(
				// addr=LCL+i
				fmt.Sprintf("@%d", instr.value),
				"D=A",
				fmt.Sprintf("@%v", segCode), // Get Base address
				"A=M",
				"D=D+A", // Add value offset e.g. 300+i
				fmt.Sprintf("@%v", segCode),
				"M=D", // Set Mem loc corresponding to segment to computed val
				// SP--
				"@SP",
				"M=M-1",
				// *addr=*SP
				"A=M",
				"D=M",
				fmt.Sprintf("@%v", segCode),
				"A=M",
				"M=D",
				fmt.Sprintf("@%v", instr.value),
				"D=A",
				fmt.Sprintf("@%v", segCode),
				"A=M",
				"D=A-D",
				fmt.Sprintf("@%v", segCode),
				"M=D",
			)
		case "constant":
			log.Fatalf("`pop constant` not implemented, doesn't make sense")
		case "static":
			// Translate `static i` into  `@Foo.i` in Foo.vm
			instr.outputLines("// UNDEF")
		case "temp":
			// addr=5+i, SP--, *addr=*SP
			instr.outputLines(
				// SP--
				"@SP",
				"M=M-1",
				// *addr=*SP
				"A=M",
				"D=M",
				// addr=i+5
				fmt.Sprintf("@%d", instr.value+5),
				"M=D", // RAM[addr] = @SP
			)
		case "pointer":
			// pointer 0/1 -> SP--, THIS/THAT=*SP
			thisthat := "THIS"
			if instr.value == 1 {
				thisthat = "THAT"
			}

			instr.outputLines(
				// SP--
				"@SP",
				"M=M-1",
				// THIS/THAT=*SP
				"@SP",
				"D=M",
				fmt.Sprintf("@%v", thisthat),
				"M=D",
			)
		}
	case "add":
		// Take top two stack variables and perform add
		instr.outputLines(
			// Find vals and compute Sum
			"@SP",
			"A=M",   // SP address
			"A=A-1", // SP -1 address
			"A=A-1", // SP -2 address
			"D=M",   // Store SP -2 data in D register
			"A=A+1", // SP -1 address
			"D=D+M", // Store SP -2 data + SP -1 data
			// Retract SP by 2 and store val
			"@SP",
			"M=M-1",
			"M=M-1",
			"A=M",
			"M=D",
			// Advance SP by 1
			"@SP",
			"M=M+1",
		)
	case "sub":
		// Take top two stack variables and perform sub
		instr.outputLines(
			"@SP",
			"A=M",   // SP address
			"A=A-1", // SP -1 address
			"A=A-1", // SP -2 address
			"D=M",   // Store SP -2 data in D register
			"A=A+1", // SP -1 address
			"D=D-M", // Store SP -2 data + SP -1 data
			// Retract SP by 2 and store val
			"@SP",
			"M=M-1",
			"M=M-1",
			"A=M",
			"M=D",
			// Advance SP by 1
			"@SP",
			"M=M+1",
		)
	}
}

// Read a .vm file specified as the only argument
// Translate and produce a .asm file in the same folder as run
func main() {
	var err error
	log.SetPrefix("debug: ")
	log.SetFlags(0)

	// Read the args for the filename .asm file
	args := os.Args
	inSuffix := ".vm"
	filename := ""
	if len(args) < 2 || args[1] == "" {
		filename = "input.vm"
		// filename = "materials/pong/Pong.asm"
		log.Printf("No filename specified as first arg. Defaulting to %v", filename)
	} else {
		filename = args[1]
	}

	// Compute file metadata
	dir := filepath.Dir(filename)                  // Directory we're reading/writing in
	base := filepath.Base(filename)                // Input base filename
	basename := strings.TrimSuffix(base, inSuffix) // Input filename without suffix

	// Open file
	file, err := os.Open(filename)
	check(err)
	defer file.Close()

	// Scan through it line by line
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	// Start translation
	log.Println("Starting translation")
	var processedInstructions []*Instruction
	for scanner.Scan() {
		text := scanner.Text()
		inLine := NewInstruction(text)
		err := inLine.parse()
		if err != nil {
			log.Fatalf(err.Error())
		}

		// Only store line if has valid instruction
		if !inLine.empty {
			inLine.Translate()
			processedInstructions = append(processedInstructions, &inLine)
		}
	}

	// Open output file for writing
	log.Println("Writing output")
	filenameo := filepath.Join(dir, basename+".asm")
	ofile, err := os.Create(filenameo)
	check(err)
	defer ofile.Close()

	// Write each line token as a line in the output file
	w := bufio.NewWriter(ofile)
	var newline = "\n"
	for instrNum, instr := range processedInstructions {
		// Omit newline if last line of file or if empty line

		DEBUG := true
		// Output command with original line num and instruction
		if DEBUG {
			comment := fmt.Sprintf("// %v\n", instr.stripped)
			_, err = w.WriteString(comment)
			check(err)
		}

		// Output translated lines
		for tNum, tLine := range instr.translatedLines {

			// Omit newline if last line of last instruction
			if tNum == len(instr.translatedLines)-1 && instrNum == len(processedInstructions)-1 {
				newline = ""
			}

			line := fmt.Sprintf("%v%v", tLine, newline)
			_, err = w.WriteString(line)
			check(err)
		}
		w.WriteString(newline)
	}
	log.Println("Output to", filenameo)
	w.Flush()
}
