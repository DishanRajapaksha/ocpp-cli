package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DishanRajapaksha/industrial-cli-kit/command"
	"github.com/DishanRajapaksha/industrial-cli-kit/completion"
)

func TestRegistryMatchesTopLevelDispatcher(t *testing.T) {
	dispatched := []string{
		"version", "run", "init-config", "validate-config", "test-connection", "status", "send",
		"boot-notification", "heartbeat", "authorize", "status-notification", "meter-values",
		"start-transaction", "stop-transaction", "data-transfer", "diagnostics-status",
		"firmware-status", "security-event", "log-status", "signed-firmware-status",
		"sign-certificate", "completions", "help",
	}
	registered := map[string]bool{}
	for _, registeredCommand := range cliRegistry.Commands {
		if registered[registeredCommand.Name] {
			t.Fatalf("duplicate registry command %q", registeredCommand.Name)
		}
		registered[registeredCommand.Name] = true
	}
	for _, name := range dispatched {
		if !registered[name] {
			t.Errorf("dispatcher command %q is not registered", name)
		}
	}
	for name := range registered {
		found := false
		for _, candidate := range dispatched {
			if candidate == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("registered command %q is not dispatched", name)
		}
	}
}

func TestRegistryIncludesNestedOperationsAndSafetyFlags(t *testing.T) {
	var send []commandSnapshot
	for _, registeredCommand := range cliRegistry.Commands {
		if registeredCommand.Name != "send" {
			continue
		}
		for _, subcommand := range registeredCommand.Subcommands {
			send = append(send, commandSnapshot{name: subcommand.Name, flags: subcommand.Flags})
		}
	}
	if len(send) != len(operationCommands) {
		t.Fatalf("send operations = %d, want %d", len(send), len(operationCommands))
	}
	for _, operation := range send {
		if operation.name != "start-transaction" && operation.name != "stop-transaction" {
			continue
		}
		seen := map[string]bool{}
		for _, flag := range operation.flags {
			if flag.Name == "yes" || flag.Name == "dry-run" {
				seen[flag.Name] = true
				if flag.TakesValue {
					t.Fatalf("%s %s unexpectedly takes a value", operation.name, flag.Name)
				}
			}
		}
		if !seen["yes"] || !seen["dry-run"] {
			t.Fatalf("%s registry missing safety flags", operation.name)
		}
	}
}

func TestGeneratedCompletionsContainNestedSafetyMetadata(t *testing.T) {
	var out bytes.Buffer
	if err := completion.Write(&out, "bash", cliRegistry); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"send:start-transaction", "--yes", "--dry-run"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("completion output missing %q", want)
		}
	}
}

type commandSnapshot struct {
	name  string
	flags []command.Flag
}
