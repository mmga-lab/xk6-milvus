package milvus

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/column"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/milvus", new(Milvus))
}

type Milvus struct{}

type Client struct {
	client *milvusclient.Client
	ctx    context.Context
}

func (*Milvus) Client(address string) (*Client, error) {
	ctx := context.Background()
	c, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus client: %v", err)
	}

	return &Client{
		client: c,
		ctx:    ctx,
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close(c.ctx)
}

func (c *Client) CreateCollection(collectionName string, dimension int64) error {
	schema := entity.NewSchema().
		WithName(collectionName).
		WithDescription("Test collection for k6").
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName("vector").WithDataType(entity.FieldTypeFloatVector).WithDim(dimension))

	option := milvusclient.NewCreateCollectionOption(collectionName, schema)
	err := c.client.CreateCollection(c.ctx, option)
	return err
}

func (c *Client) DropCollection(collectionName string) error {
	option := milvusclient.NewDropCollectionOption(collectionName)
	return c.client.DropCollection(c.ctx, option)
}

func (c *Client) HasCollection(collectionName string) (bool, error) {
	option := milvusclient.NewHasCollectionOption(collectionName)
	return c.client.HasCollection(c.ctx, option)
}

func (c *Client) Insert(collectionName string, vectors [][]float32) ([]int64, error) {
	vectorColumn := column.NewColumnFloatVector("vector", getDimension(vectors), vectors)
	
	option := milvusclient.NewColumnBasedInsertOption(collectionName, vectorColumn)
	result, err := c.client.Insert(c.ctx, option)
	if err != nil {
		return nil, fmt.Errorf("failed to insert: %v", err)
	}
	
	// Return placeholder IDs for now 
	ids := make([]int64, len(vectors))
	for i := range ids {
		ids[i] = int64(i)
	}
	
	// Check if insert was successful
	if result.InsertCount != int64(len(vectors)) {
		return nil, fmt.Errorf("insert count mismatch: expected %d, got %d", len(vectors), result.InsertCount)
	}
	
	return ids, nil
}

func (c *Client) Search(collectionName string, vectors [][]float32, topK int) ([]SearchResult, error) {
	searchVectors := make([]entity.Vector, len(vectors))
	for i, v := range vectors {
		searchVectors[i] = entity.FloatVector(v)
	}

	option := milvusclient.NewSearchOption(collectionName, topK, searchVectors).
		WithANNSField("vector").
		WithOutputFields("id")
	
	searchResult, err := c.client.Search(c.ctx, option)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %v", err)
	}

	var results []SearchResult
	for _, result := range searchResult {
		for i := 0; i < result.ResultCount; i++ {
			// Get ID from the IDs column
			idVal, err := result.IDs.Get(i)
			if err != nil {
				continue
			}
			id := idVal.(int64)
			score := result.Scores[i]
			results = append(results, SearchResult{
				ID:    id,
				Score: score,
			})
		}
	}

	return results, nil
}

func (c *Client) CreateIndex(collectionName string, fieldName string) error {
	idx := index.NewFlatIndex(entity.L2)
	
	option := milvusclient.NewCreateIndexOption(collectionName, fieldName, idx)
	task, err := c.client.CreateIndex(c.ctx, option)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}
	
	// Wait for index creation to complete
	err = task.Await(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for index creation: %v", err)
	}
	
	return nil
}

func (c *Client) LoadCollection(collectionName string) error {
	option := milvusclient.NewLoadCollectionOption(collectionName)
	task, err := c.client.LoadCollection(c.ctx, option)
	if err != nil {
		return err
	}
	
	// Wait for collection to be loaded
	return task.Await(c.ctx)
}

func (c *Client) ReleaseCollection(collectionName string) error {
	option := milvusclient.NewReleaseCollectionOption(collectionName)
	return c.client.ReleaseCollection(c.ctx, option)
}

type SearchResult struct {
	ID    int64
	Score float32
}

// Helper function to get dimension from vectors
func getDimension(vectors [][]float32) int {
	if len(vectors) > 0 {
		return len(vectors[0])
	}
	return 0
}