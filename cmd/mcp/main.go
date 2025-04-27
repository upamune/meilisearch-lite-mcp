package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/meilisearch/meilisearch-go"
	"github.com/upamune/meilisearch-hybrid-mcp/internal/indexer"
	"github.com/upamune/meilisearch-hybrid-mcp/internal/mcpserver"
	"github.com/upamune/meilisearch-hybrid-mcp/meilisearchutil"
)

type CLI struct {
	// Global flags, accessible by all commands
	HttpAddr      string  `env:"MEILI_HTTP_ADDR" default:"http://localhost:7700" help:"Meilisearch server address."`
	ApiKey        string  `env:"MEILI_MASTER_KEY" help:"Meilisearch API key."`
	SemanticRatio float64 `env:"HYBRID_SEARCH_SEMANTIC_RATIO" default:"0.5" help:"Semantic ratio for hybrid search (0.0 to 1.0)."`

	Build  IndexCmd  `cmd:"" help:"Build Meilisearch index from markdown docs."`
	Serve  ServeCmd  `cmd:"" help:"Run MCP server (stdio)."`
	Search SearchCmd `cmd:"" help:"Perform hybrid search directly."`
}

type IndexCmd struct {
	Dirs         []string `arg:"" required:"" help:"Directories to crawl (.md)."`
	ChunkSize    int      `default:"350" help:"Chunk token size."`
	ChunkOverlap int      `default:"50" help:"Chunk token overlap."`
	Concurrency  int      `default:"30" help:"Parallel workers."`
}

func (c *IndexCmd) Run(ctx context.Context, cli *CLI) error {
	return indexer.Run(ctx, indexer.RunParam{
		HttpAddr:     cli.HttpAddr,
		ApiKey:       cli.ApiKey,
		ChunkSize:    c.ChunkSize,
		ChunkOverlap: c.ChunkOverlap,
		Concurrency:  c.Concurrency,
		Dirs:         c.Dirs,
	})
}

type ServeCmd struct {
	PathPrefix string `env:"SEARCH_PATH_PREFIX" default:"" help:"Prefix for search paths (optional)."`
}

func (c *ServeCmd) Run(ctx context.Context, cli *CLI) error {
	mcpserver.Run(ctx, mcpserver.RunParam{
		HttpAddr:      cli.HttpAddr,
		ApiKey:        cli.ApiKey,
		SemanticRatio: cli.SemanticRatio,
		PathPrefix:    c.PathPrefix,
	})

	return nil
}

type SearchCmd struct {
	Query       string `arg:"" required:"" help:"Search query."`
	Interactive bool   `short:"i" help:"Enable interactive mode."`
}

func (c *SearchCmd) Run(ctx context.Context, cli *CLI) error {
	client := meilisearchutil.NewClient(cli.HttpAddr, cli.ApiKey)

	// Perform initial search if a query is provided via flag
	if c.Query != "" {
		if err := performSearch(ctx, client, c.Query, cli.SemanticRatio); err != nil {
			fmt.Fprintf(os.Stderr, "Error during initial search: %v\n", err)
			// Decide if we should exit or continue to interactive mode
			// For now, let's continue to interactive mode even if the initial query failed
		}
	}

	// Start interactive search loop
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\nEnter query (or type 'exit' to quit):")
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			} else {
				// EOF (e.g., Ctrl+D)
				fmt.Println("\nExiting.")
			}
			break
		}

		query := scanner.Text()
		if strings.ToLower(strings.TrimSpace(query)) == "exit" {
			fmt.Println("Exiting.")
			break
		}

		if query == "" {
			continue // Skip empty input
		}

		if err := performSearch(ctx, client, query, cli.SemanticRatio); err != nil {
			fmt.Fprintf(os.Stderr, "Error during search: %v\n", err)
		}
	}

	return nil
}

// performSearch executes the search and prints the results.
func performSearch(ctx context.Context, client meilisearch.ServiceReader, query string, semanticRatio float64) error {
	// Basic validation
	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	if semanticRatio < 0 || semanticRatio > 1 {
		return fmt.Errorf("semantic ratio must be between 0.0 and 1.0")
	}

	// Prepare the search request for hybrid search
	fmt.Printf("\nSearching for: '%s' (ratio: %.2f)...\n", query, semanticRatio)

	// Set up the Meilisearch search request with hybrid search parameters.
	// TODO: Allow configuring the embedder name if necessary later.
	// The semantic ratio comes from the global flag or environment variable.
	// Uses the default limit of 10 results for now,
	//      and the default embedder defined in meilisearchutil.
	searchRequest := &meilisearch.SearchRequest{
		Hybrid: &meilisearch.SearchRequestHybrid{
			SemanticRatio: semanticRatio,
		},
		AttributesToRetrieve: []string{"path", "content"},
		Limit:                10, // Add a default limit
	}

	// Execute the search using the utility function
	resp, err := meilisearchutil.SearchDocument(ctx, client, query, searchRequest)
	if err != nil {
		return fmt.Errorf("failed to search documents: %w", err)
	}

	// Format and print results
	if len(resp.Hits) == 0 {
		fmt.Println("No documents found.")
		return nil
	}

	fmt.Printf("Found %d documents:\n", len(resp.Hits)) // Removed EstimatedTotalHits and ProcessingTimeMs
	for i, hit := range resp.Hits {
		fmt.Printf(" %d. Score: %.4f\n", i+1, hit.Score)
		fmt.Printf("    Path: %s\n", hit.Path) // Use hit.Path directly
		fmt.Printf("    ID: %s\n", hit.ID)
		fmt.Printf("    Kind: %s\n", hit.Kind)
		fmt.Printf("    StartLine: %d\n", hit.StartLine)
		fmt.Printf("    EndLine: %d\n", hit.EndLine)

		// Optionally truncate content for brevity
		content := hit.Text
		maxLen := 150
		if len(content) > maxLen {
			content = content[:maxLen] + "..."
		}
		fmt.Printf("    Text: %s\n\n", content)
	}

	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var cli CLI
	k := kong.Parse(&cli,
		kong.Name("meilisearch-hymcp"),
		kong.Description("Meilisearch Hybrid Search MCP & CLI"),
		kong.UsageOnError(),
		kong.Bind(ctx),
	)

	switch k.Command() {
	case "build <dirs>":
		err := cli.Build.Run(ctx, &cli)
		k.FatalIfErrorf(err)
		log.Println("Indexing completed successfully.")
	case "serve":
		err := cli.Serve.Run(ctx, &cli)
		k.FatalIfErrorf(err)
		log.Println("MCP server finished.")
	case "search <query>":
		err := cli.Search.Run(ctx, &cli)
		k.FatalIfErrorf(err)
	default:
		k.FatalIfErrorf(fmt.Errorf("unknown command %s", k.Command()))
	}
}
