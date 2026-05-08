package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

func NewMinIOStorage(endpoint string, accessKey string, secretKey string, bucket string, publicBaseURL string, useSSL bool) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("init minio client failed: %w", err)
	}

	s := &MinIOStorage{
		client:        client,
		bucket:        bucket,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}
	if err := s.ensureBucket(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *MinIOStorage) UploadImage(ctx context.Context, r io.Reader, size int64, contentType string, ext string) (string, error) {
	key := s.newObjectKey(ext)
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload image failed: %w", err)
	}
	return fmt.Sprintf("%s/%s/%s", s.publicBaseURL, s.bucket, key), nil
}

func (s *MinIOStorage) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket exists failed: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket failed: %w", err)
		}
	}

	policyObj := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect":    "Allow",
				"Principal": map[string]interface{}{"AWS": []string{"*"}},
				"Action":    []string{"s3:GetObject"},
				"Resource":  []string{fmt.Sprintf("arn:aws:s3:::%s/*", s.bucket)},
			},
		},
	}
	raw, err := json.Marshal(policyObj)
	if err != nil {
		return fmt.Errorf("marshal bucket policy failed: %w", err)
	}
	if err := s.client.SetBucketPolicy(ctx, s.bucket, string(raw)); err != nil {
		return fmt.Errorf("set bucket policy failed: %w", err)
	}
	return nil
}

func (s *MinIOStorage) newObjectKey(ext string) string {
	now := time.Now()
	cleanExt := strings.TrimPrefix(strings.ToLower(ext), ".")
	if cleanExt == "" {
		cleanExt = "jpg"
	}
	name := fmt.Sprintf("%d-%06d.%s", now.UnixMilli(), rand.Intn(1000000), cleanExt)
	return path.Join("images", now.Format("2006"), now.Format("01"), now.Format("02"), name)
}
