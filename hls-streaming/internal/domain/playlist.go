package domain

import (
	"bufio"
	"strconv"
	"strings"
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

const (
	SegmentDuration float64 = 3
	PlaylistLength  int     = 15
)

func NewM3U8Playlist() *M3U8Playlist {
	return &M3U8Playlist{
		Segments: make([]M3U8Segment, 0),
	}
}

func ParseM3U8Content(m3u8Data string) (*M3U8Playlist, error) {
	playlist := NewM3U8Playlist()
	scanner := bufio.NewScanner(strings.NewReader(m3u8Data))

	var currentDuration float64

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
			parts := strings.Split(strings.TrimPrefix(line, "#EXTINF:"), ",")
			if len(parts) > 0 {
				duration, _ := strconv.ParseFloat(parts[0], 64)
				currentDuration = duration
			}
		} else if line != "" && !strings.HasPrefix(line, "#") {
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

func (p *M3U8Playlist) GetCurrentSegmentIndex(timeIntoProgram float64) int {
	var accumulatedTime float64 = 0
	var currentSegmentIndex int = 0

	for i, segment := range p.Segments {
		if accumulatedTime+segment.Duration > timeIntoProgram {
			currentSegmentIndex = i
			break
		}
		accumulatedTime += segment.Duration
		currentSegmentIndex = i + 1
	}

	return currentSegmentIndex
}

func (p *M3U8Playlist) GetSegmentRange(currentSegmentIndex int) (int, int) {
	startIndex := max(0, currentSegmentIndex-PlaylistLength+1)
	endIndex := min(currentSegmentIndex, len(p.Segments)-1)
	return startIndex, endIndex
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}