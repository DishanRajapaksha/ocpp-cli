package cli

import "testing"

func TestRegistryCommandsAreUnique(t *testing.T) {
	seen := map[string]struct{}{}
	for _, command := range cliRegistry.Commands {
		if _, ok := seen[command.Name]; ok {
			t.Fatalf("duplicate command %q", command.Name)
		}
		seen[command.Name] = struct{}{}
	}
}
