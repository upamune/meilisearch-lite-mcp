package meilisearchutil

import (
	"context"
	"encoding/json"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

const (
	DocsIndexName  = "docs"
	DocsPrimaryKey = "id"

	DefaultEmbedderName = "default"
)

func NewClient(meiliHttpAddr, meiliMasterKey string) meilisearch.ServiceManager {
	return meilisearch.New(meiliHttpAddr,
		meilisearch.WithAPIKey(meiliMasterKey))
}

type DocumentResponse struct {
	ID        string  `json:"id"`
	Path      string  `json:"path"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Text      string  `json:"text"`
	Kind      string  `json:"kind"`
	Score     float64 `json:"_rankingScore"`
}

type SearchResp[T any] struct {
	Hits []T `json:"hits"`
}

func SearchDocument(
	ctx context.Context,
	client meilisearch.ServiceReader,
	query string,
	searchRequest *meilisearch.SearchRequest,
) (*SearchResp[DocumentResponse], error) {
	return search[DocumentResponse](
		ctx, client, DocsIndexName, query, searchRequest,
	)
}

func search[T any](
	ctx context.Context,
	client meilisearch.ServiceReader,
	indexName string,
	query string,
	searchRequest *meilisearch.SearchRequest,
) (*SearchResp[T], error) {
	raw, err := client.
		Index(indexName).SearchRawWithContext(
		ctx, query, searchRequest,
	)
	if err != nil {
		return nil, err
	}

	var resp SearchResp[T]
	if err := json.Unmarshal(*raw, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func CreateDocumentIndex(
	ctx context.Context,
	client meilisearch.ServiceManager,
	embedder meilisearch.Embedder,
) (meilisearch.IndexManager, error) {
	return createIndex(
		ctx,
		client,
		DocsIndexName,
		DocsPrimaryKey,
		&meilisearch.Settings{
			SearchableAttributes: []string{"text", "headings"},
			FilterableAttributes: []string{"path", "kind"},
			Embedders: map[string]meilisearch.Embedder{
				DefaultEmbedderName: embedder,
			},
		},
	)
}

func createIndex(
	ctx context.Context,
	client meilisearch.ServiceManager,
	uid string,
	primaryKey string,
	settings *meilisearch.Settings,
) (meilisearch.IndexManager, error) {
	task, err := client.CreateIndexWithContext(ctx, &meilisearch.IndexConfig{
		Uid:        uid,
		PrimaryKey: primaryKey,
	})
	if err != nil {
		return nil, err
	}
	if _, err := client.WaitForTask(task.TaskUID, 3*time.Second); err != nil {
		return nil, err
	}

	idx := client.Index(uid)
	task, err = idx.UpdateSettingsWithContext(ctx, settings)
	if err != nil {
		return nil, err
	}
	if _, err := client.WaitForTask(task.TaskUID, 3*time.Second); err != nil {
		return nil, err
	}

	return client.Index(uid), nil
}
