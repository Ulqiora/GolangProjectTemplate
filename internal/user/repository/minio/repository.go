package minio

import "github.com/minio/minio-go/v7"

type Minio struct {
	db minio.Client
}
