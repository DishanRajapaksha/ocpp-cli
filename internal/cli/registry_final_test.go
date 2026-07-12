package cli

import "testing"

func TestRegistryHasLifecycleCommands(t *testing.T) {
	required := map[string]bool{"status": false, "completions": false, "help": false, "version": false}
	for _, command := range cliRegistry.Commands {
		if _, ok := required[command.Name]; ok {
			required[command.Name] = true
		}
	}
	for name, found := range required {
		if !found {
			t.Errorf("missing lifecycle command %q", name)
		}
	}
}
