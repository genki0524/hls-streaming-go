package ffmpeg

import (
	"log"
	"os"
	"os/exec"
)

func mp4tohls(inputPath string, outputPath string) {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-c:v", "copy", "-c:a", "copy", "-f", "hls", 
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

	log.Println("Running ffmpeg version check...")

	// コマンドを実行し、完了するまで待機する
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Command failed to run: %v", err)
	}

	log.Println("Command finished successfully.")
}
