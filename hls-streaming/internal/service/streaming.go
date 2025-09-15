package service

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/genki0524/hls_striming_go/internal/domain"
	"github.com/genki0524/hls_striming_go/internal/repository"
	"github.com/joho/godotenv"
)

type StreamingService struct {
	gcsRepo *repository.GCSRepository
}

func NewStreamingService(gcsRepo *repository.GCSRepository) *StreamingService {
	return &StreamingService{
		gcsRepo: gcsRepo,
	}
}

func (s *StreamingService) GenerateStaticImagePlaylist(schedule []domain.ProgramItem) string {
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Now().In(jst)

	nextProgram := domain.FindNextProgram(schedule, now, jst)

	var segmentDuration float64
	var segmentCount int

	if nextProgram != nil {
		nextStartTime, err := nextProgram.GetStartTime()
		if err == nil {
			nextStartTimeJST := nextStartTime.In(jst)
			timeUntilNext := nextStartTimeJST.Sub(now).Seconds()
			segmentDuration = math.Min(math.Max(5, timeUntilNext), 30)
			segmentCount = int(math.Max(1, timeUntilNext/segmentDuration))
		} else {
			segmentDuration = 30
			segmentCount = 10
		}
	} else {
		segmentDuration = 30
		segmentCount = 10
	}

	var m3u8Content []string
	m3u8Content = append(m3u8Content, "#EXTM3U")
	m3u8Content = append(m3u8Content, "#EXT-X-VERSION:3")
	m3u8Content = append(m3u8Content, "#EXT-X-TARGETDURATION:"+strconv.Itoa(int(segmentDuration)+1))
	m3u8Content = append(m3u8Content, "#EXT-X-MEDIA-SEQUENCE:0")
	m3u8Content = append(m3u8Content, "#EXT-X-ALLOW-CACHE:NO")

	maxSegments := int(math.Min(float64(segmentCount), 20))
	for i := 0; i < maxSegments; i++ {
		m3u8Content = append(m3u8Content, fmt.Sprintf("#EXTINF:%.1f,", segmentDuration))
		m3u8Content = append(m3u8Content, "/static/images/picture.jpg")
	}

	return strings.Join(m3u8Content, "\n") + "\n"
}

func (s *StreamingService) GenerateVODPlaylist(schedule []domain.ProgramItem) (string, error) {
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Now().In(jst)

	currentProgram, currentProgramIndex := domain.FindCurrentProgram(schedule, now, jst)
	if currentProgram == nil {
		return s.GenerateStaticImagePlaylist(schedule), nil
	}

	if err := godotenv.Load(".env"); err != nil {
		return "", fmt.Errorf("環境変数の読み込みに失敗: %w", err)
	}

	bucket := os.Getenv("BUCKET")
	todayString := "2025-09-09" // TODO: 実際の日付を使用
	programName := currentProgram.Title

	playlist, err := s.gcsRepo.GetM3U8WithSignedURLs(bucket, todayString, programName)
	if err != nil {
		log.Printf("m3u8ファイルの読み込みに失敗: %v", err)
		return s.GenerateStaticImagePlaylist(schedule), nil
	}

	log.Printf("読み込んだセグメント数: %d", len(playlist.Segments))

	programStartTime, err := currentProgram.GetStartTime()
	if err != nil {
		return "", err
	}
	programStartTimeJST := programStartTime.In(jst)
	timeIntoProgram := now.Sub(programStartTimeJST).Seconds()

	currentSegmentIndex := playlist.GetCurrentSegmentIndex(timeIntoProgram)
	startIndex, endIndex := playlist.GetSegmentRange(currentSegmentIndex)

	var m3u8Content []string
	m3u8Content = append(m3u8Content, "#EXTM3U")
	m3u8Content = append(m3u8Content, "#EXT-X-VERSION:3")
	m3u8Content = append(m3u8Content, "#EXT-X-TARGETDURATION:"+strconv.Itoa(playlist.TargetDuration))
	m3u8Content = append(m3u8Content, "#EXT-X-MEDIA-SEQUENCE:"+strconv.Itoa(startIndex))
	m3u8Content = append(m3u8Content, "#EXT-X-ALLOW-CACHE:YES")

	for i := startIndex; i <= endIndex && i < len(playlist.Segments); i++ {
		segment := playlist.Segments[i]
		m3u8Content = append(m3u8Content, fmt.Sprintf("#EXTINF:%.1f,", segment.Duration))
		m3u8Content = append(m3u8Content, segment.Filename)
	}

	if endIndex == len(playlist.Segments)-1 && (endIndex+1)-startIndex != domain.PlaylistLength {
		if currentProgramIndex >= 0 && currentProgramIndex+1 < len(schedule) {
			nextProgram := &schedule[currentProgramIndex+1]
			s.appendNextProgramSegments(&m3u8Content, nextProgram, bucket, todayString, startIndex, endIndex)
		}
	}

	return strings.Join(m3u8Content, "\n") + "\n", nil
}

func (s *StreamingService) appendNextProgramSegments(m3u8Content *[]string, nextProgram *domain.ProgramItem, bucket, todayString string, startIndex, endIndex int) {
	*m3u8Content = append(*m3u8Content, "#EXT-X-DISCONTINUITY")

	neededSegments := domain.PlaylistLength - ((endIndex + 1) - startIndex)

	nextPlaylist, err := s.gcsRepo.GetM3U8WithSignedURLs(bucket, todayString, nextProgram.Title)
	if err != nil {
		log.Printf("次の番組のm3u8ファイルの読み込みに失敗: %v", err)
		return
	}

	for i := 0; i < neededSegments && i < len(nextPlaylist.Segments); i++ {
		nextSegment := nextPlaylist.Segments[i]
		*m3u8Content = append(*m3u8Content, fmt.Sprintf("#EXTINF:%.1f,", nextSegment.Duration))
		*m3u8Content = append(*m3u8Content, nextSegment.Filename)
	}
}

func (s *StreamingService) CheckStreamStatus(schedule []domain.ProgramItem) int {
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Now().In(jst)

	currentProgram, _ := domain.FindCurrentProgram(schedule, now, jst)
	if currentProgram != nil {
		return http.StatusOK
	}

	return http.StatusNoContent
}