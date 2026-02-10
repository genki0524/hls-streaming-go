package repository

import (
	"context"
	"io"

	"github.com/genki0524/hls_striming_go/internal/domain"
)

type StorageRepository interface {
	UploadFile(ctx context.Context, bucket, object, filePath string) error
	UploadStream(ctx context.Context, bucket, object string, reader io.Reader) error
	UploadVideoData(ctx context.Context, bucket, object string, data []byte) error
	DownloadFile(ctx context.Context, bucket, object string) ([]byte, error)

	GetM3U8WithSignedURLs(ctx context.Context, bucket, date, programName string) (*domain.M3U8Playlist, error)
	GenerateSignedURL(bucket, object string, expiration int) (string, error)
}
