package splitter

import (
	"testing"
)

func TestCalcLines(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		full      []byte
		startIdx  int
		endIdx    int
		wantStart int
		wantEnd   int
	}{
		{
			name:      "Single line, middle",
			full:      []byte("Hello world"),
			startIdx:  6,
			endIdx:    11,
			wantStart: 1,
			wantEnd:   1,
		},
		{
			name:      "Multi-line, span lines",
			full:      []byte("Line 1\nLine 2\nLine 3"),
			startIdx:  7,
			endIdx:    13,
			wantStart: 2,
			wantEnd:   2,
		},
		{
			name:      "Multi-line, span multiple lines",
			full:      []byte("Line 1\nLine 2\nLine 3\nLine 4"),
			startIdx:  7,
			endIdx:    20,
			wantStart: 2,
			wantEnd:   3,
		},
		{
			name:      "Start at beginning",
			full:      []byte("Line 1\nLine 2"),
			startIdx:  0,
			endIdx:    5,
			wantStart: 1,
			wantEnd:   1,
		},
		{
			name:      "End at end",
			full:      []byte("Line 1\nLine 2"),
			startIdx:  7,
			endIdx:    12,
			wantStart: 2,
			wantEnd:   2,
		},
		{
			name:      "Empty input",
			full:      []byte(""),
			startIdx:  0,
			endIdx:    0,
			wantStart: 1,
			wantEnd:   1,
		},
		{
			name:      "Start and end on newline",
			full:      []byte("Line1\nLine2\nLine3"),
			startIdx:  5,
			endIdx:    11,
			wantStart: 1, // Start is before the first \n
			wantEnd:   2, // End is before the second \n
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotStart, gotEnd := CalcLines(tt.full, tt.startIdx, tt.endIdx)
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Errorf("CalcLines() = (%v, %v), want (%v, %v)", gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestSanitize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "No changes",
			input: "Hello world",
			want:  "Hello world",
		},
		{
			name:  "Trim leading/trailing spaces",
			input: "  Hello world  ",
			want:  "Hello world",
		},
		{
			name:  "Trim leading/trailing newlines",
			input: "\nHello world\n",
			want:  "Hello world",
		},
		{
			name:  "Trim leading/trailing tabs",
			input: "\tHello world\t",
			want:  "Hello world",
		},
		{
			name:  "Trim mixed whitespace",
			input: " \n\t Hello world \t\n ",
			want:  "Hello world",
		},
		{
			name:  "Normalize CR/LF",
			input: "Line 1\r\nLine 2",
			want:  "Line 1\nLine 2",
		},
		{
			name:  "Normalize multiple CR/LF",
			input: "Line 1\r\n\r\nLine 2",
			want:  "Line 1\n\nLine 2",
		},
		{
			name:  "Trim and normalize",
			input: " \r\n Hello\r\nWorld \r\n ",
			want:  "Hello\nWorld",
		},
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
		{
			name:  "Whitespace only string",
			input: " \n\t \r\n ",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Sanitize(tt.input); got != tt.want {
				t.Errorf("Sanitize() = %q, want %q", got, tt.want)
			}
		})
	}
}
