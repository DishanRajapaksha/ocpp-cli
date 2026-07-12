package cli

import "testing"

func TestRegistryContainsSendOperations(t *testing.T) {
	for _, command := range cliRegistry.Commands {
		if command.Name == "send" && len(command.Subcommands) == 0 {
			t.Fatal("send command has no registered operations")
		}
	}
}
