package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type M3U8Segment struct {
	Duration float64
	Filename string
}

type M3U8Playlist struct {
	Version        int
	TargetDuration int
	MediaSequence  int
	PlaylistType   string
	AllowCache     string
	Segments       []M3U8Segment
}

var globalProgram ProgramItem

// m3u8ファイルを読み込んで解析する
func parseM3U8File(filePath string) (*M3U8Playlist, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	playlist := &M3U8Playlist{}
	scanner := bufio.NewScanner(file)

	var currentDuration float64

	//m3u8ファイルからヘッダ情報とセグメント情報を取得
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "#EXT-X-VERSION:") {
			version, _ := strconv.Atoi(strings.TrimPrefix(line, "#EXT-X-VERSION:"))
			playlist.Version = version
		} else if strings.HasPrefix(line, "#EXT-X-TARGETDURATION:") {
			duration, _ := strconv.Atoi(strings.TrimPrefix(line, "#EXT-X-TARGETDURATION:"))
			playlist.TargetDuration = duration
		} else if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
			sequence, _ := strconv.Atoi(strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:"))
			playlist.MediaSequence = sequence
		} else if strings.HasPrefix(line, "#EXT-X-PLAYLIST-TYPE:") {
			playlist.PlaylistType = strings.TrimPrefix(line, "#EXT-X-PLAYLIST-TYPE:")
		} else if strings.HasPrefix(line, "#EXT-X-ALLOW-CACHE:") {
			playlist.AllowCache = strings.TrimPrefix(line, "#EXT-X-ALLOW-CACHE:")
		} else if strings.HasPrefix(line, "#EXTINF:") {
			// #EXTINF:9.000000, の形式から時間を抽出
			parts := strings.Split(strings.TrimPrefix(line, "#EXTINF:"), ",")
			if len(parts) > 0 {
				duration, _ := strconv.ParseFloat(parts[0], 64)
				currentDuration = duration
			}
		} else if line != "" && !strings.HasPrefix(line, "#") {
			// セグメントファイル名
			segment := M3U8Segment{
				Duration: currentDuration,
				Filename: line,
			}
			playlist.Segments = append(playlist.Segments, segment)
			currentDuration = 0
		}
	}

	return playlist, scanner.Err()
}

func getStataicImagePlaylist(w http.ResponseWriter, r *http.Request, schedule []ProgramItem) {
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Now().In(jst)

	//次の番組の開始時間を取得、存在しない場合はnil
	var nextProgramStart *time.Time
	for _, program := range schedule {
		startTime, err := time.Parse(time.RFC3339, program.StartTime)
		if err != nil {
			continue
		}

		startTimeJST := startTime.In(jst)
		if startTimeJST.After(now) {
			nextProgramStart = &startTimeJST
			break
		}
	}

	var segmentDuration float64
	var segmentCount int

	//静止画を流す時間を計測
	if nextProgramStart != nil {
		//次の番組がある場合は、残り時間から取得
		timeUntilNext := nextProgramStart.Sub(now).Seconds()
		segmentDuration = math.Min(math.Max(5, timeUntilNext), 30)
		segmentCount = int(math.Max(1, timeUntilNext/segmentDuration))
	} else {
		//次の番組がない場合は固定
		segmentDuration = 30
		segmentCount = 10
	}

	var m3u8Content []string
	m3u8Content = append(m3u8Content, "#EXTM3U")
	m3u8Content = append(m3u8Content, "#EXT-X-VERSION:3")
	m3u8Content = append(m3u8Content, "#EXT-X-TARGETDURATION:"+strconv.Itoa(int(segmentDuration)+1))
	m3u8Content = append(m3u8Content, "#EXT-X-MEDIA-SEQUENCE:0")
	m3u8Content = append(m3u8Content, "#EXT-X-ALLOW-CACHE:NO")

	maxSegments := int(math.Min(float64(segmentCount), 20)) // Maximum 20 segments
	for i := 0; i < maxSegments; i++ {
		m3u8Content = append(m3u8Content, fmt.Sprintf("#EXTINF:%.1f,", segmentDuration))
		m3u8Content = append(m3u8Content, "/static/images/picture.jpg")
	}

	finalContent := strings.Join(m3u8Content, "\n") + "\n"

	fmt.Printf("HLSプレイリスト :%v", finalContent)

	// Set response headers
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(finalContent))
}

