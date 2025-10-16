package logging

import (
	"testing"
)

func TestConstants(t *testing.T) {
	if FieldFile == "" {
		t.Error("FieldFile constant should not be empty")
	}
	if FieldCount == "" {
		t.Error("FieldCount constant should not be empty")
	}
	if FieldDelimiter == "" {
		t.Error("FieldDelimiter constant should not be empty")
	}
	if FieldInputFile == "" {
		t.Error("FieldInputFile constant should not be empty")
	}
	if FieldOutputFile == "" {
		t.Error("FieldOutputFile constant should not be empty")
	}
}
