package main

import (
	"testing"
)

func TestParseSuccess(t *testing.T) {
	// Setup
	var tests = []struct {
		// input
		instruction string
		// expected
		operation string
		segment   string
		value     int
	}{
		{"push local 1", "push", "local", 1},
		{"push local 1200", "push", "local", 1200},
		{"push temp 1", "push", "temp", 1},
		{"push this 1", "push", "this", 1},
		{"push that 1", "push", "that", 1},
		{"push static 1", "push", "static", 1},
		{"push pointer 1", "push", "pointer", 1},
		{"push  pointer 1", "push", "pointer", 1}, // multispace separator is valid
		{"add", "add", "", 0},
	}

	for _, test := range tests {
		// Test
		line := NewInstruction(test.instruction)
		err := line.parse()

		// Assert
		assertOp := test.operation == line.operation
		assertSegment := test.segment == line.segment
		assertValue := test.value == line.value

		if err != nil {
			t.Fatalf(`parsing %v produced error "%v"`, test, err)
		}

		if !assertOp || !assertSegment || !assertValue {
			t.Fatalf(`parsed improperly "%v"`, test)
		}
	}
}

func TestParseFail(t *testing.T) {
	// Setup
	var tests = []string{
		"pop main",         // invalid number of args
		"invalid",          // invalid operation
		"pop invalid 0",    // invalid segment
		"pop local notnum", // invalid value
	}

	for _, instruction := range tests {
		// Test
		line := NewInstruction(instruction)
		err := line.parse()

		// Assert
		if err == nil {
			t.Fatalf(`Expected "%v" produce err`, instruction)
		}
	}
}

func TestFilterBlanks(t *testing.T) {
	// setup
	s := []string{"hello", "", "world", "", ""}
	expected_len := 2
	// test
	result := filterBlanks(s)
	// assert
	if len(result) != expected_len {
		t.Fatalf("Incorrect filtering. Wanted len %d, got %q", expected_len, result)
	}
}
