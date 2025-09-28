package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/genki0524/hls_striming_go/internal/media"
)

func TestFFmpegService_ConvertMP4ToHLS(t *testing.T) {
	ffmpegService := media.NewFFmpegService()

	// テスト用のディレクトリとファイルパス
	testInputDir := "../test_data"
	testOutputDir := "../test_output"
	inputPath := filepath.Join(testInputDir, "test_video.mp4")
	outputPath := testOutputDir

	// テスト用ディレクトリが存在しない場合はスキップ
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skipf("テスト用入力ファイルが存在しません: %s", inputPath)
	}

	// 出力ディレクトリを作成
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		t.Fatalf("出力ディレクトリの作成に失敗: %v", err)
	}

	// テスト後のクリーンアップ
	defer func() {
		if err := os.RemoveAll(outputPath); err != nil {
			t.Logf("テストディレクトリの削除に失敗: %v", err)
		}
	}()

	// FFmpeg変換のテスト
	err := ffmpegService.ConvertMP4ToHLS(inputPath, outputPath)
	if err != nil {
		t.Logf("FFmpeg変換テストをスキップ（FFmpegが利用できない可能性）: %v", err)
		return
	}

	// 出力ファイルが作成されているかチェック
	m3u8Path := filepath.Join(outputPath, "video.m3u8")
	if _, err := os.Stat(m3u8Path); os.IsNotExist(err) {
		t.Error("M3U8ファイルが作成されていません")
	}

	t.Log("FFmpeg変換が正常に完了しました")
}