func getVodPlaylist(w http.ResponseWriter, r *http.Request, schedule []ProgramItem) {
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Now().In(jst)

	var currentProgram *ProgramItem
	var programStartTime time.Time

	//現在放送されている番組を取得
	for _, program := range schedule {
		startTime, err := time.Parse(time.RFC3339, program.StartTime)
		if err != nil {
			continue
		}

		startTimeJST := startTime.In(jst)
		endTime := startTimeJST.Add(time.Duration(program.DurationSec) * time.Second)

		if startTimeJST.Before(now) && now.Before(endTime) {
			currentProgram = &program
			programStartTime = startTimeJST
			break
		}
	}

	//番組がない場合は静止画を表示
	if currentProgram == nil {
		getStataicImagePlaylist(w, r, schedule)
		return
	}

	var m3u8Content []string

	if globalProgram != *currentProgram {
		globalProgram = *currentProgram
		m3u8Content = append(m3u8Content, "#EXT-X-DISCONTINUITY")
	}

	// 番組が始まってからの経過時間
	timeInfoProgram := now.Sub(programStartTime).Seconds()

	// PathTemplateからディレクトリパスを抽出 (例: static/stream/hoooope-2025-01-15-copy/video{}.ts -> static/stream/hoooope-2025-01-15-copy/)
	programDir := filepath.Dir(currentProgram.PathTemplate)
	m3u8FilePath := filepath.Join(programDir, "video.m3u8")

	log.Printf("m3u8ファイルパス: %s", m3u8FilePath)

	// m3u8ファイルを読み込み
	playlist, err := parseM3U8File(m3u8FilePath)
	if err != nil {
		log.Printf("m3u8ファイルの読み込みに失敗: %v", err)
		getStataicImagePlaylist(w, r, schedule)
		return
	}

	log.Printf("読み込んだセグメント数: %d", len(playlist.Segments))

	// セグメントの長さを使って現在のセグメントインデックスを計算
	var accumulatedTime float64 = 0
	var currentSegmentIndex int = 0

	for i, segment := range playlist.Segments {
		if accumulatedTime+segment.Duration > timeInfoProgram {
			currentSegmentIndex = i
			break
		}
		accumulatedTime += segment.Duration
		currentSegmentIndex = i + 1
	}

	// プレイリストに含めるセグメントの範囲を決定
	startIndex := int(math.Max(0, float64(currentSegmentIndex-PLAYLIST_LENGTH+1)))
	endIndex := int(math.Min(float64(currentSegmentIndex), float64(len(playlist.Segments)-1)))

	// HLSプレイリストを生成
	m3u8Content = append(m3u8Content, "#EXTM3U")
	m3u8Content = append(m3u8Content, "#EXT-X-VERSION:3")
	m3u8Content = append(m3u8Content, "#EXT-X-TARGETDURATION:"+strconv.Itoa(playlist.TargetDuration))
	m3u8Content = append(m3u8Content, "#EXT-X-MEDIA-SEQUENCE:"+strconv.Itoa(startIndex))
	m3u8Content = append(m3u8Content, "#EXT-X-ALLOW-CACHE:YES")

	for i := startIndex; i <= endIndex && i < len(playlist.Segments); i++ {
		segment := playlist.Segments[i]
		m3u8Content = append(m3u8Content, fmt.Sprintf("#EXTINF:%.1f,", segment.Duration))

		// セグメントのパスを絶対URLに変換
		segmentPath := filepath.Join(programDir, segment.Filename)
		absoluteURL := "/" + segmentPath
		m3u8Content = append(m3u8Content, absoluteURL)
	}

	finalContent := strings.Join(m3u8Content, "\n") + "\n"

	fmt.Printf("HLSプレイリスト :%v", finalContent)

	// Set response headers
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(finalContent))
}

func getStreamStatus(c *gin.Context, schedule []ProgramItem) {
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Now().In(jst)

	// 現在の番組があるかチェック
	for i := 0; i < len(schedule); i++ {
		program := &schedule[i]
		programStartTime, err := time.Parse(time.RFC3339, program.StartTime)
		if err != nil {
			continue
		}

		programStartTimeJST := programStartTime.In(jst)
		programEndTime := programStartTimeJST.Add(time.Duration(program.DurationSec) * time.Second)

		if now.After(programStartTimeJST) && now.Before(programEndTime) {
			// 番組がある場合は200を返す
			c.Status(http.StatusOK)
			return
		}
	}

	// 番組がない場合は204 No Contentを返す
	c.Status(http.StatusNoContent)
}
