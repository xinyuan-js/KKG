package bootstrap

import (
	"fmt"
	"log"

	"kkg-backend/internal/config"
	"kkg-backend/internal/infra/cache"
	"kkg-backend/internal/infra/db"
	"kkg-backend/internal/search"
	"kkg-backend/internal/session"
	"kkg-backend/internal/storage"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Config   *config.Config
	DB       *gorm.DB
	Redis    *redis.Client
	Sessions *session.Manager
	Storage  *storage.MinIOStorage
	ES       *search.Client
}

func NewApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	db, err := db.NewMySQL(cfg)
	if err != nil {
		return nil, err
	}

	rdb, err := cache.NewRedisClient(cfg)
	if err != nil {
		return nil, err
	}
	objStorage, err := storage.NewMinIOStorage(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOPublicBaseURL,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		return nil, err
	}
	esClient, err := search.NewClient(cfg.ElasticsearchURL)
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		return nil, err
	}
	if err := ensureSuperAdmin(db, cfg); err != nil {
		return nil, fmt.Errorf("ensure super admin failed: %w", err)
	}
	if err := syncSearchData(db, esClient, cfg); err != nil {
		log.Printf("warn: sync search data failed on startup, continue without blocking service: %v", err)
	}

	return &App{Config: cfg, DB: db, Redis: rdb, Sessions: session.NewManager(rdb), Storage: objStorage, ES: esClient}, nil
}
