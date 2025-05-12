// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

package apply

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/worldiety/nago-runner/configuration"
	"io"
	"time"
)

type S3Open func(s3 configuration.S3, name string) (io.ReadCloser, error)

func NewS3Open() S3Open {
	return func(s3 configuration.S3, name string) (io.ReadCloser, error) {
		// not sure how inefficient it is to create always new client instances
		client, err := minio.New(s3.Endpoint, &minio.Options{
			Secure: true,
			Creds: credentials.NewStaticV4(
				s3.AccessKey,
				s3.SecretKey,
				"",
			),
		})
		if err != nil {
			return nil, fmt.Errorf("error creating S3 client: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
		defer cancel()

		obj, err := client.GetObject(ctx, s3.Bucket, name, minio.GetObjectOptions{})
		if err != nil {
			return nil, fmt.Errorf("error getting object: %v", err)
		}

		return obj, nil
	}
}

func OpenByHash(open S3Open, s3 configuration.S3, hash configuration.Sha3V512) (io.ReadCloser, error) {
	return open(s3, fmt.Sprintf("bin/%s", hash))
}
