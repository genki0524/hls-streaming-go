package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
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

// グローバルな番組表管理用の構造体
type ScheduleManager struct {
	schedule []ProgramItem
	mutex    sync.RWMutex
	client   *firestore.Client
}

// 番組表を取得する（スレッドセーフ）
func (sm *ScheduleManager) GetSchedule() []ProgramItem {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	// スライスのコピーを返す
	result := make([]ProgramItem, len(sm.schedule))
	copy(result, sm.schedule)
	return result
}

// 番組表を更新する（スレッドセーフ）
func (sm *ScheduleManager) UpdateSchedule(newSchedule []ProgramItem) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.schedule = newSchedule
	log.Printf("番組表を更新しました。番組数: %d", len(newSchedule))
}

// Firestoreから番組表を取得して更新する
func (sm *ScheduleManager) RefreshFromFirestore(ctx context.Context) error {
	jst := time.FixedZone("JST", 9*60*60)
	todayString := time.Now().In(jst).Format("2006-01-02")
	docRef := sm.client.Collection("schedules").Doc(todayString)

	doc, err := docRef.Get(ctx)
	if err != nil {
		log.Printf("Firestoreからの取得に失敗: %v", err)
		return err
	}

	var data Schedule
	if err := doc.DataTo(&data); err != nil {
		log.Printf("データのマッピングに失敗: %v", err)
		return err
	}

	// 番組を開始時間順にソート
	sort.Slice(data.Programs, func(i, j int) bool {
		return data.Programs[i].StartTime < data.Programs[j].StartTime
	})

	sm.UpdateSchedule(data.Programs)
	return nil
}

// 定期的に番組表を更新するゴルーチン
func (sm *ScheduleManager) StartPeriodicRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("番組表の定期更新を停止します")
			return
		case <-ticker.C:
			log.Println("番組表を定期更新中...")
			if err := sm.RefreshFromFirestore(ctx); err != nil {
				log.Printf("定期更新でエラーが発生: %v", err)
			}
		}
	}
}

const SEGMENT_DURATION float64 = 3
const PLAYLIST_LENGTH int = 15

func main() {
	ctx := context.Background()
	client := initFireStore(ctx)
	defer client.Close()

	// 番組表マネージャーを初期化
	scheduleManager := &ScheduleManager{
		client: client,
	}

	// 初回の番組表読み込み
	if err := scheduleManager.RefreshFromFirestore(ctx); err != nil {
		log.Printf("初回番組表読み込みに失敗: %v", err)
		// 失敗した場合は空の番組表で開始
		scheduleManager.UpdateSchedule([]ProgramItem{})
	}

	// 番組表の定期更新を開始（5分間隔）
	go scheduleManager.StartPeriodicRefresh(ctx, 5*time.Minute)

	router := gin.Default()

	// HTMLファイルを配信するエンドポイント
	router.GET("/", func(c *gin.Context) {
		c.File("./index.html")
	})

	// ライブストリーム用のプレイリスト
	router.GET("/live/video.m3u8", func(c *gin.Context) {
		schedule := scheduleManager.GetSchedule()
		getVodPlaylist(c.Writer, c.Request, schedule)
	})

	// ストリーム状態確認用エンドポイント
	router.HEAD("/live/status", func(c *gin.Context) {
		schedule := scheduleManager.GetSchedule()
		getStreamStatus(c, schedule)
	})

	// 番組表を手動で更新するエンドポイント（デバッグ用）
	router.POST("/api/refresh-schedule", func(c *gin.Context) {
		if err := scheduleManager.RefreshFromFirestore(ctx); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		schedule := scheduleManager.GetSchedule()
		c.JSON(200, gin.H{
			"message": "番組表を更新しました",
			"count":   len(schedule),
		})
	})

	// 現在の番組表を取得するエンドポイント（デバッグ用）
	router.GET("/api/schedule", func(c *gin.Context) {
		schedule := scheduleManager.GetSchedule()
		c.JSON(200, gin.H{
			"schedule": schedule,
			"count":    len(schedule),
		})
	})

	// 静的ファイル（画像など）を配信
	router.Static("/static", "./static")

	log.Println("サーバーを開始します: http://0.0.0.0:8080")
	router.Run("0.0.0.0:8080")
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
