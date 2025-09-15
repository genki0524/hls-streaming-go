package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ProjectID string
	Bucket    string
	Port      string
}

func Load() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Printf("環境変数ファイルの読み込みに失敗: %v\n", err)
	}

	config := &Config{
		ProjectID: getEnv("PROJECT_ID", ""),
		Bucket:    getEnv("BUCKET", ""),
		Port:      getEnv("PORT", "8080"),
	}

	if config.ProjectID == "" {
		return nil, fmt.Errorf("PROJECT_ID環境変数が設定されていません")
	}

	if config.Bucket == "" {
		return nil, fmt.Errorf("BUCKET環境変数が設定されていません")
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}