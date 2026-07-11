package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
)

func TestDataTransferRejectedRendersThenExitsSeven(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	app.newStation = func(config.ClientConfig) (ocppclient.Station, error) {
		return &fakeStation{
			dataTransferResult: ocppclient.DataTransferResult{Status: "UnknownVendorId", Data: map[string]any{"reason": "missing"}},
			dataTransferErr:    ocppclient.ErrRejected,
		}, nil
	}
	code := app.Run([]string{"data-transfer", "--vendor-id", "example.org", "--data", `{"hello":"world"}`, "--format", "json"})
	if code != exitRejected {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"status": "UnknownVendorId"`) {
		t.Fatalf("rejection response was not rendered: %s", out.String())
	}
}

func TestDataTransferRejectsInvalidJSON(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	code := app.Run([]string{"data-transfer", "--vendor-id", "example.org", "--data", `{bad`})
	if code != exitConfigError {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "valid JSON") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestNotificationEnumValidation(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	if code := app.Run([]string{"firmware-status", "--status", "Exploded"}); code != exitConfigError {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(errOut.String(), "invalid firmware status") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestCompletionsContainExtendedCommands(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	if code := app.Run([]string{"completions", "bash"}); code != exitSuccess {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	for _, command := range []string{"data-transfer", "security-event", "sign-certificate"} {
		if !strings.Contains(out.String(), command) {
			t.Fatalf("completion output missing %q", command)
		}
	}
}
