package program

import 
(
	"strings"
	"bufio"
 	"strconv"
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

func ParseM3U8File(m3u8Data string) (*M3U8Playlist,error) {
	playlist := &M3U8Playlist{}

	var currentDuration float64

	scanner := bufio.NewScanner(strings.NewReader(m3u8Data))

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