package cli

import "testing"

func TestRegistryIncludesCanonicalAndLegacyOperations(t *testing.T) {
	wantTopLevel := map[string]bool{
		"status": false,
		"send": false,
		"start-transaction": false,
		"stop-transaction": false,
	}
	for _, registered := range cliRegistry.Commands {
		if _, ok := wantTopLevel[registered.Name]; ok {
			wantTopLevel[registered.Name] = true
		}
	}
	for name, found := range wantTopLevel {
		if !found {
			t.Fatalf("registry missing top-level command %q", name)
		}
	}

	var sendFound bool
	for _, registered := range cliRegistry.Commands {
		if registered.Name != "send" {
			continue
		}
		sendFound = true
		wantOperations := map[string]bool{"heartbeat": false, "start-transaction": false, "stop-transaction": false}
		for _, operation := range registered.Subcommands {
			if _, ok := wantOperations[operation.Name]; ok {
				wantOperations[operation.Name] = true
			}
		}
		for name, found := range wantOperations {
			if !found {
				t.Fatalf("send registry missing operation %q", name)
			}
		}
	}
	if !sendFound {
		t.Fatal("registry missing send command")
	}
}

func TestRegistryMarksSafetyFlagsAsBoolean(t *testing.T) {
	for _, registered := range operationCommands {
		if registered.Name != "start-transaction" && registered.Name != "stop-transaction" {
			continue
		}
		seen := map[string]bool{}
		for _, flag := range registered.Flags {
			if flag.Name == "yes" || flag.Name == "dry-run" {
				seen[flag.Name] = true
				if flag.TakesValue {
					t.Fatalf("%s %s unexpectedly takes a value", registered.Name, flag.Name)
				}
			}
		}
		if !seen["yes"] || !seen["dry-run"] {
			t.Fatalf("%s registry missing safety flags", registered.Name)
		}
	}
}
