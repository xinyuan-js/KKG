package cache

import (
	"strings"
	"time"

	"kkg-backend/internal/config"

	"github.com/gomodule/redigo/redis"
)

func NewRedisPool(cfg config.OJRedisConfig) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 5 * time.Minute,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", cfg.Addr)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(cfg.Password) != "" {
				if _, err = c.Do("AUTH", cfg.Password); err != nil {
					msg := strings.ToLower(err.Error())
					if !strings.Contains(msg, "without any password configured") {
						_ = c.Close()
						return nil, err
					}
				}
			}
			if cfg.DB > 0 {
				if _, err = c.Do("SELECT", cfg.DB); err != nil {
					_ = c.Close()
					return nil, err
				}
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < 30*time.Second {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}
