package indexer

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	meilisearch "github.com/meilisearch/meilisearch-go"
)

// BuildIndexCmd indexes Markdown files into Meilisearch.
type BuildIndexCmd struct {
	Dir              []string `arg:"" name:"dir" help:"Directory paths to index Markdown files"`
	HTTPAddr         string   `env:"MEILI_HTTP_ADDR" default:"http://localhost:7700" help:"Meilisearch HTTP address"`
	MasterKey        string   `env:"MEILI_MASTER_KEY" help:"Meilisearch master key (optional)"`
	Source           string   `env:"EMBEDDERS_SOURCE,required" help:"Embedders source"`
	Model            string   `env:"EMBEDDERS_MODEL,required" help:"Embedders model"`
	APIKey           string   `env:"EMBEDDERS_API_KEY,required" help:"Embedders API key"`
	DocumentTemplate string   `env:"EMBEDDERS_DOCUMENT_TEMPLATE,required" help:"Embedders document template"`
	Concurrency      int      `env:"CONCURRENCY" default:"5" help:"Number of parallel workers for indexing"`
	PathPrefix       string   `env:"DOCUMENT_PATH_PREFIX" default:"" help:"Prefix to add to document paths"`
}

// Run executes the indexing command.
func (cmd *BuildIndexCmd) Run(ctx context.Context) error {
	client := meilisearch.New(cmd.HTTPAddr, meilisearch.WithAPIKey(cmd.MasterKey))
	prefix := cmd.PathPrefix
	for _, dir := range cmd.Dir {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) == ".md" {
				// apply prefix if set
				fullPath := path
				if prefix != "" {
					fullPath = filepath.Join(prefix, path)
				}
				fmt.Printf("Indexing %s\n", fullPath)
				// TODO: read file, generate embedding, index to Meilisearch
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error walking directory %s: %w", dir, err)
		}
	}
	// ensure client is used in future implementation
	_ = client
	return nil
}
