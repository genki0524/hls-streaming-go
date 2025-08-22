package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/joho/godotenv"
)

type ProgramItem struct {
	StartTime    string `firestore:"start_time"`
	DurationSec  int32  `firestore:"duration_sec"`
	Type         string `firestore:"type"`
	PathTemplate string `firestore:"path_template"`
}

type Schedule struct {
	Programs []ProgramItem `firestore:"programs"`
}

const SEGMENT_DURATION int = 9
const PLAYLIST_LENGTH int = 6

func main() {
	ctx := context.Background()
	client := initFireStore(ctx)
	defer client.Close()

	todayString := time.Now().Format("2006-01-02")

	docRef := client.Collection("schedules").Doc(todayString)

	doc, err := docRef.Get(ctx)
	if err != nil {
		log.Fatalf("ドキュメントの取得に失敗しました: %v", err)
	}

	var data Schedule
	if err := doc.DataTo(&data); err != nil {
		log.Fatalf("データのマッピングに失敗しました: %v", err)
	}

	schedule := data.Programs
	sort.Slice(schedule, func(i, j int) bool {
		return schedule[i].StartTime < schedule[j].StartTime
	})

	// router := gin.Default()

	// // GET /programs エンドポイントを設定
	// router.GET("/programs", func(c *gin.Context) {
	// 	getProgramsHandler(client, ctx, c)
	// })

	// router.Run("localhost:8000")
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

func getProgramByGlobalSegment(globalSegmentIndex int, schedule []ProgramItem) (ProgramItem, int, error) {
	currentGlobalIndex := 0

	for _, program := range schedule {
		programSegments := math.Floor(float64(program.DurationSec) / float64(SEGMENT_DURATION))

		if currentGlobalIndex <= globalSegmentIndex && globalSegmentIndex < currentGlobalIndex+int(programSegments) {
			programSegmentIndex := globalSegmentIndex - currentGlobalIndex
			return program, programSegmentIndex, nil
		}
	}
	return ProgramItem{}, 0, fmt.Errorf("globalSegmentIndex is out of range")
}
