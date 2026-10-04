package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionOutput(t *testing.T) {
	origVersion := version
	defer func() { version = origVersion }()

	testVersion := "1.2.3-test"
	version = testVersion

	cases := []struct {
		name string
		flag string
	}{
		{"subcommand version", "version"},
		{"short flag -v", "-v"},
		{"long flag --version", "--version"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			handled, err := handleCLICommand([]string{tc.flag}, buf)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !handled {
				t.Fatalf("expected command %q to be handled", tc.flag)
			}
			output := buf.String()
			if !strings.Contains(output, testVersion) {
				t.Errorf("expected output to contain version %q, got: %s", testVersion, output)
			}
			if !strings.Contains(output, "Apirout version") {
				t.Errorf("expected output to contain 'Apirout version', got: %s", output)
			}
		})
	}
}

func TestDefaultVersionFallback(t *testing.T) {
	if version == "" {
		t.Errorf("expected default version to be non-empty")
	}
}
