package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/DishanRajapaksha/ocpp-cli/internal/ocppclient"
	"github.com/DishanRajapaksha/ocpp-cli/internal/output"
)

type simulatorStreamWriter struct {
	out       io.Writer
	format    string
	json      *json.Encoder
	csv       *csv.Writer
	csvHeader bool
}

func newSimulatorStreamWriter(out io.Writer, format string) *simulatorStreamWriter {
	writer := &simulatorStreamWriter{out: out, format: format}
	if format == output.FormatJSONL {
		writer.json = json.NewEncoder(out)
	}
	if format == output.FormatCSV {
		writer.csv = csv.NewWriter(out)
	}
	return writer
}

func (w *simulatorStreamWriter) Write(event ocppclient.SimulatorEvent) error {
	switch w.format {
	case output.FormatText:
		connector := ""
		if event.ConnectorID != nil {
			connector = fmt.Sprintf(" connector=%d", *event.ConnectorID)
		}
		detail := ""
		if event.Detail != "" {
			detail = " detail=" + strconv.Quote(event.Detail)
		}
		if _, err := fmt.Fprintf(w.out, "%s direction=%s action=%s status=%s%s%s\n", event.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z07:00"), event.Direction, event.Action, event.Status, connector, detail); err != nil {
			return fmt.Errorf("%w: %v", output.ErrOutput, err)
		}
		return nil
	case output.FormatJSONL:
		if err := w.json.Encode(event); err != nil {
			return fmt.Errorf("%w: %v", output.ErrOutput, err)
		}
		return nil
	case output.FormatCSV:
		if !w.csvHeader {
			if err := w.csv.Write([]string{"timestamp", "direction", "action", "status", "connector_id", "detail"}); err != nil {
				return fmt.Errorf("%w: %v", output.ErrOutput, err)
			}
			w.csvHeader = true
		}
		connector := ""
		if event.ConnectorID != nil {
			connector = strconv.Itoa(*event.ConnectorID)
		}
		if err := w.csv.Write([]string{event.Timestamp.UTC().Format("2006-01-02T15:04:05.000Z07:00"), event.Direction, event.Action, event.Status, connector, event.Detail}); err != nil {
			return fmt.Errorf("%w: %v", output.ErrOutput, err)
		}
		w.csv.Flush()
		if err := w.csv.Error(); err != nil {
			return fmt.Errorf("%w: %v", output.ErrOutput, err)
		}
		return nil
	default:
		return fmt.Errorf("%w: unsupported stream format %q", output.ErrOutput, w.format)
	}
}
