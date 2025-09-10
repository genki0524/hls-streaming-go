package utils

import (
	"testing"
)

func TestFfmpeg(t *testing.T) {
	inputPath := "../static/stream/sushida_1/sushida_1_cutted.mp4"
	outputPath := "../static/stream/sushida_1/"
	mp4tohls(inputPath, outputPath)
}
