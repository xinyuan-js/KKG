package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Client struct {
	client *elasticsearch.Client
}

func NewClient(baseURL string) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("elasticsearch url is required")
	}
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{baseURL},
	})
	if err != nil {
		return nil, fmt.Errorf("create elasticsearch client failed: %w", err)
	}
	return &Client{client: es}, nil
}

func (c *Client) Index(ctx context.Context, index string, id string, doc interface{}) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewReader(body),
		Refresh:    "false",
	}
	resp, err := req.Do(ctx, c.client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("elasticsearch index failed: status=%s body=%s", resp.Status(), string(raw))
	}
	return nil
}

func (c *Client) Delete(ctx context.Context, index string, id string) error {
	req := esapi.DeleteRequest{
		Index:      index,
		DocumentID: id,
		Refresh:    "false",
	}
	resp, err := req.Do(ctx, c.client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil
	}
	if resp.IsError() {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("elasticsearch delete failed: status=%s body=%s", resp.Status(), string(raw))
	}
	return nil
}

func (c *Client) CreateIndex(ctx context.Context, index string, body interface{}) error {
	rawBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req := esapi.IndicesCreateRequest{
		Index: index,
		Body:  bytes.NewReader(rawBody),
	}
	resp, err := req.Do(ctx, c.client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if strings.Contains(string(raw), "resource_already_exists_exception") {
			return nil
		}
		return fmt.Errorf("elasticsearch create index failed: status=%s body=%s", resp.Status(), string(raw))
	}
	if resp.IsError() {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("elasticsearch create index failed: status=%s body=%s", resp.Status(), string(raw))
	}
	return nil
}

type SearchResult struct {
	ID       string          `json:"id"`
	Score    float64         `json:"score"`
	Source   json.RawMessage `json:"source"`
	SortData []interface{}   `json:"sort,omitempty"`
}

func (c *Client) Search(ctx context.Context, index string, query interface{}) ([]SearchResult, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	req := esapi.SearchRequest{
		Index: []string{index},
		Body:  bytes.NewReader(body),
	}
	resp, err := req.Do(ctx, c.client)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return []SearchResult{}, nil
	}
	if resp.IsError() {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("elasticsearch search failed: status=%s body=%s", resp.Status(), string(raw))
	}

	var parsed struct {
		Hits struct {
			Hits []struct {
				ID     string          `json:"_id"`
				Score  float64         `json:"_score"`
				Source json.RawMessage `json:"_source"`
				Sort   []interface{}   `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	results := make([]SearchResult, 0, len(parsed.Hits.Hits))
	for _, hit := range parsed.Hits.Hits {
		results = append(results, SearchResult{
			ID:       hit.ID,
			Score:    hit.Score,
			Source:   hit.Source,
			SortData: hit.Sort,
		})
	}
	return results, nil
}
