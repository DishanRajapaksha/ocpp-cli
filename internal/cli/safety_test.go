package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
)

func TestStartTransactionIsDryRunByDefault(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	connected := false
	app.newStation = func(config.ClientConfig) (ocppclient.Station, error) {
		connected = true
		return &fakeStation{}, nil
	}

	code := app.Run([]string{"start-transaction", "--connector", "1", "--id-tag", "ABC123", "--meter-start", "100", "--format", "json"})
	if code != exitSuccess {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if connected {
		t.Fatal("dry-run opened an OCPP connection")
	}
	if !strings.Contains(out.String(), `"mode": "dry-run"`) || !strings.Contains(out.String(), `"action": "StartTransaction"`) {
		t.Fatalf("unexpected dry-run output: %s", out.String())
	}
}

func TestStartTransactionYesExecutes(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	connected := false
	app.newStation = func(config.ClientConfig) (ocppclient.Station, error) {
		connected = true
		return &fakeStation{}, nil
	}

	code := app.Run([]string{"send", "start-transaction", "--connector", "1", "--id-tag", "ABC123", "--meter-start", "100", "--yes", "--format", "json"})
	if code != exitSuccess {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if !connected {
		t.Fatal("--yes did not execute the OCPP operation")
	}
}

func TestStopTransactionRejectsConflictingSafetyFlags(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)

	code := app.Run([]string{"stop-transaction", "--transaction-id", "42", "--meter-stop", "200", "--yes", "--dry-run"})
	if code != exitConfigError {
		t.Fatalf("exit code = %d, want %d", code, exitConfigError)
	}
	if !strings.Contains(errOut.String(), "--yes and --dry-run cannot be used together") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}
