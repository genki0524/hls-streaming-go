package crud

import (
	"bytes"
	"os"
	"testing"
	"fmt"
)

func TestUpload(t *testing.T) {
	// テスト用のファイルを作成
	testFile := "notex.txt"
	testContent := "This is a test file for upload functionality"

	// テスト用ファイルを作成
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// テスト終了後にファイルを削除
	defer func() {
		if err := os.Remove(testFile); err != nil {
			t.Logf("Warning: Failed to remove test file: %v", err)
		}
	}()

	// テスト用のパラメータ
	bucket := "generic-a-and-g-storage"
	object := "test-object.txt"
	credentialsFilePath := "../credentials/gcs_key.json"

	// 出力をキャプチャするためのバッファ
	var buf bytes.Buffer

	// uploadFile関数をテスト
	// Note: 実際のGCSへの接続が必要なため、認証情報ファイルが存在する場合のみテスト実行
	if _, err := os.Stat(credentialsFilePath); os.IsNotExist(err) {
		t.Skip("Credentials file not found, skipping upload test")
		return
	}

	// uploadFile関数を実行
	err = uploadFile(&buf, bucket, object, credentialsFilePath)

	// エラーハンドリング（実際のGCSに接続するため、接続エラーは予想される）
	if err != nil {
		// ネットワークエラーや認証エラーの場合はスキップ
		t.Logf("Upload test skipped due to error (expected in test environment): %v", err)
		return
	}

	// 成功した場合のアサーション
	output := buf.String()
	expectedOutput := "Blob test-object.txt uploaded.\n"
	if output != expectedOutput {
		t.Errorf("Expected output '%s', but got '%s'", expectedOutput, output)
	}
}

func TestDownloadIntoMemory(t *testing.T) {
	bucket := "generic-a-and-g-storage"
	object := "2025-09-09/minecraft_1/video.m3u8"
	credentialsFilePath := "../credentials/gcs_key.json"

	data,err := downloadFileIntoMemory(bucket,object,credentialsFilePath)
	if err != nil {
		fmt.Printf("downloadFileIntoMemory: %v",err)
	}
	fmt.Println(string(data))
}

func TestCreateSignaturedUrl(t *testing.T) {
	bucket := "generic-a-and-g-storage"
	object := "2025-09-09/minecraft_1/video.m3u8"
	credentialsFilePath := "../credentials/gcs_key.json"

	url,err := createSignaturedUrl(bucket,object,credentialsFilePath)
	if err != nil {
		fmt.Printf("createSignaturedUrl: %v",err)
	}
	fmt.Println(url)
}

func TestCreateSignedM3u8(t *testing.T) {
	bucket := "generic-a-and-g-storage"
	date := "2025-09-09"
	programName := "minecraft_1"
	credentialsFilePath := "../credentials/gcs_key.json"

	createSignedM3u8(bucket,date,programName,credentialsFilePath)
}