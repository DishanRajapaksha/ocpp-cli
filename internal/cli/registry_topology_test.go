package cli

import "testing"

func TestRegistryHasOneSendCommand(t *testing.T) {
	count := 0
	for _, command := range cliRegistry.Commands {
		if command.Name == "send" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("send registry count = %d", count)
	}
}
