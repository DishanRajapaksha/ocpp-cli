package cli

import "testing"

func TestRegistryIsPopulated(t *testing.T) {
	if len(cliRegistry.Commands) == 0 {
		t.Fatal("command registry is empty")
	}
}
