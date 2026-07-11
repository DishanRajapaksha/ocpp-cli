package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/DishanRajapaksha/ocpp-cli/internal/output"
)

type snapshot struct {
	headers []string
	rows    [][]string
	text    string
	value   any
}

func renderSnapshot(w io.Writer, format string, result snapshot) error {
	switch format {
	case output.FormatTable:
		return output.WriteTable(w, result.headers, result.rows)
	case output.FormatText:
		return output.WriteText(w, result.text)
	case output.FormatJSON:
		return output.WriteJSON(w, result.value)
	case output.FormatCSV:
		return output.WriteCSV(w, result.headers, result.rows)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func keyValueSnapshot(value any, fields ...[2]string) snapshot {
	rows := make([][]string, 0, len(fields))
	text := make([]string, 0, len(fields))
	for _, field := range fields {
		rows = append(rows, []string{field[0], field[1]})
		text = append(text, fmt.Sprintf("%s: %s", field[0], field[1]))
	}
	return snapshot{
		headers: []string{"FIELD", "VALUE"},
		rows:    rows,
		text:    strings.Join(text, "\n"),
		value:   value,
	}
}
