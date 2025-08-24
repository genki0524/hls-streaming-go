# HLS Streaming Server

HLS（HTTP Live Streaming）を使用したライブストリーミングサーバーです。Google Cloud Firestoreと連携して、スケジュールに基づいた動画配信を行います。

## 特徴

- HLS形式での動画ストリーミング配信
- Google Cloud Firestoreを使用したスケジュール管理
- リアルタイムでの番組切り替え
- 静的画像表示（番組間の待機時間）
- WebベースのHLSプレイヤー

## ディレクトリ構成

```
hls-streaming/
├── docker-compose.yaml         # Docker Compose設定ファイル
├── go.mod                     # Go モジュール定義
├── go.sum                     # Go モジュール依存関係
├── main.go                    # メインアプリケーション
├── program.go                 # 番組管理ロジック
├── index.html                 # HLSプレイヤーのWebページ
├── .env                       # 環境変数設定ファイル（要作成）
├── .gitignore                 # Git除外ファイル設定
├── credentials/               # Google Cloud認証情報
│   └── *.json                 # サービスアカウントキーファイル
└── static/                    # 静的ファイル
    ├── images/                # 画像ファイル（Git管理外）
    │   └── *.jpg, *.png       # 静止画・待機画面用画像
    └── stream/                # HLSストリームファイル（Git管理外）
        ├── [番組名]/          # 各番組ディレクトリ
        │   ├── *.mp4          # 元動画ファイル
        │   ├── video.m3u8     # HLSプレイリストファイル
        │   └── video*.ts      # HLSセグメントファイル
        └── [番組名]/          # 別の番組ディレクトリ
            ├── *.mp4          # 元動画ファイル
            ├── video.m3u8     # HLSプレイリストファイル
            └── video*.ts      # HLSセグメントファイル
```

## 起動に必要なこと

### 1. 環境変数の設定

プロジェクトルートに `.env` ファイルを作成し、以下の設定を記述してください：

```env
PROJECT_ID=your-google-cloud-project-id
```

### 2. Google Cloud Firestore の設定

1. Google Cloud Projectを作成
2. Firestoreを有効化
3. サービスアカウントキーを作成し、`credentials/` ディレクトリに配置
4. 環境変数 `GOOGLE_APPLICATION_CREDENTIALS` を設定（オプション）

### 3. Firestoreデータ構造

`schedules` コレクションに以下の形式でドキュメントを作成してください：

```json
{
  "programs": [
    {
      "start_time": "2025-08-25T09:00:00+09:00",
      "duration_sec": 1800,
      "type": "video",
      "path_template": "handgesture",
      "title": "手話動画"
    },
    {
      "start_time": "2025-08-25T10:00:00+09:00", 
      "duration_sec": 1200,
      "type": "video",
      "path_template": "minecraft",
      "title": "Minecraft動画"
    }
  ]
}
```

ドキュメントIDは日付形式（例：`2025-08-25`）で作成してください。

### 4. 動画ファイルの準備

HLS形式の動画ファイルを `static/stream/` ディレクトリに配置してください。各動画フォルダには以下のファイルが必要です：

- `video.m3u8` - HLSプレイリストファイル
- `video000.ts`, `video001.ts`, ... - 動画セグメントファイル

## 起動方法

### Docker Composeを使用する場合（推奨）

```bash
# プロジェクトディレクトリに移動
cd hls-streaming

# コンテナの起動
docker-compose up -d

# コンテナ内でのコマンド実行
docker-compose exec hls-striming bash

# 依存関係のインストール
go mod download

# アプリケーションの起動
go run .
```

### ローカル環境で直接起動する場合

```bash
# プロジェクトディレクトリに移動
cd hls-streaming

# 依存関係のインストール
go mod download

# アプリケーションの起動
go run .
```

## アクセス方法

アプリケーション起動後、以下のURLにアクセスしてください：

- **Webプレイヤー**: http://localhost:8080 (Docker) / http://localhost:8000 (ローカル)
- **HLSプレイリスト**: http://localhost:8080/live/video.m3u8
- **ストリーム状態確認**: http://localhost:8080/live/status

## 主要な機能

### エンドポイント

- `GET /` - HLSプレイヤーのWebページ
- `GET /live/video.m3u8` - ライブストリーム用のHLSプレイリスト
- `HEAD /live/status` - ストリーム状態確認
- `GET /static/*` - 静的ファイルの配信

### 動作仕様

- セグメント長: 12秒
- プレイリスト長: 15セグメント
- スケジュールはFirestoreから日次で取得
- 番組間の待機時間は静的画像を表示

## 依存関係

- Go 1.25.0
- Gin Web Framework
- Google Cloud Firestore
- godotenv（環境変数管理）
