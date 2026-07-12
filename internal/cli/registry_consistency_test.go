package cli

import "testing"

func TestOperationRegistryNamesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, operation := range operationCommands {
		if seen[operation.Name] {
			t.Fatalf("duplicate operation %q", operation.Name)
		}
		seen[operation.Name] = true
	}
}
