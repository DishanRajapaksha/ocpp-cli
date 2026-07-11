package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, []string{"A", "B"}, [][]string{{"1", "2"}}); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "A,B\n1,2\n" {
		t.Fatalf("unexpected CSV: %q", got)
	}
}

func TestWriteJSONWrapsWriterFailure(t *testing.T) {
	err := WriteJSON(failingWriter{}, map[string]bool{"ok": true})
	if err == nil || !strings.Contains(err.Error(), ErrOutput.Error()) {
		t.Fatalf("expected output error, got %v", err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, bytes.ErrTooLarge }
