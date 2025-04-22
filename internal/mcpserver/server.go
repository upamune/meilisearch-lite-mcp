package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/server"
)

// ServeCmd handles the MCP server.
type ServeCmd struct {
	HTTPAddr      string  `env:"MEILI_HTTP_ADDR" default:"http://localhost:7700" help:"Meilisearch HTTP address"`
	MasterKey     string  `env:"MEILI_MASTER_KEY" help:"Meilisearch master key (optional)"`
	SemanticRatio float64 `env:"HYBRID_SEARCH_SEMANTIC_RATIO" default:"0.5" help:"Semantic ratio (0.0-1.0)" validate:"min=0.0,max=1.0"`
}

// Run starts the MCP stdio server.
func (cmd *ServeCmd) Run(ctx context.Context) error {
	s := server.NewMCPServer(
		"Meilisearch Hybrid Search",
		"0.1.0",
	)

	// TODO: define tools for hybrid search

	// Start stdio server
	err := server.ServeStdio(s)
	if err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
