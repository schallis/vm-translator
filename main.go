package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Stack struct {
	data [1024]int
	sp   int // default 0
}

func (s *Stack) Pop() (int, error) {
	if s.sp <= 0 {
		return 0, errors.New("SP already at 0")
	}
	s.sp -= 1
	return s.data[s.sp], nil
}

func (s *Stack) Push(val int) {
	s.data[s.sp] = val
	s.sp += 1
}

// The line struct stores information about the lines we are translating
type Instruction struct {
	raw string

	// computed values (by NewLine constructor)
	stripped        string
	empty           bool // default: false
	lineNum         int  // Input linenum
	translatedLines []string

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

func (l *Instruction) isValid() bool {
	return true
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

// Filter empty strings from slice of strings
func filterBlanks(slice []string) []string {
	var filtered = []string{}
	for _, t := range slice {
		if t != "" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// Parse instruction, tokenize and validate tokens
func (l *Instruction) parse() error {
	if l.empty {
		// log.Println("Empty line, not translated")
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

// Utility function for error handling
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Take a line struct, translate it into binary and store translation
// e.g. MD=A-1;JGE -> 1110110010011011
func (instr *Instruction) Translate(m *Stack) {
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
				"D=D+M",
				"@addr",
				"M=D",
				// *SP=*addr
				"@addr",
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
				// *addr=LCL+2
				// Compute the address and store in @addr
				fmt.Sprintf("@%d", instr.value),
				"D=A+5", // Might not be valid +5
				// "@5 // Not sure",
				// "D=D+5 // Not sure",
				"@addr",
				"M=D",
				// *SP=*addr
				"@addr",
				"D=M",
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
			// e.g. pop local 2
			// addr=LCL+2, SP--, *addr=*SP
			instr.outputLines(
				// addr=LCL+2
				// Compute the address and store in @addr
				fmt.Sprintf("@%d", instr.value),
				"D=A",
				fmt.Sprintf("@%v", segmentMap[instr.segment]),
				"D=D+M",
				"@addr",
				"M=D",
				// SP--
				// Decrement the SP
				"@SP",
				"M=M-1",
				// *addr=*SP
				// Assign the SP value to our addr value
				"@SP",
				"D=M",
				"@addr",
				"A=M", // write to memory using pointer
				"M=D", // RAM[addr] = @SP
			)
		case "constant":
			log.Fatalf("`pop constant` not implemented, doesn't make sense")
		case "static":
			// Translate `static i` into  `@Foo.i` in Foo.vm
			instr.outputLines("// UNDEF")
		case "temp":
			// addr=5+i, SP--, *addr=*SP
			instr.outputLines(
				// addr=5+i
				// Compute the address and store in @addr
				fmt.Sprintf("@%d", instr.value),
				"D=A+5", // Might not be valid +5
				// "@5 // Not sure",
				// "D=D+5 // Not sure",
				"@addr",
				"M=D",
				// SP--
				// Decrement the SP
				"@SP",
				"M=M-1",
				// *addr=*SP
				// Assign the SP value to our addr value
				"@SP",
				"D=M",
				"@addr",
				"A=M", // write to memory using pointer
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
			// SP--
			"@SP",
			"M=M-1",
			// @one=*SP
			"@SP",
			"D=M",
			"@one",
			"M=D",
			// SP--
			"@SP",
			"M=M-1",
			// @two=*SP
			"@SP",
			"D=M",
			"@two",
			"M=D",
			// *SP=@one+@two
			"@one",
			"D=M",
			"@two",
			"D=D+M",
			"@SP",
			"M=D",
		)
	case "sub":
		// Take top two stack variables and perform sub
		instr.outputLines(
			// SP--
			"@SP",
			"M=M-1",
			// @one=*SP
			"@SP",
			"D=M",
			"@one",
			"M=D",
			// SP--
			"@SP",
			"M=M-1",
			// @two=*SP
			"@SP",
			"D=M",
			"@two",
			"M=D",
			// *SP=@one-@two
			"@one",
			"D=M",
			"@two",
			"D=D-M",
			"@SP",
			"M=D",
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
	filename := ""
	if len(args) < 2 || args[1] == "" {
		filename = "input.vm"
		// filename = "materials/pong/Pong.asm"
		log.Printf("No filename specified as first arg. Defaulting to %v", filename)
	} else {
		filename = args[1]
	}

	// Open file
	file, err := os.Open(filename)
	check(err)
	defer file.Close()

	// Define memory stack
	var stack = Stack{}

	// Scan through it line by line
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	// Start translation
	log.Println("Starting translation")
	var processedInstructions []*Instruction
	lineNum := 0
	for scanner.Scan() {
		text := scanner.Text()
		inLine := NewInstruction(text)
		err := inLine.parse()
		if err != nil {
			log.Fatalf(err.Error())
		}
		inLine.lineNum = lineNum

		// Only store line if has valid instruction
		if inLine.isValid() && !inLine.empty {
			lineNum += 1
			inLine.Translate(&stack)
			processedInstructions = append(processedInstructions, &inLine)
		}
	}

	// Open output file for writing
	log.Println("Writing output")
	filenameo := "output.asm"
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
			comment := fmt.Sprintf("// L%-3v %v\n", instr.lineNum, instr.stripped)
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
