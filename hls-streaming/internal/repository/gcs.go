package repository

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
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
	ctx, cancel := context.WithTimeout(ctx, time.Second*300) // 大きなファイル用に5分
	defer cancel()

	// ローカルファイルを開く
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ファイルオープンエラー: %w", err)
	}
	defer file.Close()

	// GCSオブジェクトライターを作成
	obj := r.client.Bucket(bucket).Object(object)
	writer := obj.NewWriter(ctx)
	defer writer.Close()

	// ファイルの内容をGCSにコピー
	if _, err := io.Copy(writer, file); err != nil {
		return fmt.Errorf("ファイルアップロードエラー: %w", err)
	}

	return nil
}

// UploadStream はio.Readerから動画データを受け取ってGCSにアップロードします
func (r *GCSRepository) UploadStream(ctx context.Context, bucket, object string, reader io.Reader) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*300) // 大きなファイル用に5分
	defer cancel()

	// GCSオブジェクトライターを作成
	obj := r.client.Bucket(bucket).Object(object)
	writer := obj.NewWriter(ctx)
	defer writer.Close()

	// ストリームの内容をGCSにコピー
	if _, err := io.Copy(writer, reader); err != nil {
		return fmt.Errorf("ストリームアップロードエラー: %w", err)
	}

	return nil
}

// UploadVideoData はバイト配列の動画データをGCSにアップロードします
func (r *GCSRepository) UploadVideoData(ctx context.Context, bucket, object string, data []byte) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*300) // 大きなファイル用に5分
	defer cancel()

	// GCSオブジェクトライターを作成
	obj := r.client.Bucket(bucket).Object(object)
	writer := obj.NewWriter(ctx)
	defer writer.Close()

	// バイトデータを書き込み
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("動画データアップロードエラー: %w", err)
	}

	return nil
}

// UploadVideoWithMetadata はメタデータ付きで動画データをアップロードします
func (r *GCSRepository) UploadVideoWithMetadata(ctx context.Context, bucket, object string, reader io.Reader, contentType string, metadata map[string]string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*300) // 大きなファイル用に5分
	defer cancel()

	// GCSオブジェクトライターを作成
	obj := r.client.Bucket(bucket).Object(object)
	writer := obj.NewWriter(ctx)
	defer writer.Close()

	// メタデータを設定
	if contentType != "" {
		writer.ContentType = contentType
	}
	if metadata != nil {
		writer.Metadata = metadata
	}

	// ストリームの内容をGCSにコピー
	if _, err := io.Copy(writer, reader); err != nil {
		return fmt.Errorf("メタデータ付きアップロードエラー: %w", err)
	}

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

func (r *GCSRepository) GetM3U8WithSignedURLs(ctx context.Context, bucket, date, programName string) (*domain.M3U8Playlist, error) {
	resourcePath := date + "/" + programName
	m3u8Data, err := r.DownloadFileToMemory(ctx, bucket, resourcePath+"/video.m3u8")
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

func (r *GCSRepository) ObjectExists(ctx context.Context, bucket, object string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10) // 存在チェックは短時間で
	defer cancel()

	obj := r.client.Bucket(bucket).Object(object)
	_, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}
		return false, fmt.Errorf("オブジェクト存在チェックエラー: %w", err)
	}
	return true, nil
}

func (r *GCSRepository) DeleteObject(ctx context.Context, bucket, object string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	obj := r.client.Bucket(bucket).Object(object)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("オブジェクト削除エラー: %w", err)
	}
	return nil
}
