package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/meilisearch/meilisearch-go"
	"github.com/upamune/meilisearch-hybrid-mcp/meilisearchutil"
)

type searchToolRequest struct {
	Query         string  `json:"query" validate:"required"`
	TopK          int     `json:"top_k" validate:"min=1,max=50"`
	SemanticRatio float64 `json:"semantic_ratio" validate:"min=0,max=1"`
}

type searchToolResult struct {
	Rank      int     `json:"rank"`
	FilePath  string  `json:"file_path"`
	ChunkID   string  `json:"chunk_id"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Text      string  `json:"text"`
	Kind      string  `json:"kind"`
	Score     float64 `json:"_rankingScore"`
}

type SearchResponse struct {
	Results       []searchToolResult `json:"results"`
	TopK          int                `json:"top_k"`
	SemanticRatio float64            `json:"semantic_ratio"`
}

type ServeCmd interface {
	HttpAddr() string
	ApiKey() string
	SemanticRatio() float64
	PathPrefix() string
}

const (
	queryParamName         = "query"
	topKParamName          = "top_k"
	semanticRatioParamName = "semantic_ratio"
)

type RunParam struct {
	HttpAddr      string
	ApiKey        string
	SemanticRatio float64
	PathPrefix    string
}

// Define the function type for dependency injection
type searchFunc func(ctx context.Context, client meilisearch.ServiceReader, query string, searchRequest *meilisearch.SearchRequest) (*meilisearchutil.SearchResp[meilisearchutil.DocumentResponse], error)

func Run(ctx context.Context, param RunParam) error {
	client := meilisearchutil.NewClient(param.HttpAddr, param.ApiKey)
	defer client.Close()

	srv := server.NewMCPServer("meilisearch-hybrid-mcp", "v0.1.0")

	searchDocumentTool := mcp.NewTool(
		"search_document",
		mcp.WithDescription("Search documents using Meilisearch search"),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "Search Document",
			ReadOnlyHint:    true,
			DestructiveHint: false,
			IdempotentHint:  true,
			OpenWorldHint:   false,
		}),
		mcp.WithString(
			"query",
			mcp.Required(),
			mcp.Description("Search query"),
		),
		mcp.WithNumber(
			"top_k",
			mcp.DefaultNumber(5),
			mcp.Min(1),
			mcp.Max(50),
			mcp.Description("Number of results to return"),
		),
		mcp.WithNumber(
			"semantic_ratio",
			mcp.DefaultNumber(0.5),
			mcp.Min(0.0),
			mcp.Max(1.0),
			mcp.Description("Semantic search ratio"),
		),
	)

	handler := newSearchDocumentHandler(client, param.PathPrefix)
	srv.AddTool(searchDocumentTool, handler)

	if err := server.ServeStdio(srv); err != nil {
		return err
	}

	return nil
}

func newSearchDocumentHandler(client meilisearch.ServiceReader, pathPrefix string) server.ToolHandlerFunc {
	// The actual search function to use in production
	actualSearch := func(ctx context.Context, passedClient meilisearch.ServiceReader, query string, searchRequest *meilisearch.SearchRequest) (*meilisearchutil.SearchResp[meilisearchutil.DocumentResponse], error) {
		// Use the client captured by the closure
		return meilisearchutil.SearchDocument(ctx, client, query, searchRequest)
	}
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Pass the search function and necessary context (client is needed by the signature, though mocks might ignore it)
		return searchDocumentHandler(ctx, request, actualSearch, client, pathPrefix)
	}
}

func searchDocumentHandler(ctx context.Context, request mcp.CallToolRequest, searcher searchFunc, client meilisearch.ServiceReader, pathPrefix string) (*mcp.CallToolResult, error) {
	query, ok := request.Params.Arguments[queryParamName].(string)
	if !ok {
		return nil, fmt.Errorf("expected string for query, got %T", request.Params.Arguments[queryParamName])
	}
	topK, ok := request.Params.Arguments[topKParamName].(float64)
	if !ok {
		return nil, fmt.Errorf("expected float64 for top_k, got %T", request.Params.Arguments[topKParamName])
	}
	semanticRatio, ok := request.Params.Arguments[semanticRatioParamName].(float64)
	if !ok {
		return nil, fmt.Errorf("expected float64 for semantic_ratio, got %T", request.Params.Arguments[semanticRatioParamName])
	}

	searchRequest := &meilisearch.SearchRequest{
		Limit: int64(topK),
		Hybrid: &meilisearch.SearchRequestHybrid{
			Embedder:      meilisearchutil.DefaultEmbedderName,
			SemanticRatio: semanticRatio,
		},
	}

	// Use the injected searcher function
	searchResponse, err := searcher(ctx, client, query, searchRequest) // Pass client for signature, though mocks might ignore it
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{Text: err.Error()},
			},
		}, nil
	}

	res := SearchResponse{TopK: int(topK), SemanticRatio: semanticRatio}
	for i, hit := range searchResponse.Hits {
		res.Results = append(res.Results, searchToolResult{
			Rank:      i + 1,
			FilePath:  filepath.Join(pathPrefix, hit.Path),
			ChunkID:   hit.ID,
			StartLine: hit.StartLine,
			EndLine:   hit.EndLine,
			Text:      hit.Text,
			Score:     hit.Score,
			Kind:      hit.Kind,
		})
	}

	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(res); err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{Text: err.Error()},
			},
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Text: buf.String()},
		},
	}, nil
}
