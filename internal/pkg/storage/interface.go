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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type Storage interface {
	Upload(ctx context.Context, bucketName, objectKey string, objectData []byte) error
	Download(ctx context.Context, bucketName, objectKey string) ([]byte, error)
	HeadObject(ctx context.Context, bucketName, objectKey string) (bool, error)
	HeadObjectFromAddress(ctx context.Context, address string) (bucket, object string, err error)
	HeadObjectFromNewURIs(ctx context.Context, address string, insertBucket, insertObject string, newURI []string) (string, error)
}

type Bucket interface {
	MkBucket(ctx context.Context, bucketName string) error
	DeleteObject(ctx context.Context, bucketName, objectKey string) error
	DeleteBucket(ctx context.Context, bucketName string) error
	ListObjects(ctx context.Context, bucketName, prefix string) ([]string, error)
	ListBuckets(ctx context.Context) ([]string, error)
	ListPrefix(ctx context.Context, bucketName, prefix string) ([]string, error)
}

type Config struct {
	Type string
	// Access key of S3 AWS.
	Key string
	// Access secret of S3 AWS.
	Secret string
	// Region.
	Region string
	// AWS endpoint.
	Endpoint string
	// Maximum backoff delay (ms, default: 20 sec).
	MaxBackoffDelay *int32
	// Maximum attempts to retry operation on error (default: 5).
	MaxRetryAttempts *int32
}

func NewStorage(cfg Config) (Storage, error) {
	if !strings.HasPrefix(cfg.Endpoint, "http") {
		cfg.Endpoint = "http://" + cfg.Endpoint
	}
	if cfg.Type == "" || cfg.Type == "s3" {
		return newS3Service(cfg)
	}
	return nil, fmt.Errorf("not support this s3 type: %s", cfg.Type)
}

func NewBucket(cfg Config) (Bucket, error) {
	if !strings.HasPrefix(cfg.Endpoint, "http") {
		cfg.Endpoint = "http://" + cfg.Endpoint
	}
	if cfg.Type == "" || cfg.Type == "s3" {
		return newS3BucketService(cfg)
	}
	return nil, fmt.Errorf("not support this s3 type: %s", cfg.Type)
}

func getHttpClient() *http.Client {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   90 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          2000,
		MaxIdleConnsPerHost:   2000,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   90 * time.Second,
		ExpectContinueTimeout: 90 * time.Second,
		ResponseHeaderTimeout: 90 * time.Second,
		// Set this value so that the underlying transport round-tripper
		// doesn't try to auto decode the body of objects with
		// content-encoding set to `gzip`.
		//
		// Refer:
		//    https://golang.org/src/net/http/transport.go?h=roundTrip#L1843
		DisableCompression: false,
		DisableKeepAlives:  false,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{
		Transport: tr,
		Timeout:   time.Second * 90,
	}
}
