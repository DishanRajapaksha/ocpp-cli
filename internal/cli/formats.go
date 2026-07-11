package cli

import (
	"fmt"

	"github.com/DishanRajapaksha/ocpp-cli/internal/output"
)

func validateSnapshotFormat(value string) (string, error) {
	format := output.NormaliseFormat(value)
	switch format {
	case output.FormatTable, output.FormatText, output.FormatJSON, output.FormatCSV:
		return format, nil
	case output.FormatJSONL:
		return "", fmt.Errorf("format %q is stream-only; snapshots support table, text, json, and csv", value)
	default:
		return "", fmt.Errorf("unsupported snapshot format %q; use table, text, json, or csv", value)
	}
}
