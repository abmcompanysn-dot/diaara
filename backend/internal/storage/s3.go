package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Storage struct {
	client *s3.Client
	// presignClient sert UNIQUEMENT à générer les URL signées remises au
	// navigateur : il pointe sur PublicEndpoint (accessible depuis Internet).
	// client, lui, pointe sur Endpoint (réseau interne) pour Upload/Download/
	// Ping côté serveur. Si PublicEndpoint est vide, les deux sont identiques.
	presignClient *s3.Client
	bucket        string
}

type S3Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Endpoint        string // ex: https://fly.storage.tigris.dev (Tigris) ou https://<ACCOUNT>.r2.cloudflarestorage.com (R2)
	// PublicEndpoint : endpoint joignable depuis le navigateur, utilisé pour
	// signer les URL de téléchargement. Nécessaire quand le stockage n'est
	// pas directement exposé (MinIO derrière un reverse proxy : Endpoint =
	// http://minio:9000 interne, PublicEndpoint = https://files.exemple.com).
	// Vide => on signe avec Endpoint (cas Tigris/R2, déjà publics).
	PublicEndpoint string
	Region         string // défaut: auto
}

func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	region := cfg.Region
	if region == "" {
		region = "auto"
	}

	creds := credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(creds),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
	})

	presignClient := client
	if cfg.PublicEndpoint != "" && cfg.PublicEndpoint != cfg.Endpoint {
		presignClient = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.PublicEndpoint)
			o.UsePathStyle = true
		})
	}

	return &S3Storage{
		client:        client,
		presignClient: presignClient,
		bucket:        cfg.Bucket,
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, key string, data []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (s *S3Storage) Download(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *S3Storage) GenerateSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presigner := s3.NewPresignClient(s.presignClient)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(o *s3.PresignOptions) {
		o.Expires = expiry
	})
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// Ping vérifie que le bucket configuré est joignable (utilisé par le
// endpoint de santé infra de l'admin).
func (s *S3Storage) Ping(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	return err
}

func (s *S3Storage) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// NewFileKey génère une clé objet unique pour le stockage objet.
func NewFileKey(vendorID, filename string) string {
	buf := make([]byte, 8)
	rand.Read(buf)
	return fmt.Sprintf("%s/%s_%s", vendorID, hex.EncodeToString(buf), filename)
}
