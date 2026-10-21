package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIO struct {
	client *minio.Client
	bucket string
	public string // optional public base for clients
}

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	PublicURL string
}

func NewMinIO(cfg Config) (*MinIO, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("minio endpoint empty")
	}
	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	m := &MinIO{client: cli, bucket: cfg.Bucket, public: cfg.PublicURL}
	ctx := context.Background()
	exists, err := cli.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := cli.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}
	return m, nil
}

type PresignResult struct {
	UploadURL string `json:"uploadUrl"`
	ObjectKey string `json:"objectKey"`
	Bucket    string `json:"bucket"`
	ExpiresIn int64  `json:"expiresIn"`
}

func (m *MinIO) PresignPut(ctx context.Context, contentType, ext string, ttl time.Duration) (*PresignResult, error) {
	if ext == "" {
		ext = "bin"
	}
	key := fmt.Sprintf("uploads/%s/%s.%s", time.Now().Format("2006/01/02"), uuid.NewString(), ext)
	u, err := m.client.PresignedPutObject(ctx, m.bucket, key, ttl)
	if err != nil {
		return nil, err
	}
	upload := u.String()
	if m.public != "" {
		// rewrite host for browser-facing URL if needed
		if pu, err := url.Parse(m.public); err == nil && pu.Host != "" {
			u.Scheme = pu.Scheme
			u.Host = pu.Host
			upload = u.String()
		}
	}
	return &PresignResult{
		UploadURL: upload,
		ObjectKey: key,
		Bucket:    m.bucket,
		ExpiresIn: int64(ttl.Seconds()),
	}, nil
}
