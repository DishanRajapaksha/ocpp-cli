package cli

import "testing"

func TestSharedExitCodeContract(t *testing.T) {
	got := []int{exitSuccess, exitGeneralError, exitConfigError, exitConnection, exitProtocolRequest, exitAuthSecurity, exitResourceMissing, exitRejected, exitTimeout, exitOutputError}
	for i, code := range got {
		if code != i {
			t.Fatalf("exit code at position %d = %d", i, code)
		}
	}
}

func TestSnapshotFormats(t *testing.T) {
	for _, format := range []string{"table", "text", "json", "csv"} {
		if _, err := validateSnapshotFormat(format); err != nil {
			t.Fatalf("format %q rejected: %v", format, err)
		}
	}
	if _, err := validateSnapshotFormat("jsonl"); err == nil {
		t.Fatal("jsonl should be rejected for snapshots")
	}
}
