package splitter

import (
	"bytes"
	"strings"
)

// CalcLines converts rune offset to (start,end) line numbers.
func CalcLines(full []byte, startIdx, endIdx int) (int, int) {
	start := bytes.Count(full[:startIdx], []byte{'\n'}) + 1
	end := start + bytes.Count(full[startIdx:endIdx], []byte{'\n'})
	return start, end
}

// Sanitize trims whitespace & normalizes newlines.
func Sanitize(s string) string {
	return strings.Trim(strings.ReplaceAll(s, "\r\n", "\n"), "\n\t ")
}
