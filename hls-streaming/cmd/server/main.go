package main

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/genki0524/hls_striming_go/internal/domain"
	"github.com/genki0524/hls_striming_go/internal/handler"
	"github.com/genki0524/hls_striming_go/internal/repository"
	"github.com/genki0524/hls_striming_go/internal/service"
	"github.com/genki0524/hls_striming_go/pkg/config"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗: %v", err)
	}

	ctx := context.Background()

	firestoreClient, err := initFirestore(ctx, cfg.ProjectID)
	if err != nil {
		log.Fatalf("Firestoreクライアントの初期化に失敗: %v", err)
	}
	defer firestoreClient.Close()

	gcsClient, err := initGCS(ctx)
	if err != nil {
		log.Fatalf("GCSクライアントの初期化に失敗: %v", err)
	}
	defer gcsClient.Close()

	scheduleRepo := repository.NewFirestoreScheduleRepository(firestoreClient)
	gcsRepo := repository.NewGCSRepository(gcsClient)

	scheduleService := service.NewScheduleService(scheduleRepo)
	streamingService := service.NewStreamingService(gcsRepo)

	if err := scheduleService.RefreshFromRepository(ctx); err != nil {
		log.Printf("初回番組表読み込みに失敗: %v", err)
		scheduleService.UpdateSchedule([]domain.ProgramItem{})
	}

	go scheduleService.StartPeriodicRefresh(ctx, 5*time.Minute)

	httpHandler := handler.NewHTTPHandler(scheduleService, streamingService)

	router := gin.Default()
	httpHandler.SetupRoutes(router)

	log.Printf("サーバーを開始します: http://0.0.0.0:%s", cfg.Port)
	if err := router.Run("0.0.0.0:" + cfg.Port); err != nil {
		log.Fatalf("サーバーの起動に失敗: %v", err)
	}
}

func initFirestore(ctx context.Context, projectID string) (*firestore.Client, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func initGCS(ctx context.Context) (*storage.Client, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return client, nil
}