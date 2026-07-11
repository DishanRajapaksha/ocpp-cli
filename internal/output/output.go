package output

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

var ErrOutput = errors.New("output error")

const (
	FormatTable = "table"
	FormatText  = "text"
	FormatJSON  = "json"
	FormatJSONL = "jsonl"
	FormatCSV   = "csv"
)

func NormaliseFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", FormatTable:
		return FormatTable
	case FormatText:
		return FormatText
	case FormatJSON:
		return FormatJSON
	case FormatJSONL:
		return FormatJSONL
	case FormatCSV:
		return FormatCSV
	default:
		return value
	}
}

func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("%w: %v", ErrOutput, err)
	}
	return nil
}

func WriteTable(w io.Writer, headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
		return fmt.Errorf("%w: %v", ErrOutput, err)
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return fmt.Errorf("%w: %v", ErrOutput, err)
		}
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("%w: %v", ErrOutput, err)
	}
	return nil
}

func WriteText(w io.Writer, value string) error {
	if _, err := fmt.Fprintln(w, value); err != nil {
		return fmt.Errorf("%w: %v", ErrOutput, err)
	}
	return nil
}

func WriteCSV(w io.Writer, headers []string, rows [][]string) error {
	cw := csv.NewWriter(w)
	if len(headers) > 0 {
		if err := cw.Write(headers); err != nil {
			return fmt.Errorf("%w: %v", ErrOutput, err)
		}
	}
	if err := cw.WriteAll(rows); err != nil {
		return fmt.Errorf("%w: %v", ErrOutput, err)
	}
	if err := cw.Error(); err != nil {
		return fmt.Errorf("%w: %v", ErrOutput, err)
	}
	return nil
}
