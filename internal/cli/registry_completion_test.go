package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DishanRajapaksha/industrial-cli-kit/completion"
)

func TestZshCompletionsContainCommandSummaries(t *testing.T) {
	var out bytes.Buffer
	if err := completion.Write(&out, "zsh", cliRegistry); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"send:Send a named OCPP operation", "start-transaction:Send StartTransaction"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("zsh completions missing %q", want)
		}
	}
}
