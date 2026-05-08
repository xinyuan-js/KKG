package bootstrap

import (
	"awesomeProject/internal/config"
	"awesomeProject/internal/model"
	"awesomeProject/internal/search"
	"awesomeProject/internal/storage"
	"awesomeProject/pkg/security"
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type App struct {
	Config  *config.Config
	DB      *gorm.DB
	Redis   *redis.Client
	Storage *storage.MinIOStorage
	ES      *search.Client
}

func NewApp() (*App, error) {
	cfg := config.Load()

	db, err := initMySQL(cfg)
	if err != nil {
		return nil, err
	}

	rdb, err := initRedis(cfg)
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

	if err := db.AutoMigrate(&model.User{}, &model.Post{}, &model.PostVersion{}, &model.Comment{}, &model.Notification{}, &model.Tweet{}, &model.PostLike{}, &model.PostFavorite{}, &model.AdminAuditLog{}); err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}
	if err := ensureSuperAdmin(db, cfg); err != nil {
		return nil, fmt.Errorf("ensure super admin failed: %w", err)
	}
	if err := syncSearchData(db, esClient, cfg); err != nil {
		log.Printf("warn: sync search data failed on startup, continue without blocking service: %v", err)
	}

	return &App{Config: cfg, DB: db, Redis: rdb, Storage: objStorage, ES: esClient}, nil
}

func ensureSuperAdmin(db *gorm.DB, cfg *config.Config) error {
	username := strings.TrimSpace(cfg.SuperAdminUsername)
	email := strings.TrimSpace(cfg.SuperAdminEmail)
	password := strings.TrimSpace(cfg.SuperAdminPassword)
	if username == "" || email == "" || password == "" {
		return nil
	}
	if len(password) < 8 {
		return errors.New("SUPER_ADMIN_PASSWORD must be at least 8 characters")
	}

	var user model.User
	err := db.Where("username = ? OR email = ?", username, email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, hashErr := security.HashPassword(password)
		if hashErr != nil {
			return hashErr
		}
		u := &model.User{
			Username:     username,
			Email:        email,
			PasswordHash: hash,
			Role:         "super_admin",
			Status:       1,
		}
		return db.Create(u).Error
	}
	if err != nil {
		return err
	}

	updates := map[string]interface{}{}
	if user.Role != "super_admin" {
		updates["role"] = "super_admin"
	}
	if user.Status != 1 {
		updates["status"] = 1
	}
	if len(updates) == 0 {
		return nil
	}
	return db.Model(&model.User{}).Where("id = ?", user.ID).Updates(updates).Error
}

func syncSearchData(db *gorm.DB, es *search.Client, cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var posts []model.Post
	if err := db.Where("status = ?", "published").Find(&posts).Error; err != nil {
		return err
	}
	for _, p := range posts {
		doc := map[string]interface{}{
			"id":         p.ID,
			"author_id":  p.AuthorID,
			"title":      p.Title,
			"summary":    p.Summary,
			"tags":       p.Tags,
			"updated_at": p.UpdatedAt.Format(time.RFC3339),
		}
		if p.PublishAt != nil {
			doc["publish_at"] = p.PublishAt.Format(time.RFC3339)
		}
		if err := es.Index(ctx, cfg.ESPostIndex, strconv.FormatUint(p.ID, 10), doc); err != nil {
			return err
		}
	}
	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		doc := map[string]interface{}{
			"id":         u.ID,
			"username":   u.Username,
			"email":      u.Email,
			"avatar_url": u.AvatarURL,
			"updated_at": u.UpdatedAt.Format(time.RFC3339),
		}
		if err := es.Index(ctx, cfg.ESUserIndex, strconv.FormatUint(u.ID, 10), doc); err != nil {
			return err
		}
	}
	return nil
}

func initMySQL(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.MySQLUser, cfg.MySQLPass, cfg.MySQLHost, cfg.MySQLPort, cfg.MySQLDB)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect mysql failed: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db failed: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql failed: %w", err)
	}

	return db, nil
}

func initRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPass,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis failed: %w", err)
	}

	return client, nil
}
