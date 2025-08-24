package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func getProgramByGlobalSegment(globalSegmentIndex int, schedule []ProgramItem) (ProgramItem, int) {
	currentGlobalIndex := 0
	for _, program := range schedule {
		programSegments := math.Ceil(float64(program.DurationSec) / float64(SEGMENT_DURATION))

		if currentGlobalIndex <= globalSegmentIndex && globalSegmentIndex < currentGlobalIndex+int(programSegments) {
			programSegmentIndex := globalSegmentIndex - currentGlobalIndex
			return program, programSegmentIndex
		}
		currentGlobalIndex += int(programSegments)
	}
	return ProgramItem{}, 0
}

func getStataicImagePlaylist(w http.ResponseWriter, r *http.Request, schedule []ProgramItem) {
	jst := time.FixedZone("JST", 9*60*60)
	now := time.Now().In(jst)

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

	if nextProgramStart != nil {
		timeUntilNext := nextProgramStart.Sub(now).Seconds()
		segmentDuration = math.Min(math.Max(5, timeUntilNext), 30)
		segmentCount = int(math.Max(1, timeUntilNext/segmentDuration))
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

	//番組が始まってからの経過時間
	timeInfoPragram := now.Sub(programStartTime).Seconds()
	//番組が始まってから経過したセグメントの数
	currentSegmentIndex := int(timeInfoPragram / SEGMENT_DURATION)

	// 番組の終了時間をチェック
	programEndTime := programStartTime.Add(time.Duration(currentProgram.DurationSec) * time.Second)
	if now.After(programEndTime) {
		getStataicImagePlaylist(w, r, schedule)
		return
	}

	totalSegmentsBefore := 0
	for _, program := range schedule {
		if program.StartTime == currentProgram.StartTime {
			log.Printf("現在の番組を発見: %s", program.StartTime)
			break
		}
		segmentCount := int(math.Ceil(float64(program.DurationSec) / SEGMENT_DURATION))
		totalSegmentsBefore += segmentCount
		log.Printf("スキップした番組: %s, セグメント数: %d", program.StartTime, segmentCount)
	}

	//全体での経過したセグメントの数
	globalCurrentSegment := totalSegmentsBefore + currentSegmentIndex

	// 現在の番組の総セグメント数
	currentProgramTotalSegments := int(math.Ceil(float64(currentProgram.DurationSec) / SEGMENT_DURATION))

	//プレイリストに含める最初のセグメントのインデックスを取得
	// 番組内のセグメントのみを含むように制限
	startGlobalIndex := int(math.Max(float64(totalSegmentsBefore), float64(globalCurrentSegment-PLAYLIST_LENGTH+1)))
	endGlobalIndex := int(math.Min(float64(globalCurrentSegment), float64(totalSegmentsBefore+currentProgramTotalSegments-1)))

	var m3u8Content []string
	m3u8Content = append(m3u8Content, "#EXTM3U")
	m3u8Content = append(m3u8Content, "#EXT-X-VERSION:3")
	m3u8Content = append(m3u8Content, "#EXT-X-TARGETDURATION:"+strconv.Itoa(int(SEGMENT_DURATION)))
	m3u8Content = append(m3u8Content, "#EXT-X-MEDIA-SEQUENCE:"+strconv.Itoa(startGlobalIndex))
	m3u8Content = append(m3u8Content, "#EXT-X-ALLOW-CACHE:YES")

	var lastProgram *ProgramItem
	for globalIndex := startGlobalIndex; globalIndex <= endGlobalIndex; globalIndex++ {
		segmentProgram, programSegmentIndex := getProgramByGlobalSegment(globalIndex, schedule)

		if segmentProgram == (ProgramItem{}) {
			continue
		}

		// 現在の番組以外のセグメントは含めない
		if segmentProgram.StartTime != currentProgram.StartTime {
			continue
		}

		if lastProgram != nil && *lastProgram != segmentProgram {
			m3u8Content = append(m3u8Content, "#EXT-X-DISCONTINUITY")
		}

		m3u8Content = append(m3u8Content, fmt.Sprintf("#EXTINF:%.1f,", SEGMENT_DURATION))

		// セグメントファイル名を生成（3桁の0埋め）
		segmentFilename := strings.Replace(segmentProgram.PathTemplate, "{}", fmt.Sprintf("%03d", programSegmentIndex), 1)
		absoluteURL := "/" + segmentFilename
		m3u8Content = append(m3u8Content, absoluteURL)

		lastProgram = &segmentProgram
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
