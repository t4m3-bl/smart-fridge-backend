package storage

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"smart-fridge-backend/internal/config"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

func New(ctx context.Context, cfg config.Storage) (*S3Storage, error) {
	endpoint, err := url.Parse(cfg.Endpoint)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil {
		return nil, errors.New("storage endpoint must be an HTTP(S) URL")
	}
	if cfg.Bucket == "" || cfg.Region == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, errors.New("storage bucket, region, access key ID and secret access key are required")
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

func (s *S3Storage) PresignPut(
	ctx context.Context,
	key string,
	contentType string,
	uploadTTL time.Duration,
) (string, map[string]string, error) {
	request, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	},
		s3.WithPresignExpires(uploadTTL),
	)
	if err != nil {
		return "", nil, fmt.Errorf("presign image upload: %w", err)
	}

	return request.URL, s.getHeaders(request), nil
}

func (s *S3Storage) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if _, ok := errors.AsType[*types.NotFound](err); ok {
			return false, nil
		}
		return false, fmt.Errorf("head object %s: %w", key, err)
	}

	return true, nil
}

func (s *S3Storage) getHeaders(request *v4.PresignedHTTPRequest) map[string]string {
	headers := make(map[string]string, len(request.SignedHeader))
	for name, values := range request.SignedHeader {
		if strings.EqualFold(name, "host") || len(values) == 0 {
			continue
		}

		headers[name] = strings.Join(values, ",")
	}

	return headers
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete image %s: %w", key, err)
	}

	return nil
}
