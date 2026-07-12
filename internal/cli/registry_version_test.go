package cli

import "testing"

func TestRegistryBinaryName(t *testing.T) {
	if cliRegistry.Binary != "ocpp-cli" {
		t.Fatalf("registry binary = %q", cliRegistry.Binary)
	}
}
