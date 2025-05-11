package assert

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

var writer io.Writer

func SetOutput(w io.Writer) {
	writer = w
}

// TODO: Think about passing around a context for debugging purposes
func Assert(truth bool, msg string, data ...any) {
	if truth {
		return
	}

	doExit := false
	if writer == nil {
		writer = os.Stderr
		doExit = true
	}

	slog.Error("msg='this is a message'", "key", "value")
	fmt.Fprintf(writer, "%s: ", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "error: runtime assert failure: %s: ", msg)

	for _, item := range data {
		fmt.Fprintf(writer, "%v ", stringify(item))
	}

	fmt.Fprintln(writer, "")

	if doExit {
		os.Exit(1)
	}
}

// Stringify converts various data types into a string representation.
func stringify(item any) string {
	if item == nil {
		return "nil"
	}

	switch t := item.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", t)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", t)
	case float32, float64:
		return fmt.Sprintf("%g", t) // Use %g for cleaner float representation
	case bool:
		return fmt.Sprintf("%t", t)
	default:
		d, err := json.Marshal(item)
		if err != nil {
			return fmt.Sprintf("error: error='%v', input='%v'", err, item)
		}
		return string(d)
	}
}
