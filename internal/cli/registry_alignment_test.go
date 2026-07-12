package cli

import "testing"

func TestRegistryMatchesTopLevelDispatcher(t *testing.T) {
	dispatched := []string{
		"version", "run", "init-config", "validate-config", "test-connection", "status", "send",
		"boot-notification", "heartbeat", "authorize", "status-notification", "meter-values",
		"start-transaction", "stop-transaction", "data-transfer", "diagnostics-status",
		"firmware-status", "security-event", "log-status", "signed-firmware-status",
		"sign-certificate", "completions", "help",
	}
	registered := make(map[string]bool, len(cliRegistry.Commands))
	for _, command := range cliRegistry.Commands {
		registered[command.Name] = true
	}
	for _, name := range dispatched {
		if !registered[name] {
			t.Errorf("dispatcher command %q is not registered", name)
		}
	}
	for name := range registered {
		found := false
		for _, candidate := range dispatched {
			if candidate == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("registered command %q is not dispatched", name)
		}
	}
}
