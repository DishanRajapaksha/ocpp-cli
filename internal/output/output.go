package output

import (
	shared "github.com/DishanRajapaksha/industrial-cli-kit/output"
	"io"
)

var ErrOutput = shared.ErrOutput

const (
	FormatTable = shared.FormatTable
	FormatText  = shared.FormatText
	FormatJSON  = shared.FormatJSON
	FormatJSONL = shared.FormatJSONL
	FormatCSV   = shared.FormatCSV
)

func NormaliseFormat(value string) string    { return shared.NormaliseFormat(value) }
func WriteJSON(w io.Writer, value any) error { return shared.WriteJSON(w, value) }
func WriteTable(w io.Writer, headers []string, rows [][]string) error {
	return shared.WriteTable(w, headers, rows)
}
func WriteText(w io.Writer, value string) error { return shared.WriteText(w, value) }
func WriteCSV(w io.Writer, headers []string, rows [][]string) error {
	return shared.WriteCSV(w, headers, rows)
}
