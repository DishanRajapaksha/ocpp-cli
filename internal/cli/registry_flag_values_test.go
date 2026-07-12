package cli

import "testing"

func TestRegistryValueFlagsAreMarked(t *testing.T) {
	for _, flag := range cliRegistry.GlobalFlags {
		if flag.Name == "config" && !flag.TakesValue {
			t.Fatal("config flag must take a value")
		}
	}
}
