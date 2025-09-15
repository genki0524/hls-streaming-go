package media

import (
	"log"
	"os"
	"os/exec"
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