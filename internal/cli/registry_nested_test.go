package cli

import "testing"

func TestSendSubcommandsMirrorOperationCommands(t *testing.T) {
	var send []string
	for _, command := range cliRegistry.Commands {
		if command.Name == "send" {
			for _, subcommand := range command.Subcommands {
				send = append(send, subcommand.Name)
			}
		}
	}
	if len(send) != len(operationCommands) {
		t.Fatalf("send has %d operations; registry defines %d", len(send), len(operationCommands))
	}
	for i, operation := range operationCommands {
		if send[i] != operation.Name {
			t.Fatalf("send operation %d = %q, want %q", i, send[i], operation.Name)
		}
	}
}
