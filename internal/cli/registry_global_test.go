package cli

import "testing"

func TestRegistryContainsRequiredLifecycleCommands(t *testing.T) {
	required := map[string]bool{"init-config": false, "validate-config": false, "test-connection": false, "status": false, "completions": false, "help": false, "version": false}
	for _, command := range cliRegistry.Commands {
		if _, ok := required[command.Name]; ok {
			required[command.Name] = true
		}
	}
	for name, found := range required {
		if !found {
			t.Errorf("registry missing lifecycle command %q", name)
		}
	}
}
