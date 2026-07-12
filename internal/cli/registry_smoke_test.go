package cli

import "testing"

func TestRegistryIsNotEmpty(t *testing.T) {
	if len(cliRegistry.Commands) == 0 {
		t.Fatal("command registry is empty")
	}
}
