package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"smart-fridge-backend/internal/config"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

func New(ctx context.Context, cfg config.Storage) (*S3Storage, error) {
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil {
		return nil, fmt.Errorf("storage endpoint must be an HTTP(S) URL")
	}
	if cfg.Bucket == "" || cfg.Region == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("storage bucket, region, access key ID and secret access key are required")
	}
	awsCfg, err := awscfg.LoadDefaultConfig(
		ctx,
		awscfg.WithRegion(cfg.Region),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load storage configuration: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})

	return &S3Storage{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucket:        cfg.Bucket,
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, fileKey string, fileReader io.Reader, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(fileKey),
		Body:        fileReader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("upload image: %w", err)
	}

	return nil
}

func (s *S3Storage) GetURL(ctx context.Context, fileKey string, expiresAfter time.Duration) (string, error) {
	presignedReq, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileKey),
	}, s3.WithPresignExpires(expiresAfter))

	if err != nil {
		return "", fmt.Errorf("get image url: %w", err)
	}

	return presignedReq.URL, nil
}

func (s *S3Storage) Delete(ctx context.Context, fileKey string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		return fmt.Errorf("delete image: %w", err)
	}

	return nil
}
