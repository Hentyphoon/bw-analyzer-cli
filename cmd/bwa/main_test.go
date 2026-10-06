package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunDispatch(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{"no args", nil, exitUsage, "", "Usage: bwa"},
		{"help", []string{"help"}, exitOK, "Usage: bwa", ""},
		{"dash h", []string{"-h"}, exitOK, "Usage: bwa", ""},
		{"unknown", []string{"frobnicate"}, exitUsage, "", `unknown command "frobnicate"`},
		{"known stub", []string{"serve"}, exitFatal, "", "bwa serve: not implemented yet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout = %q, want it to contain %q", stdout.String(), tt.wantStdout)
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestUsageListsEveryCommand(t *testing.T) {
	var buf bytes.Buffer
	usage(&buf)
	for _, c := range commands {
		if !strings.Contains(buf.String(), c.name) {
			t.Errorf("usage does not mention %q", c.name)
		}
	}
}
