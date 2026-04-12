package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
)

var (
	Green  = color.New(color.FgGreen).SprintFunc()
	Red    = color.New(color.FgRed).SprintFunc()
	Yellow = color.New(color.FgYellow).SprintFunc()
	Cyan   = color.New(color.FgCyan).SprintFunc()
	Bold   = color.New(color.Bold).SprintFunc()
	Dim    = color.New(color.Faint).SprintFunc()
)

func StatusIcon(status string) string {
	switch status {
	case "ready":
		return Green("●")
	case "building", "queued":
		return Yellow("●")
	case "failed":
		return Red("●")
	default:
		return Dim("●")
	}
}

func PrintJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func NewTable(headers ...string) *Table {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	t := &Table{w: w}
	if len(headers) > 0 {
		t.Row(headers...)
	}
	return t
}

type Table struct {
	w *tabwriter.Writer
}

func (t *Table) Row(cols ...string) {
	for i, col := range cols {
		if i > 0 {
			fmt.Fprint(t.w, "\t")
		}
		fmt.Fprint(t.w, col)
	}
	fmt.Fprintln(t.w)
}

func (t *Table) Flush() {
	t.w.Flush()
}

func Success(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, Green("✓ ")+msg+"\n", args...)
}

func Error(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, Red("✗ ")+msg+"\n", args...)
}

func Info(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, Cyan("ℹ ")+msg+"\n", args...)
}

func Warn(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, Yellow("⚠ ")+msg+"\n", args...)
}

// Prompt reads a line from the reader (default stdin).
func Prompt(prompt string, reader io.Reader) string {
	if reader == nil {
		reader = os.Stdin
	}
	fmt.Fprint(os.Stderr, prompt)
	var s string
	fmt.Fscanln(reader, &s)
	return s
}
