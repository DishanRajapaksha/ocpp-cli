package cli

import "testing"

func TestRegistryRequiredCommandsArePresent(t *testing.T) {
	required := []string{"run", "send", "status", "completions"}
	for _, name := range required {
		found := false
		for _, command := range cliRegistry.Commands {
			if command.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required command %q is missing", name)
		}
	}
}
