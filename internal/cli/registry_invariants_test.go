package cli

import "testing"

func TestRegistryHasUniqueCommandNames(t *testing.T) {
	seen := map[string]bool{}
	for _, command := range cliRegistry.Commands {
		if seen[command.Name] {
			t.Fatalf("duplicate registry command %q", command.Name)
		}
		seen[command.Name] = true
	}
}
