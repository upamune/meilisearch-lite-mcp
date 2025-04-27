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

// Implement indexer.Config interface for IndexCmd
func (c *IndexCmd) GetDirs() []string    { return c.Dirs }
func (c *IndexCmd) GetChunkSize() int    { return c.ChunkSize }
func (c *IndexCmd) GetChunkOverlap() int { return c.ChunkOverlap }
func (c *IndexCmd) GetConcurrency() int  { return c.Concurrency }

type ServeCmd struct {
	// HttpAddr and SemanticRatio are now global flags in CLI
	PathPrefix string `env:"SEARCH_PATH_PREFIX" default:"" help:"Prefix for search paths (optional)."`
}

// SearchCmd defines arguments for the search command
type SearchCmd struct {
	Query       string `arg:"" required:"" help:"Search query."`
	Interactive bool   `short:"i" help:"Enable interactive mode."`
}

// serveAdapter wraps main.ServeCmd to satisfy mcpserver.ServeCmd interface
// It now accesses global flags via the main CLI struct.
type serveAdapter struct {
	cli *CLI // Keep a reference to the main CLI struct for global flags
}

func (s serveAdapter) PathPrefix() string     { return s.cli.Serve.PathPrefix }
func (s serveAdapter) HttpAddr() string       { return s.cli.HttpAddr }
func (s serveAdapter) SemanticRatio() float64 { return s.cli.SemanticRatio }

func (c *SearchCmd) Run(cli *CLI) error {
	ctx := context.Background()
	// NewClient reads env vars set in main()
	client := meilisearchutil.NewClient()

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
		// Optionally truncate content for brevity
		content := hit.Text // Use hit.Text directly
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
		// Bind the CLI struct itself to the context so subcommands can access global flags if needed
		// (though serveAdapter accesses it directly via closure here)
		kong.BindTo(&cli, (*CLI)(nil)),
	)

	// Set environment variables for Meilisearch client initialization
	// Why: Centralizes config access, keeps NewClient simple.
	os.Setenv("MEILI_HTTP_ADDR", cli.HttpAddr)
	os.Setenv("MEILI_MASTER_KEY", cli.ApiKey)
	// Note: SemanticRatio is passed directly to search function

	switch k.Command() {
	case "build <dirs>":
		// indexer.Run does not return an error
		indexer.Run(ctx, &cli.Build)
		log.Println("Indexing completed successfully.") // Provide feedback
	case "serve":
		// mcpserver.Run does not return an error
		mcpserver.Run(ctx, serveAdapter{cli: &cli})
		log.Println("MCP server finished.") // Provide feedback
	case "search <query>":
		if err := cli.Search.Run(&cli); err != nil {
			log.Fatal(err)
		}

	default:
		// Kong should handle unknown commands with UsageOnError
		k.FatalIfErrorf(fmt.Errorf("unknown command %s", k.Command()))
	}
}
