package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/genki0524/hls_striming_go/crud"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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

// スライスのインデックスが有効かどうかをチェックするジェネリクス関数
func isValidIndex[T any](slice []T, index int) bool {
	return index >= 0 && index < len(slice)
}

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

/*
番組がない時間の静止画像を配信する関数
*/
func getStataicImagePlaylist(w http.ResponseWriter, r *http.Request, schedule []ProgramItem) {
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Now().In(jst)

	//次の番組の開始時間を取得、存在しない場合はnil
	//現在時刻から次の番組の開始時間を取得
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

	//次の番組までの残り時間から1セグメントあたりの時間とセグメント数を取得
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
	var nextProgram *ProgramItem
	currentProgramIndex := -1

	//現在放送されている番組を取得
	for index, program := range schedule {
		startTime, err := time.Parse(time.RFC3339, program.StartTime)
		if err != nil {
			continue
		}

		startTimeJST := startTime.In(jst)
		endTime := startTimeJST.Add(time.Duration(program.DurationSec) * time.Second)

		if startTimeJST.Before(now) && now.Before(endTime) {
			currentProgram = &program
			programStartTime = startTimeJST
			currentProgramIndex = index
			break
		}
	}

	if currentProgramIndex >= 0 && isValidIndex(schedule, currentProgramIndex+1) {
		nextProgram = &schedule[currentProgramIndex+1]
	}

	//番組がない場合は静止画を表示
	if currentProgram == nil {
		getStataicImagePlaylist(w, r, schedule)
		return
	}

	var m3u8Content []string

	// 番組が始まってからの経過時間
	timeInfoProgram := now.Sub(programStartTime).Seconds()

	if err := godotenv.Load(".env"); err != nil {
		fmt.Printf("読み込みができませんでした: %v", err)
	}

	bucket := os.Getenv("BUCKET")
	// todayString := time.Now().In(jst).Format("2006-01-02")
	todayString := "2025-09-09"
	programName := currentProgram.Title

	playlist, err := crud.CreateSignedM3u8(bucket, todayString, programName)

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
		m3u8Content = append(m3u8Content, segment.Filename)
	}

	if endIndex == len(playlist.Segments)-1 && (endIndex+1)-startIndex != PLAYLIST_LENGTH && nextProgram != nil {
		// 不連続性を示すタグを追加
		m3u8Content = append(m3u8Content, "#EXT-X-DISCONTINUITY")

		// 次の番組のセグメントを追加してPLAYLIST_LENGTHに合わせる
		neededSegments := PLAYLIST_LENGTH - ((endIndex + 1) - startIndex)

		// 次の番組のディレクトリパスを取得
		// nextProgramDir := filepath.Dir(nextProgram.PathTemplate)
		// nextM3u8FilePath := filepath.Join(nextProgramDir, "video.m3u8")

		// 次の番組のm3u8ファイルを読み込み
		// nextPlaylist, err := parseM3U8File(nextM3u8FilePath)
		nextPlaylist, err := crud.CreateSignedM3u8(bucket, todayString, nextProgram.Title)

		if err != nil {
			log.Printf("次の番組のm3u8ファイルの読み込みに失敗: %v", err)
		} else {
			// 必要な数だけ次の番組のセグメントを追加
			for i := 0; i < neededSegments && i < len(nextPlaylist.Segments); i++ {
				nextSegment := nextPlaylist.Segments[i]
				m3u8Content = append(m3u8Content, fmt.Sprintf("#EXTINF:%.1f,", nextSegment.Duration))

				// 次の番組のセグメントパスを絶対URLに変換
				// nextSegmentPath := filepath.Join(nextProgramDir, nextSegment.Filename)
				// nextAbsoluteURL := "/" + nextSegmentPath
				m3u8Content = append(m3u8Content, nextSegment.Filename)
			}
		}
	}

	finalContent := strings.Join(m3u8Content, "\n") + "\n"

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
