package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRegistryCompletionsIncludeNestedOperationsAndSafetyFlags(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	if code := app.Run([]string{"completions", "bash"}); code != exitSuccess {
		t.Fatalf("bash completions exit code = %d, stderr=%s", code, errOut.String())
	}
	for _, want := range []string{"send:start-transaction", "--yes", "--dry-run"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("bash completions missing %q", want)
		}
	}
}
