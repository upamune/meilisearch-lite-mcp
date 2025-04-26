package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/upamune/meilisearch-hybrid-mcp/internal/indexer"
	"github.com/upamune/meilisearch-hybrid-mcp/internal/mcpserver"
)

type CLI struct {
	Build IndexCmd `cmd:"" help:"Build Meilisearch index from markdown docs"`
	Serve ServeCmd `cmd:"" help:"Run MCP server (stdio)"`
}

type IndexCmd struct {
	Dirs         []string `arg:"" required:"" help:"Directories to crawl (.md)"`
	ChunkSize    int      `default:"350" help:"Chunk token size"`
	ChunkOverlap int      `default:"50" help:"Chunk token overlap"`
	Concurrency  int      `default:"30" help:"Parallel workers"`
}

// Implement indexer.Config interface for IndexCmd
func (c IndexCmd) GetDirs() []string    { return c.Dirs }
func (c IndexCmd) GetChunkSize() int    { return c.ChunkSize }
func (c IndexCmd) GetChunkOverlap() int { return c.ChunkOverlap }
func (c IndexCmd) GetConcurrency() int  { return c.Concurrency }

type ServeCmd struct {
	HttpAddr      string  `env:"MEILI_HTTP_ADDR" default:"http://localhost:7700"`
	SemanticRatio float64 `env:"HYBRID_SEARCH_SEMANTIC_RATIO" default:"0.5"`
	PathPrefix    string  `env:"SEARCH_PATH_PREFIX" default:""`
}

// serveAdapter wraps main.ServeCmd to satisfy mcpserver.ServeCmd interface
type serveAdapter struct{ cfg ServeCmd }

func (s serveAdapter) PathPrefix() string     { return s.cfg.PathPrefix }
func (s serveAdapter) HttpAddr() string       { return s.cfg.HttpAddr }
func (s serveAdapter) SemanticRatio() float64 { return s.cfg.SemanticRatio }

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var cli CLI
	k := kong.Parse(&cli,
		kong.Name("meilisearch-hymcp"),
		kong.Description("Meilisearch Hybrid Search MCP"),
		kong.UsageOnError(),
		kong.Bind(ctx),
	)
	switch k.Command() {
	case "build <dirs>":
		indexer.Run(ctx, cli.Build)
	case "serve":
		mcpserver.Run(ctx, serveAdapter{cli.Serve})
	}
}
