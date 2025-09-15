package repository

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"github.com/genki0524/hls_striming_go/internal/domain"
)

type GCSRepository struct {
	client *storage.Client
}

func NewGCSRepository(client *storage.Client) *GCSRepository {
	return &GCSRepository{
		client: client,
	}
}

func (r *GCSRepository) UploadFile(ctx context.Context, bucket, object, filePath string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	o := r.client.Bucket(bucket).Object(object)
	o = o.If(storage.Conditions{DoesNotExist: true})

	wc := o.NewWriter(ctx)
	defer wc.Close()

	return nil
}

func (r *GCSRepository) DownloadFileToMemory(ctx context.Context, bucket, object string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	rc, err := r.client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %w", object, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}
	return data, nil
}

func (r *GCSRepository) CreateSignedURL(bucket, object string) (string, error) {
	expiresTime := 3 * time.Minute

	u, err := r.client.Bucket(bucket).SignedURL(object, &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  http.MethodGet,
		Expires: time.Now().Add(expiresTime),
	})
	if err != nil {
		return "", fmt.Errorf("Bucket(%q).SignedURL: %w", bucket, err)
	}
	return u, nil
}

func (r *GCSRepository) GetM3U8WithSignedURLs(bucket, date, programName string) (*domain.M3U8Playlist, error) {
	resourcePath := date + "/" + programName
	m3u8Data, err := r.DownloadFileToMemory(context.Background(), bucket, resourcePath+"/video.m3u8")
	if err != nil {
		return nil, fmt.Errorf("downloadFileIntoMemory: %w", err)
	}

	playlist, err := domain.ParseM3U8Content(string(m3u8Data))
	if err != nil {
		return nil, fmt.Errorf("ParseM3U8Content: %w", err)
	}

	for index, segment := range playlist.Segments {
		fileName := segment.Filename
		url, err := r.CreateSignedURL(bucket, resourcePath+"/"+fileName)
		if err != nil {
			return nil, fmt.Errorf("createSignedURL: %w", err)
		}
		playlist.Segments[index].Filename = url
	}

	return playlist, nil
}