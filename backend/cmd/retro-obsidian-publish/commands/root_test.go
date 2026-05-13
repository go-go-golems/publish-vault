package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelpIncludesPrimaryVerbs(t *testing.T) {
	cmd, err := NewRootCommand()
	if err != nil {
		t.Fatalf("NewRootCommand() error = %v", err)
	}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help Execute() error = %v", err)
	}
	text := out.String()
	for _, want := range []string{"serve", "build"} {
		if !strings.Contains(text, want) {
			t.Fatalf("help output missing %q:\n%s", want, text)
		}
	}
}
