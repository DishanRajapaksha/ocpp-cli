package cli

import "testing"

func TestRegistryGlobalFlagsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, flag := range cliRegistry.GlobalFlags {
		if seen[flag.Name] {
			t.Fatalf("duplicate global flag %q", flag.Name)
		}
		seen[flag.Name] = true
	}
}
