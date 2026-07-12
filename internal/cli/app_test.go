package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DishanRajapaksha/industrial-cli-kit/contracttest"
	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
)

func TestSharedCommandContract(t *testing.T) {
	contracttest.Baseline(t, func(args ...string) contracttest.Result {
		var out, errOut bytes.Buffer
		code := NewApp(&out, &errOut).Run(args)
		return contracttest.Result{Code: code, Stdout: out.String(), Stderr: errOut.String()}
	})
}

func TestHelpAndUnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	if code := app.Run(nil); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	if !strings.Contains(out.String(), "boot-notification") || !strings.Contains(out.String(), "data-transfer") {
		t.Fatalf("usage missing command: %s", out.String())
	}
	out.Reset()
	if code := app.Run([]string{"bogus"}); code != exitGeneralError {
		t.Fatalf("unknown command exit code = %d", code)
	}
}

func TestSnapshotRejectsJSONL(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	code := app.Run([]string{"validate-config", "--format", "jsonl"})
	if code != exitConfigError {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "stream-only") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestGlobalFlagsBeforeCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	code := app.Run([]string{"--format", "json", "validate-config"})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"valid": true`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestBootRejectedUsesExitSevenAfterRendering(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	app.newStation = func(config.ClientConfig) (ocppclient.Station, error) {
		return &fakeStation{bootResult: ocppclient.BootResult{Status: "Rejected", Interval: 30, CurrentTime: time.Unix(0, 0).UTC()}, bootErr: ocppclient.ErrRejected}, nil
	}
	code := app.Run([]string{"boot-notification", "--format", "json"})
	if code != exitRejected {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"status": "Rejected"`) {
		t.Fatalf("response was not rendered: %s", out.String())
	}
}

func TestBootProtocolErrorDoesNotRenderEmptyResponse(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	app.newStation = func(config.ClientConfig) (ocppclient.Station, error) {
		return &fakeStation{bootErr: ocppclient.ErrProtocol}, nil
	}
	code := app.Run([]string{"boot-notification", "--format", "json"})
	if code != exitProtocolRequest {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("protocol failure rendered a synthetic response: %s", out.String())
	}
}

func TestInvalidCommandArgumentsUseExitTwo(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	code := app.Run([]string{"status-notification", "--status", "Banana"})
	if code != exitConfigError {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
}

func TestConnectionErrorUsesExitThree(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	app.newStation = func(config.ClientConfig) (ocppclient.Station, error) {
		return &fakeStation{connectErr: ocppclient.ErrConnection}, nil
	}
	if code := app.Run([]string{"heartbeat"}); code != exitConnection {
		t.Fatalf("exit code = %d", code)
	}
}

func TestStatusAndSendSupportCanonicalForms(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	app.newStation = func(config.ClientConfig) (ocppclient.Station, error) {
		return &fakeStation{}, nil
	}
	if code := app.Run([]string{"status", "--format", "json"}); code != exitSuccess {
		t.Fatalf("status exit code = %d, stderr=%s", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := app.Run([]string{"send", "heartbeat", "--format", "json"}); code != exitSuccess {
		t.Fatalf("send heartbeat exit code = %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "current_time") {
		t.Fatalf("heartbeat output = %s", out.String())
	}
}

type fakeStation struct {
	connectErr         error
	bootResult         ocppclient.BootResult
	bootErr            error
	dataTransferResult ocppclient.DataTransferResult
	dataTransferErr    error
	signResult         ocppclient.SignCertificateResult
	signErr            error
}

func (f *fakeStation) Connect(context.Context) error { return f.connectErr }
func (f *fakeStation) Close()                        {}
func (f *fakeStation) Boot(context.Context, ocppclient.BootRequest) (ocppclient.BootResult, error) {
	return f.bootResult, f.bootErr
}
func (f *fakeStation) Heartbeat(context.Context) (ocppclient.HeartbeatResult, error) {
	return ocppclient.HeartbeatResult{CurrentTime: time.Unix(0, 0).UTC()}, nil
}
func (f *fakeStation) Authorize(context.Context, string) (ocppclient.AuthorizationResult, error) {
	return ocppclient.AuthorizationResult{Status: "Accepted"}, nil
}
func (f *fakeStation) StatusNotification(context.Context, ocppclient.StatusRequest) error {
	return nil
}
func (f *fakeStation) MeterValues(context.Context, ocppclient.MeterRequest) error { return nil }
func (f *fakeStation) StartTransaction(context.Context, ocppclient.StartTransactionRequest) (ocppclient.StartTransactionResult, error) {
	return ocppclient.StartTransactionResult{}, nil
}
func (f *fakeStation) StopTransaction(context.Context, ocppclient.StopTransactionRequest) (ocppclient.StopTransactionResult, error) {
	return ocppclient.StopTransactionResult{}, nil
}
func (f *fakeStation) DataTransfer(context.Context, ocppclient.DataTransferRequest) (ocppclient.DataTransferResult, error) {
	return f.dataTransferResult, f.dataTransferErr
}
func (f *fakeStation) DiagnosticsStatusNotification(context.Context, string) error { return nil }
func (f *fakeStation) FirmwareStatusNotification(context.Context, string) error    { return nil }
func (f *fakeStation) SecurityEventNotification(context.Context, ocppclient.SecurityEventRequest) error {
	return nil
}
func (f *fakeStation) LogStatusNotification(context.Context, ocppclient.LogStatusRequest) error {
	return nil
}
func (f *fakeStation) SignedFirmwareStatusNotification(context.Context, ocppclient.SignedFirmwareStatusRequest) error {
	return nil
}
func (f *fakeStation) SignCertificate(context.Context, ocppclient.SignCertificateRequest) (ocppclient.SignCertificateResult, error) {
	return f.signResult, f.signErr
}

var _ ocppclient.Station = (*fakeStation)(nil)
