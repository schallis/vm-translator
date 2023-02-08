package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// The line struct stores information about the lines we are translating
type Instruction struct {
	raw string

	// computed values (by NewLine constructor)
	stripped   string
	empty      bool // default: false
	lineNum    int  // Input linenum
	translated string

	// Parsed values
	operation string // push, pop, `function`
	register  string
	value     int
}

// Constructor for the Line type
func NewInstruction(rawline string) Instruction {
	line := Instruction{
		raw: rawline,
	}
	line.clean()

	return line
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

func validateRegister(register string) bool {
	switch register {
	case "local":
	case "constant":
	case "static":
	case "pointer":
	case "this":
	case "that":
	case "temp":
	default:
		return false // Not one of allowed registers
	}
	return true
}

// Parse instruction, tokenize and validate tokens
func (l *Instruction) parse() error {
	if l.empty {
		log.Println("Empty line, not translated")
		return nil
	}

	// Should be either 1 or 3 tokens separated by spaces
	tokens := strings.Split(l.stripped, " ")
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
		l.register = tokens[1]
		if ok := validateRegister(l.register); !ok {
			return fmt.Errorf("undefined register type %v", l.register)
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
func (line *Instruction) Translate() {
	line.translated = fmt.Sprintf("%v %v %d", line.operation, line.register, line.value)
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

		// Store line for second pass with computed line number
		if inLine.isValid() {
			lineNum += 1
			inLine.Translate()
			processedInstructions = append(processedInstructions, &inLine)
		}
	}

	// Open output file for writing
	log.Println("Writing output")
	filenameo := "output.hack"
	ofile, err := os.Create(filenameo)
	check(err)
	defer ofile.Close()

	// Write each line token as a line in the output file
	w := bufio.NewWriter(ofile)
	var newline = ""
	for instrNum, t := range processedInstructions {
		// Omit newline if last line of file or if empty line
		if instrNum != len(processedInstructions)-1 {
			newline = "\n"
		}

		DEBUG := false
		var line string
		if DEBUG {
			line = fmt.Sprintf("%-3v %-16v %v%v", t.lineNum, t.stripped, t.translated, newline)
		} else {
			line = fmt.Sprintf("%v%v", t.translated, newline)
		}
		_, err = w.WriteString(line)
		check(err)
	}
	log.Println("Output to", filenameo)
	w.Flush()
}
