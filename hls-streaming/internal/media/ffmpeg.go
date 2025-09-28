package media

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type FFmpegService struct{}

func NewFFmpegService() *FFmpegService {
	return &FFmpegService{}
}

func (f *FFmpegService) ConvertMP4ToHLS(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:v", "copy",
		"-c:a", "copy",
		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "0",
		"-hls_allow_cache", "1",
		"-hls_segment_type", "mpegts",
		"-hls_flags", "split_by_time",
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", outputPath+"/video%3d.ts",
		outputPath+"/video.m3u8")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("FFmpegコマンドを実行中: %s", cmd.String())

	err := cmd.Run()
	if err != nil {
		log.Printf("FFmpegコマンドの実行に失敗: %v", err)
		return err
	}

	log.Println("FFmpegコマンドが正常に完了しました")
	return nil
}

func (f *FFmpegService) ConvertByteDataToHLS(data []byte, outputPath string) error {
	// 一時ファイルを作成
	tempFile, err := os.CreateTemp("", "video_input_*.mp4")
	if err != nil {
		return fmt.Errorf("一時ファイル作成エラー: %w", err)
	}
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath) // 処理完了後に削除

	// バイトデータを一時ファイルに書き込み
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("一時ファイル書き込みエラー: %w", err)
	}
	tempFile.Close()

	log.Printf("一時ファイルに書き込み完了: %s (%d bytes)", tempFilePath, len(data))

	// FFmpegコマンドを実行（一時ファイルを入力として使用）
	cmd := exec.Command("ffmpeg",
		"-i", tempFilePath,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-preset", "fast",
		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "0",
		"-hls_allow_cache", "1",
		"-hls_segment_type", "mpegts",
		"-hls_flags", "split_by_time",
		"-hls_playlist_type", "vod",
		"-force_key_frames", "expr:gte(t,n_forced*2)",
		"-hls_segment_filename", filepath.Join(outputPath, "video%03d.ts"),
		filepath.Join(outputPath, "video.m3u8"))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("FFmpegコマンド（一時ファイル版）を実行中: %s", cmd.String())

	err = cmd.Run()
	if err != nil {
		log.Printf("FFmpegコマンドの実行に失敗: %v", err)
		return err
	}

	log.Println("FFmpegコマンド（一時ファイル版）が正常に完了しました")
	return nil
}
