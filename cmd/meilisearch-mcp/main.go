package main

import (
	"github.com/alecthomas/kong"

	"github.com/upamune/meilisearch-lite-mcp/internal/indexer"
	"github.com/upamune/meilisearch-lite-mcp/internal/mcpserver"
)

type CLI struct {
	BuildIndex indexer.BuildIndexCmd `cmd:"" name:"build-index" help:"Build index"`
	Serve      mcpserver.ServeCmd    `cmd:"" name:"serve" help:"Run MCP server (stdio)"`
}

func main() {
	var cli CLI
	kongCtx := kong.Parse(&cli,
		kong.Name("meilisearch-mcp"),
		kong.Description("Custom MCP server with Meilisearch hybrid search"),
		kong.UsageOnError(),
	)
	err := kongCtx.Run()
	kongCtx.FatalIfErrorf(err)
}
