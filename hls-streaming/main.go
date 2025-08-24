package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type ProgramItem struct {
	StartTime    string `firestore:"start_time"`
	DurationSec  int32  `firestore:"duration_sec"`
	Type         string `firestore:"type"`
	PathTemplate string `firestore:"path_template"`
	Title        string `firestore:"title"`
}

type Schedule struct {
	Programs []ProgramItem `firestore:"programs"`
}

const SEGMENT_DURATION float64 = 12
const PLAYLIST_LENGTH int = 15

func main() {
	ctx := context.Background()
	client := initFireStore(ctx)
	defer client.Close()

	todayString := time.Now().Format("2006-01-02")

	docRef := client.Collection("schedules").Doc(todayString)

	doc, err := docRef.Get(ctx)
	var schedule []ProgramItem

	if err != nil {
		// Firestoreからの取得に失敗した場合、テスト用データを使用
		log.Printf("Firestoreからの取得に失敗: %v", err)
	} else {
		var data Schedule
		if err := doc.DataTo(&data); err != nil {
			log.Fatalf("データのマッピングに失敗しました: %v", err)
		}
		schedule = data.Programs
	}

	sort.Slice(schedule, func(i, j int) bool {
		return schedule[i].StartTime < schedule[j].StartTime
	})

	router := gin.Default()

	// HTMLファイルを配信するエンドポイント
	router.GET("/", func(c *gin.Context) {
		c.File("./index.html")
	})

	// ライブストリーム用のプレイリスト
	router.GET("/live/video.m3u8", func(c *gin.Context) {
		getVodPlaylist(c.Writer, c.Request, schedule)
	})

	// ストリーム状態確認用エンドポイント
	router.HEAD("/live/status", func(c *gin.Context) {
		getStreamStatus(c, schedule)
	})

	// 静的ファイル（画像など）を配信
	router.Static("/static", "./static")

	log.Println("サーバーを開始します: http://0.0.0.0:8000")
	router.Run("0.0.0.0:8000")
}

func initFireStore(ctx context.Context) *firestore.Client {
	err := godotenv.Load(".env")

	if err != nil {
		fmt.Printf("読み込みができませんでした: %v", err)
	}

	projectID := os.Getenv("PROJECT_ID")

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	return client
}
