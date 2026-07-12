package cli

import "testing"

func TestRegistryBinary(t *testing.T) {
	if cliRegistry.Binary != "ocpp-cli" {
		t.Fatalf("registry binary = %q", cliRegistry.Binary)
	}
}
