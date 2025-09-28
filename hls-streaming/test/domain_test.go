package test

import (
	"testing"
	"time"

	"github.com/genki0524/hls_striming_go/internal/domain"
)

func TestProgramItem_IsCurrentlyAiring(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)

	// テスト用の番組
	program := domain.ProgramItem{
		StartTime:   "2025-09-15T10:00:00Z",
		DurationSec: 1800, // 30分
		Title:       "テスト番組",
	}

	// 番組開始前のテスト
	beforeStart := time.Date(2025, 9, 15, 18, 30, 0, 0, jst) // JST 18:30 (UTC 09:30)
	if program.IsCurrentlyAiring(beforeStart, jst) {
		t.Error("番組開始前なのに放送中と判定されています")
	}

	// 番組放送中のテスト
	duringAiring := time.Date(2025, 9, 15, 19, 15, 0, 0, jst) // JST 19:15 (UTC 10:15)
	if !program.IsCurrentlyAiring(duringAiring, jst) {
		t.Error("番組放送中なのに放送中でないと判定されています")
	}

	// 番組終了後のテスト
	afterEnd := time.Date(2025, 9, 15, 20, 0, 0, 0, jst) // JST 20:00 (UTC 11:00)
	if program.IsCurrentlyAiring(afterEnd, jst) {
		t.Error("番組終了後なのに放送中と判定されています")
	}
}

func TestFindCurrentProgram(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)

	schedule := []domain.ProgramItem{
		{
			StartTime:   "2025-09-15T09:00:00Z", // JST 18:00
			DurationSec: 1800,                   // 30分
			Title:       "番組1",
		},
		{
			StartTime:   "2025-09-15T09:30:00Z", // JST 18:30
			DurationSec: 1800,                   // 30分
			Title:       "番組2",
		},
		{
			StartTime:   "2025-09-15T10:00:00Z", // JST 19:00
			DurationSec: 1800,                   // 30分
			Title:       "番組3",
		},
	}

	// 番組2の放送時間中でテスト
	currentTime := time.Date(2025, 9, 15, 18, 45, 0, 0, jst) // JST 18:45

	currentProgram, index := domain.FindCurrentProgram(schedule, currentTime, jst)

	if currentProgram == nil {
		t.Fatal("現在の番組が見つかりませんでした")
	}

	if currentProgram.Title != "番組2" {
		t.Errorf("期待した番組: 番組2, 実際: %s", currentProgram.Title)
	}

	if index != 1 {
		t.Errorf("期待したインデックス: 1, 実際: %d", index)
	}
}

func TestFindNextProgram(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)

	schedule := []domain.ProgramItem{
		{
			StartTime:   "2025-09-15T09:00:00Z", // JST 18:00
			DurationSec: 1800,                   // 30分
			Title:       "番組1",
		},
		{
			StartTime:   "2025-09-15T10:00:00Z", // JST 19:00
			DurationSec: 1800,                   // 30分
			Title:       "番組2",
		},
	}

	// 番組1の放送時間中でテスト
	currentTime := time.Date(2025, 9, 15, 18, 15, 0, 0, jst) // JST 18:15

	nextProgram := domain.FindNextProgram(schedule, currentTime, jst)

	if nextProgram == nil {
		t.Fatal("次の番組が見つかりませんでした")
	}

	if nextProgram.Title != "番組2" {
		t.Errorf("期待した次の番組: 番組2, 実際: %s", nextProgram.Title)
	}
}

func TestParseM3U8Content(t *testing.T) {
	m3u8Data := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:VOD
#EXT-X-ALLOW-CACHE:YES
#EXTINF:9.0,
segment000.ts
#EXTINF:9.0,
segment001.ts
#EXTINF:5.0,
segment002.ts
#EXT-X-ENDLIST`

	playlist, err := domain.ParseM3U8Content(m3u8Data)
	if err != nil {
		t.Fatalf("M3U8パースエラー: %v", err)
	}

	if playlist.Version != 3 {
		t.Errorf("期待したバージョン: 3, 実際: %d", playlist.Version)
	}

	if playlist.TargetDuration != 10 {
		t.Errorf("期待したターゲット時間: 10, 実際: %d", playlist.TargetDuration)
	}

	if len(playlist.Segments) != 3 {
		t.Errorf("期待したセグメント数: 3, 実際: %d", len(playlist.Segments))
	}

	if playlist.Segments[0].Duration != 9.0 {
		t.Errorf("期待したセグメント時間: 9.0, 実際: %f", playlist.Segments[0].Duration)
	}

	if playlist.Segments[0].Filename != "segment000.ts" {
		t.Errorf("期待したファイル名: segment000.ts, 実際: %s", playlist.Segments[0].Filename)
	}
}