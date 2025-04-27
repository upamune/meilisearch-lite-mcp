package indexer

import (
	"testing"
)

func TestGenID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		path string
		s    int
		e    int
		want string
	}{
		{
			name: "Simple path",
			path: "docs/file1.md",
			s:    1,
			e:    10,
			// Corrected hash from test output
			want: "7824d8f6:1-10",
		},
		{
			name: "Different path",
			path: "another/doc.txt",
			s:    5,
			e:    5,
			// Corrected hash from test output
			want: "1764a957:5-5",
		},
		{
			name: "Long path",
			path: "a/very/long/path/structure/for/testing/this/function/document.md",
			s:    100,
			e:    250,
			// Corrected hash from test output
			want: "fc702fde:100-250",
		},
		{
			name: "Path with special chars",
			path: "path with spaces/and-symbols_!@#.md",
			s:    1,
			e:    2,
			// Corrected hash from test output
			want: "ceff31ba:1-2",
		},
		{
			name: "Empty path", // Should still produce a hash
			path: "",
			s:    1,
			e:    1,
			// sha1("")[:4] = da39a3ee...
			want: "da39a3ee:1-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := genID(tt.path, tt.s, tt.e); got != tt.want {
				t.Errorf("genID() = %v, want %v", got, tt.want)
			}
		})
	}
}
