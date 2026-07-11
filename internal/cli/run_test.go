package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DishanRajapaksha/ocpp-cli/internal/config"
	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
)

func TestRunWritesJSONLStream(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	app.newSimulator = func(config.ClientConfig, ocppclient.SimulatorOptions) (ocppclient.Simulator, error) {
		connector := 1
		return newFakeSimulator(
			ocppclient.SimulatorEvent{Timestamp: time.Unix(0, 0).UTC(), Direction: "system", Action: "Connect", Status: "Connected"},
			ocppclient.SimulatorEvent{Timestamp: time.Unix(1, 0).UTC(), Direction: "outbound", Action: "StatusNotification", Status: "Available", ConnectorID: &connector},
		), nil
	}
	if code := app.Run([]string{"run", "--format", "jsonl"}); code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two JSONL records, got %d: %s", len(lines), out.String())
	}
	if !strings.Contains(lines[0], `"action":"Connect"`) || !strings.Contains(lines[1], `"connector_id":1`) {
		t.Fatalf("unexpected JSONL: %s", out.String())
	}
}

func TestRunCSVHeaderWrittenOnce(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	app.newSimulator = func(config.ClientConfig, ocppclient.SimulatorOptions) (ocppclient.Simulator, error) {
		return newFakeSimulator(
			ocppclient.SimulatorEvent{Timestamp: time.Unix(0, 0).UTC(), Direction: "system", Action: "Connect", Status: "Connected"},
			ocppclient.SimulatorEvent{Timestamp: time.Unix(1, 0).UTC(), Direction: "outbound", Action: "Heartbeat", Status: "Accepted"},
		), nil
	}
	if code := app.Run([]string{"run", "--format", "csv"}); code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if count := strings.Count(out.String(), "timestamp,direction,action,status,connector_id,detail"); count != 1 {
		t.Fatalf("CSV header count = %d: %s", count, out.String())
	}
}

func TestRunRejectsSnapshotFormat(t *testing.T) {
	var out, errOut bytes.Buffer
	app := NewApp(&out, &errOut)
	if code := app.Run([]string{"run", "--format", "table"}); code != exitConfigError {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "snapshot-only") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

type fakeSimulator struct {
	events []ocppclient.SimulatorEvent
	stream chan ocppclient.SimulatorEvent
}

func newFakeSimulator(events ...ocppclient.SimulatorEvent) *fakeSimulator {
	return &fakeSimulator{events: events, stream: make(chan ocppclient.SimulatorEvent, len(events))}
}

func (f *fakeSimulator) Run(context.Context) error {
	for _, event := range f.events {
		f.stream <- event
	}
	close(f.stream)
	return nil
}

func (f *fakeSimulator) Events() <-chan ocppclient.SimulatorEvent { return f.stream }

var _ ocppclient.Simulator = (*fakeSimulator)(nil)
