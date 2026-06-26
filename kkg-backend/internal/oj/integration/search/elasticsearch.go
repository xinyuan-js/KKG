package search

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ElasticsearchClient struct {
	baseURL string
	index   string
	client  *http.Client
}

type PostQuery struct {
	Current    int64
	PageSize   int64
	UserID     int64
	Title      string
	Content    string
	SearchText string
	Tags       []string
}

func NewElasticsearchClient(baseURL, index string, timeout time.Duration) *ElasticsearchClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &ElasticsearchClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		index:   strings.TrimSpace(index),
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *ElasticsearchClient) SearchPostIDs(req PostQuery) ([]int64, int64, error) {
	if c == nil || c.baseURL == "" || c.index == "" {
		return nil, 0, errors.New("elasticsearch config is incomplete")
	}
	from := (req.Current - 1) * req.PageSize
	query := map[string]interface{}{
		"from": from,
		"size": req.PageSize,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []interface{}{map[string]interface{}{"term": map[string]interface{}{"isDelete": 0}}},
				"must":   []interface{}{},
			},
		},
	}
	boolQ := query["query"].(map[string]interface{})["bool"].(map[string]interface{})
	filter := boolQ["filter"].([]interface{})
	must := boolQ["must"].([]interface{})
	if req.UserID > 0 {
		filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"userId": req.UserID}})
	}
	if req.Title != "" {
		must = append(must, map[string]interface{}{"match": map[string]interface{}{"title": req.Title}})
	}
	if req.Content != "" {
		must = append(must, map[string]interface{}{"match": map[string]interface{}{"content": req.Content}})
	}
	if req.SearchText != "" {
		must = append(must, map[string]interface{}{"multi_match": map[string]interface{}{"query": req.SearchText, "fields": []string{"title", "content", "description"}}})
	}
	for _, tag := range req.Tags {
		filter = append(filter, map[string]interface{}{"term": map[string]interface{}{"tags": tag}})
	}
	boolQ["filter"] = filter
	boolQ["must"] = must

	body, _ := json.Marshal(query)
	resp, err := c.client.Post(c.baseURL+"/"+c.index+"/_search", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err = json.Unmarshal(respBody, &result); err != nil {
		return nil, 0, err
	}
	hitsObj, ok := result["hits"].(map[string]interface{})
	if !ok {
		return nil, 0, errors.New("invalid es hits")
	}
	total := int64(0)
	if totalObj, ok := hitsObj["total"].(map[string]interface{}); ok {
		if v, ok := totalObj["value"].(float64); ok {
			total = int64(v)
		}
	}
	rawHits, ok := hitsObj["hits"].([]interface{})
	if !ok {
		return nil, total, nil
	}
	ids := make([]int64, 0, len(rawHits))
	for _, raw := range rawHits {
		hit, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		src, _ := hit["_source"].(map[string]interface{})
		if src != nil {
			if v, ok := src["id"].(float64); ok {
				ids = append(ids, int64(v))
				continue
			}
		}
		if sid, ok := hit["_id"].(string); ok {
			if id, err := strconv.ParseInt(sid, 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
	}
	return ids, total, nil
}
