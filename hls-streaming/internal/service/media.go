package service

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/genki0524/hls_striming_go/internal/media"
	"github.com/genki0524/hls_striming_go/internal/repository"
)

type MediaService struct {
	gcsRepo       *repository.GCSRepository
	ffmpegService *media.FFmpegService
}

func NewMediaService(gcsRepo *repository.GCSRepository, ffmpegService *media.FFmpegService) *MediaService {
	return &MediaService{
		gcsRepo:       gcsRepo,
		ffmpegService: ffmpegService,
	}
}
func (s *MediaService) UploadVideo(ctx context.Context, object string, data []byte) error {
	bucket := os.Getenv("BUCKET")
	if err := s.gcsRepo.UploadVideoData(ctx, bucket, object, data); err != nil {
		return fmt.Errorf("動画のアップロードでエラーが発生しました")
	}
	return nil
}

func (s *MediaService) ConvertAndUploadHLS(ctx context.Context, videoData []byte, date, programName string) error {
	bucket := os.Getenv("BUCKET")
	if bucket == "" {
		return fmt.Errorf("BUCKET環境変数が設定されていません")
	}

	// 一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "hls_conversion_")
	if err != nil {
		return fmt.Errorf("一時ディレクトリ作成エラー: %w", err)
	}
	defer os.RemoveAll(tempDir) // 処理完了後にクリーンアップ

	if err := s.ffmpegService.ConvertByteDataToHLS(videoData, tempDir); err != nil {
		return fmt.Errorf("HLS変換エラー: %w", err)
	}

	// 変換されたファイルをGCSにアップロード
	basePath := fmt.Sprintf("%s/%s", date, programName)

	// m3u8ファイルをアップロード
	m3u8Path := filepath.Join(tempDir, "video.m3u8")
	m3u8Data, err := os.ReadFile(m3u8Path)
	if err != nil {
		return fmt.Errorf("m3u8ファイル読み込みエラー: %w", err)
	}

	m3u8Object := basePath + "/video.m3u8"
	if err := s.gcsRepo.UploadVideoData(ctx, bucket, m3u8Object, m3u8Data); err != nil {
		return fmt.Errorf("m3u8ファイルアップロードエラー: %w", err)
	}

	// tsファイルをアップロード
	err = filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// .tsファイルのみを処理
		if !d.IsDir() && filepath.Ext(path) == ".ts" {
			tsData, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("tsファイル読み込みエラー (%s): %w", path, err)
			}

			fileName := d.Name()
			tsObject := basePath + "/" + fileName
			if err := s.gcsRepo.UploadVideoData(ctx, bucket, tsObject, tsData); err != nil {
				return fmt.Errorf("tsファイルアップロードエラー (%s): %w", fileName, err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("tsファイル処理エラー: %w", err)
	}

	return nil
}
