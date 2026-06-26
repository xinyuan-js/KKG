package job

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"kkg-backend/internal/config"
	"kkg-backend/internal/oj/model/entity"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func StartEsSync(db *gorm.DB, cfg config.OJConfig, log *zap.Logger) {
	if !cfg.ES.Enabled {
		return
	}
	go func() {
		fullSync(db, cfg, log)
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			incSync(db, cfg, log)
		}
	}()
}

func fullSync(db *gorm.DB, cfg config.OJConfig, log *zap.Logger) {
	var posts []entity.Post
	if err := db.Where("isDelete in (0,1)").Find(&posts).Error; err != nil {
		log.Error("es full sync query failed", zap.Error(err))
		return
	}
	for _, p := range posts {
		_ = indexPost(cfg, p)
	}
	log.Info("es full sync done", zap.Int("count", len(posts)))
}

func incSync(db *gorm.DB, cfg config.OJConfig, log *zap.Logger) {
	minTime := time.Now().Add(-5 * time.Minute)
	var posts []entity.Post
	if err := db.Where("updateTime >= ?", minTime).Find(&posts).Error; err != nil {
		log.Error("es inc sync query failed", zap.Error(err))
		return
	}
	for _, p := range posts {
		_ = indexPost(cfg, p)
	}
	if len(posts) > 0 {
		log.Info("es inc sync done", zap.Int("count", len(posts)))
	}
}

func indexPost(cfg config.OJConfig, p entity.Post) error {
	bodyMap := map[string]interface{}{
		"id":         p.ID,
		"title":      p.Title,
		"content":    p.Content,
		"tags":       parseTags(p.Tags),
		"thumbNum":   p.ThumbNum,
		"favourNum":  p.FavourNum,
		"userId":     p.UserID,
		"createTime": p.CreateTime,
		"updateTime": p.UpdateTime,
		"isDelete":   p.IsDelete,
	}
	body, _ := json.Marshal(bodyMap)
	url := fmt.Sprintf("%s/%s/_doc/%d", strings.TrimRight(cfg.ES.URL, "/"), cfg.ES.Index, p.ID)
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func parseTags(tagsJSON string) []string {
	var tags []string
	_ = json.Unmarshal([]byte(tagsJSON), &tags)
	return tags
}
