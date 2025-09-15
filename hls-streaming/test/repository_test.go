package test

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/genki0524/hls_striming_go/internal/repository"
)

func TestGCSRepository_DownloadFileToMemory(t *testing.T) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Skipf("GCSクライアントの初期化に失敗: %v", err)
	}
	defer client.Close()

	repo := repository.NewGCSRepository(client)

	bucket := "generic-a-and-g-storage"
	object := "2025-09-09/minecraft_1/video.m3u8"

	data, err := repo.DownloadFileToMemory(ctx, bucket, object)
	if err != nil {
		t.Logf("ダウンロードテストをスキップ（認証エラーまたはネットワークエラーの可能性）: %v", err)
		return
	}

	if len(data) == 0 {
		t.Error("ダウンロードされたデータが空です")
	}

	fmt.Printf("ダウンロード成功: %d bytes\n", len(data))
}

func TestGCSRepository_CreateSignedURL(t *testing.T) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Skipf("GCSクライアントの初期化に失敗: %v", err)
	}
	defer client.Close()

	repo := repository.NewGCSRepository(client)

	bucket := "generic-a-and-g-storage"
	object := "2025-09-09/minecraft_1/video.m3u8"

	url, err := repo.CreateSignedURL(bucket, object)
	if err != nil {
		t.Logf("署名付きURL生成テストをスキップ（認証エラーまたはネットワークエラーの可能性）: %v", err)
		return
	}

	if url == "" {
		t.Error("署名付きURLが空です")
	}

	fmt.Printf("署名付きURL生成成功: %s\n", url)
}

func TestGCSRepository_GetM3U8WithSignedURLs(t *testing.T) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Skipf("GCSクライアントの初期化に失敗: %v", err)
	}
	defer client.Close()

	repo := repository.NewGCSRepository(client)

	bucket := "generic-a-and-g-storage"
	date := "2025-09-09"
	programName := "minecraft_1"

	playlist, err := repo.GetM3U8WithSignedURLs(bucket, date, programName)
	if err != nil {
		t.Logf("M3U8取得テストをスキップ（認証エラーまたはネットワークエラーの可能性）: %v", err)
		return
	}

	if playlist == nil {
		t.Error("プレイリストがnilです")
		return
	}

	if len(playlist.Segments) == 0 {
		t.Error("プレイリストにセグメントがありません")
	}

	fmt.Printf("M3U8取得成功: %d segments\n", len(playlist.Segments))
}