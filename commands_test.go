// Unit Tests for the Command Processor
package main

import (
	"reflect"
	"testing"
)

func TestValidateCommand(t *testing.T) {
	// Define a table of test cases
	tests := []struct {
		name           string
		command        string
		expectedTokens []string
		expectedErr    string
	}{
		{"empty command", "", nil, "No command entered."},
		{"invalid command", "fly me to the moon", nil, "I don't understand your command."},
		{"valid command", "go north", []string{"go", "north"}, ""},
		{"extra spaces", "  look  ", []string{"look"}, ""},
		{"mixed case", "GeT item", []string{"GeT", "item"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := validateCommand(tt.command, valid_commands)

			// Check if error matches expected error
			if err != nil && err.Error() != tt.expectedErr {
				t.Errorf("validateCommand(%q) error = %v, wantErr %v", tt.command, err, tt.expectedErr)
			}

			// Check if no error was expected but an error was received
			if err == nil && tt.expectedErr != "" {
				t.Errorf("validateCommand(%q) expected error but got none", tt.command)
			}

			// Check if the tokens match the expected tokens
			if !reflect.DeepEqual(tokens, tt.expectedTokens) {
				t.Errorf("validateCommand(%q) tokens = %v, want %v", tt.command, tokens, tt.expectedTokens)
			}
		})
	}
}
