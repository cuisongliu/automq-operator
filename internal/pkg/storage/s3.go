/*
Copyright 2023 cuisongliu@qq.com.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package storage

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

func awsConfig(cfg Config) (*aws.Config, *http.Client, error) {
	cred := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.Key, cfg.Secret, ""))
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:           cfg.Endpoint,
				SigningRegion: cfg.Region,
			}, nil
		})

	if cfg.MaxBackoffDelay == nil {
		m := int32(20)
		cfg.MaxBackoffDelay = &m
	}
	if cfg.MaxRetryAttempts == nil {
		m := int32(10)
		cfg.MaxRetryAttempts = &m
	}
	httpCli := getHttpClient()
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(cred),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithHTTPClient(httpCli),
		config.WithRetryer(func() aws.Retryer {
			r := retry.AddWithMaxAttempts(retry.NewStandard(), int(*cfg.MaxRetryAttempts))
			return retry.AddWithMaxBackoffDelay(r, time.Duration(*cfg.MaxBackoffDelay*1000*1000))
		}))
	return &awsCfg, httpCli, err
}

// s3Service defines the S3 service with the client and config.
type s3Service struct {
	client  *s3.Client
	cfg     Config
	httpCli *http.Client
}

// newS3Service creates a new S3Service with the provided config.
func newS3Service(cfg Config) (Storage, error) {
	awsCfg, hcli, err := awsConfig(cfg)
	//awsCfg.HTTPClient
	if err != nil {
		return nil, err
	}
	cli := s3.NewFromConfig(*awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	if err != nil {
		return nil, err
	}
	return &s3Service{
		client:  cli,
		cfg:     cfg,
		httpCli: hcli,
	}, nil
}
func newS3BucketService(cfg Config) (Bucket, error) {
	awsCfg, hcli, err := awsConfig(cfg)
	//awsCfg.HTTPClient
	if err != nil {
		return nil, err
	}
	cli := s3.NewFromConfig(*awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	if err != nil {
		return nil, err
	}
	return &s3Service{
		client:  cli,
		cfg:     cfg,
		httpCli: hcli,
	}, nil
}

func (s *s3Service) MkBucket(ctx context.Context, bucketName string) error {
	_, _ = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucketName,
	})
	return nil
}

func (s *s3Service) HeadObjectFromAddress(ctx context.Context, address string) (bucket, object string, err error) {
	return getBucketKeyFromS3(address)
}
func (s *s3Service) HeadObjectFromNewURIs(ctx context.Context, address string, insertBucket, insertObject string, newURIs []string) (string, error) {
	bucket, object, err := s.HeadObjectFromAddress(ctx, address)
	if err != nil {
		return "", err
	}
	newURI := newURIs[rand.Intn(len(newURIs))]
	if insertBucket != "" {
		bucket = bucket + "_" + insertBucket
	}
	return fmt.Sprintf("http://%s", path.Join(newURI, bucket, insertObject, object)), nil
}
func (s *s3Service) HeadObject(ctx context.Context, bucketName, objectKey string) (bool, error) {
	//defer s.httpCli.CloseIdleConnections()
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucketName,
		Key:    &objectKey,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// Upload uploads an object to S3 bucket.
func (s *s3Service) Upload(ctx context.Context, bucketName, objectKey string, objectData []byte) error {
	//defer s.httpCli.CloseIdleConnections()
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &objectKey,
		Body:   bytes.NewReader(objectData),
		Metadata: map[string]string{"Content-Length": fmt.Sprintf(
			"%d", len(objectData))},
	})
	return err
}

// Download downloads an object from S3 bucket.
func (s *s3Service) Download(ctx context.Context, bucketName, objectKey string) ([]byte, error) {
	//defer s.httpCli.CloseIdleConnections()
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &objectKey,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (s *s3Service) ListObjects(ctx context.Context, bucketName, prefix string) ([]string, error) {
	var ptrPrefix *string
	if prefix != "" {
		ptrPrefix = &prefix
	}
	resp, err := s.client.ListObjects(ctx, &s3.ListObjectsInput{
		Bucket: &bucketName,
		Prefix: ptrPrefix,
	})
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, obj := range resp.Contents {
		keys = append(keys, *obj.Key)
	}
	return keys, nil
}

func (s *s3Service) ListPrefix(ctx context.Context, bucketName, prefix string) ([]string, error) {
	var ptrPrefix *string
	if prefix != "" {
		ptrPrefix = &prefix
	}
	resp, err := s.client.ListObjects(ctx, &s3.ListObjectsInput{
		Bucket:    &bucketName,
		Prefix:    ptrPrefix,
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, obj := range resp.CommonPrefixes {
		keys = append(keys, *obj.Prefix)
	}
	return keys, nil
}

func (s *s3Service) DeleteObject(ctx context.Context, bucketName, objectKey string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &bucketName,
		Key:    &objectKey,
	})
	return err
}

func (s *s3Service) ListBuckets(ctx context.Context) ([]string, error) {
	resp, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	var buckets []string
	for _, b := range resp.Buckets {
		buckets = append(buckets, *b.Name)
	}
	return buckets, nil
}

func (s *s3Service) DeleteBucket(ctx context.Context, bucketName string) error {
	_, err := s.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: &bucketName,
	})
	return err
}

func getBucketKeyFromS3(address string) (bucket, object string, err error) {
	urlObj, err := url.Parse(address)
	if err != nil {
		return "", "", err
	}
	if len(urlObj.Path) == 0 {
		return "", "", fmt.Errorf("path is empty")
	}
	dir, filename := path.Split(urlObj.Path)
	if len(dir) == 0 {
		return "", "", fmt.Errorf("path is invalid")
	}
	if len(filename) == 0 {
		return "", "", fmt.Errorf("path is invalid")
	}
	return strings.Trim(dir, "/"), filename, nil

}
