package indexer

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/sourcegraph/conc/pool"

	"github.com/upamune/meilisearch-hybrid-mcp/internal/splitter"
	"github.com/upamune/meilisearch-hybrid-mcp/meilisearchutil"
)

var targetFileExtensions = map[string]bool{
	".md":  true,
	".mdx": true,
}

type RunParam struct {
	HttpAddr     string
	ApiKey       string
	ChunkSize    int
	ChunkOverlap int
	Concurrency  int
	Dirs         []string
}

func Run(
	ctx context.Context,
	param RunParam,
) error {
	client := meilisearchutil.NewClient(param.HttpAddr, param.ApiKey)
	index, err := meilisearchutil.CreateDocumentIndex(
		ctx,
		client,
		meilisearch.Embedder{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create index: %v\n", err)
		os.Exit(1)
	}

	var targetFilePaths []string
	for _, root := range param.Dirs {
		if err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && targetFileExtensions[filepath.Ext(p)] {
				targetFilePaths = append(targetFilePaths, p)
			}
			return nil
		}); err != nil {
			return err
		}
	}

	chunkTok, overlapTok := param.ChunkSize, param.ChunkOverlap
	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	p := pool.New().WithContext(ctx).WithMaxGoroutines(param.Concurrency)
	for _, filePath := range targetFilePaths {
		p.Go(func(ctx context.Context) error {
			relativePath, err := filepath.Rel(baseDir, filePath)
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(relativePath)
			if err != nil {
				return err
			}

			var batches []any
			chunks, _ := splitter.SplitMarkdown(string(raw), chunkTok, overlapTok)
			for _, c := range chunks {
				s, e := splitter.CalcLines(raw, c.StartIdx, c.EndIdx)
				id := genID(relativePath, s, e)
				batches = append(batches, map[string]any{
					"id":         id,
					"path":       relativePath,
					"start_line": s,
					"end_line":   e,
					"headings":   c.Headings,
					"text":       c.Text,
					"kind":       c.Type,
				})
			}

			task, err := index.GetDocumentManager().AddDocuments(batches, meilisearchutil.DocsPrimaryKey)
			if err != nil {
				return err
			}

			if _, err := client.WaitForTaskWithContext(ctx, task.TaskUID, 5*time.Second); err != nil {
				return err
			}

			return nil
		})
	}

	if err := p.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to index: %v\n", err)
		os.Exit(1)
	}

	return nil
}

func genID(p string, s, e int) string {
	h := sha1.Sum([]byte(p))
	return fmt.Sprintf("%s:%d-%d", hex.EncodeToString(h[:4]), s, e)
}
