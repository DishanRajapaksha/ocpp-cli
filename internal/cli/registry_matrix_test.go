package cli

import "testing"

func TestRegistryCommandAndSubcommandNamesAreNonEmpty(t *testing.T) {
	for _, command := range cliRegistry.Commands {
		if command.Name == "" {
			t.Fatal("registry contains an empty command name")
		}
		for _, subcommand := range command.Subcommands {
			if subcommand.Name == "" {
				t.Fatalf("command %q contains an empty subcommand name", command.Name)
			}
		}
	}
}
