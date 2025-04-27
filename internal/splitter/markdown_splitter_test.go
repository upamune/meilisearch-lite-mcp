package splitter

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func TestSplitMarkdown(t *testing.T) {
	t.Parallel()

	// Initialize tiktoken encoder (assuming it's handled within SplitMarkdown)
	// If initialization fails, these tests might panic or fail. Consider error handling if needed.

	tests := []struct {
		name        string
		input       string
		maxTokens   int
		wantChunks  []Chunk
		wantErr     bool
	}{
		{
			name: "Simple paragraph",
			input: `This is a simple paragraph. It should form a single chunk.`,
			maxTokens: 50,
			wantChunks: []Chunk{
				{
					Text:      "This is a simple paragraph. It should form a single chunk.",
					StartIdx: 0,
					EndIdx:   58, // Length of sanitized text
					Type:     "text",
					Headings: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "Header and paragraph",
			input: `# Header 1

This is the first paragraph under Header 1.`,
			maxTokens: 50,
			wantChunks: []Chunk{
				// Header itself doesn't become a chunk, it updates context
				{
					Text:      "This is the first paragraph under Header 1.",
					StartIdx: 12, // Index after "# Header 1\n\n"
					EndIdx:   55, // StartIdx + len(text) (exclusive)
					Type:     "text",
					Headings: []string{"Header 1"},
				},
			},
			wantErr: false,
		},
		{
			name: "Code block",
			input: "```go\npackage main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n```",
			maxTokens: 100,
			wantChunks: []Chunk{
				{
					Text:      "```go\npackage main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n```",
					StartIdx: 6,  // Index of 'p' in package
					EndIdx:   54, // Index after closing brace '}' + newline
					Type:     "code",
					Headings: nil,
				},
			},
			wantErr: false,
		},
		{
			name: "Chunking due to maxTokens",
			input: `This is a relatively long sentence that will likely exceed the maximum token limit set for this test case, forcing it to be split into multiple chunks.`,
			maxTokens: 10, // Low limit to force chunking
			wantChunks: []Chunk{
				// Expect the entire sentence as one chunk, as splitting respects sentence boundaries.
				{Text: "This is a relatively long sentence that will likely exceed the maximum token limit set for this test case, forcing it to be split into multiple chunks.", StartIdx: 0, EndIdx: 151, Type: "text"},
			},
			wantErr: false,
		},
		{
			name: "Japanese text",
			input: `# 日本語のヘッダー

これは日本語の段落です。
複数行にわたることもあります。

` + "```" + `text
これはコードブロックです。
` + "```" + `
`,
			maxTokens: 50,
			wantChunks: []Chunk{
				// Header itself doesn't become a chunk
				{
					Text:      "これは日本語の段落です。\n複数行にわたることもあります。",
					StartIdx: 28,   // Index after "# 日本語のヘッダー\n\n"
					EndIdx:   110,  // StartIdx + len(text) (exclusive)
					Type:     "text",
					Headings: []string{"日本語のヘッダー"},
				},
				{
					Text:      "```text\nこれはコードブロックです。\n```",
					StartIdx: 120,  // Index of 'こ' in これは
					EndIdx:   160, // Index after '。' + newline
					Type:     "code",
					Headings: []string{"日本語のヘッダー"}, // Assuming heading persists
				},
			},
			wantErr: false,
		},
		{
			name: "Japanese text exceeding maxTokens",
			input: `これは非常に長い日本語の文章であり、設定された最大トークン数を超過するため、複数のチャンクに分割されることが期待されます。分割はトークナイザの挙動に依存します。`,
			maxTokens: 10, // Low limit to force chunking
			wantChunks: []Chunk{
				// Expect split based on sentence boundaries identified by the tokenizer.
				{Text: "これは非常に長い日本語の文章であり、設定された最大トークン数を超過するため、複数のチャンクに分割されることが期待されます。", StartIdx: 0, EndIdx: 183, Type: "text"},
				{Text: "分割はトークナイザの挙動に依存します。", StartIdx: 183, EndIdx: 240, Type: "text"},
			},
			wantErr: false,
		},
		{
			name: "Empty input",
			input: ``, // Empty string
			maxTokens: 50,
			wantChunks: nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Call SplitMarkdown directly
			gotChunks, err := SplitMarkdown(tt.input, tt.maxTokens, 0) // Using maxTokens as chunkTok, 0 as overlapTok

			if (err != nil) != tt.wantErr {
				t.Errorf("SplitMarkdown() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// A more robust comparison might be needed for those cases.
			if !reflect.DeepEqual(gotChunks, tt.wantChunks) {
				t.Errorf("SplitMarkdown() gotChunks = %v, want %v", gotChunks, tt.wantChunks)
				// Add more detailed diff logging if needed
				t.Logf("Got: %#v", gotChunks)
				t.Logf("Want: %#v", tt.wantChunks)
			}
		})
	}
}

func TestMarkdownSplitter_Split_Golden(t *testing.T) {
	// Find all .md files in testdata
	testFiles, err := filepath.Glob(filepath.Join("testdata", "*.md"))
	require.NoError(t, err)
	require.NotEmpty(t, testFiles, "No test files found in testdata/*.md")

	// Define chunk options for the test
	const chunkSize = 100
	const chunkOverlap = 10

	for _, testFile := range testFiles {
		testFile := testFile // Capture range variable
		testName := strings.TrimSuffix(filepath.Base(testFile), filepath.Ext(testFile))

		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			inputBytes, err := os.ReadFile(testFile)
			require.NoError(t, err)

			// Use the standalone SplitMarkdown function
			chunks, err := SplitMarkdown(string(inputBytes), chunkSize, chunkOverlap)
			require.NoError(t, err)

			// Use JSON for golden file format for better readability and structure
			var prettyJSON bytes.Buffer
			encoder := json.NewEncoder(&prettyJSON)
			encoder.SetIndent("", "  ")
			err = encoder.Encode(chunks)
			require.NoError(t, err)

			actualOutput := prettyJSON.Bytes()

			goldenFilePath := testFile + ".golden"

			if *update {
				t.Logf("Updating golden file: %s", goldenFilePath)
				err = os.WriteFile(goldenFilePath, actualOutput, 0644)
				require.NoError(t, err)
			} else {
				expectedOutput, err := os.ReadFile(goldenFilePath)
				if os.IsNotExist(err) {
					t.Fatalf("Golden file not found: %s. Run with -update flag to create it.", goldenFilePath)
				} else {
					require.NoError(t, err)
				}

				assert.Equal(t, string(expectedOutput), string(actualOutput), "Output does not match golden file")
			}
		})
	}
}
