package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
	"net/http"
	"strconv"
	"strings"
	"time"
	"kkg-backend/internal/oj/common"
	"kkg-backend/internal/oj/model/entity"
)

func (h *Handler) QuestionFirstACRank24h(c *gin.Context) {
	limit := int64(20)
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	conn := h.redisPool.Get()
	defer conn.Close()
	if err := h.applyExpiredFirstACEvents(conn); err != nil {
		c.JSON(http.StatusOK, common.Success(map[string]interface{}{
			"windowHours": 24,
			"records":     []interface{}{},
		}))
		return
	}
	values, err := redis.Values(conn.Do("ZREVRANGE", firstACRankZSetKey, 0, limit-1, "WITHSCORES"))
	if err != nil {
		c.JSON(http.StatusOK, common.Success(map[string]interface{}{
			"windowHours": 24,
			"records":     []interface{}{},
		}))
		return
	}
	type rankItem struct {
		UserID       int64  `json:"userId"`
		BlogUserID   int64  `json:"blogUserId"`
		UserName     string `json:"userName"`
		UserAvatar   string `json:"userAvatar"`
		FirstACCount int64  `json:"firstAcCount"`
		Rank         int64  `json:"rank"`
	}
	items := make([]rankItem, 0, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		member, _ := redis.String(values[i], nil)
		score, _ := redis.Int64(values[i+1], nil)
		uid, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			continue
		}
		var u entity.User
		if err := h.db.Where("id = ? AND isDelete = 0", uid).First(&u).Error; err != nil {
			continue
		}
		blogUserID, _ := h.userSvc.SharedUserIDByAccount(u.UserAccount)
		items = append(items, rankItem{
			UserID:       uid,
			BlogUserID:   blogUserID,
			UserName:     u.UserName,
			UserAvatar:   u.UserAvatar,
			FirstACCount: score,
		})
	}
	for i := range items {
		items[i].Rank = int64(i + 1)
	}
	c.JSON(http.StatusOK, common.Success(map[string]interface{}{
		"windowHours": 24,
		"records":     items,
	}))
}

func (h *Handler) recordFirstAccepted24h(userID, questionID, submitID int64) {
	conn := h.redisPool.Get()
	defer conn.Close()
	if err := h.applyExpiredFirstACEvents(conn); err != nil {
		return
	}
	member := fmt.Sprintf("%d:%d:%d", userID, questionID, submitID)
	expireAt := time.Now().Add(firstACEventTTL).Unix()
	if _, err := conn.Do("MULTI"); err != nil {
		return
	}
	_ = conn.Send("ZINCRBY", firstACRankZSetKey, 1, strconv.FormatInt(userID, 10))
	_ = conn.Send("ZADD", firstACExpireZSetKey, expireAt, member)
	_ = conn.Send("EXPIRE", firstACRankZSetKey, int(firstACEventTTL.Seconds())+3600)
	_ = conn.Send("EXPIRE", firstACExpireZSetKey, int(firstACEventTTL.Seconds())+3600)
	_, _ = conn.Do("EXEC")
}

func (h *Handler) applyExpiredFirstACEvents(conn redis.Conn) error {
	now := time.Now().Unix()
	for {
		values, err := redis.Values(conn.Do("ZRANGEBYSCORE", firstACExpireZSetKey, "-inf", now, "LIMIT", 0, 200))
		if err != nil {
			return err
		}
		if len(values) == 0 {
			return nil
		}
		for _, raw := range values {
			member, err := redis.String(raw, nil)
			if err != nil {
				continue
			}
			parts := strings.Split(member, ":")
			if len(parts) < 1 {
				continue
			}
			userID := parts[0]
			_, _ = conn.Do("ZINCRBY", firstACRankZSetKey, -1, userID)
			_, _ = conn.Do("ZREM", firstACExpireZSetKey, member)
		}
		_, _ = conn.Do("ZREMRANGEBYSCORE", firstACRankZSetKey, "-inf", 0)
	}
}
