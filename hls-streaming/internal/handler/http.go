package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/genki0524/hls_striming_go/internal/domain"
	"github.com/genki0524/hls_striming_go/internal/service"
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	scheduleService  *service.ScheduleService
	streamingService *service.StreamingService
	mediaService     *service.MediaService
}

func NewHTTPHandler(scheduleService *service.ScheduleService, streamingService *service.StreamingService, mediaService *service.MediaService) *HTTPHandler {
	return &HTTPHandler{
		scheduleService:  scheduleService,
		streamingService: streamingService,
		mediaService:     mediaService,
	}
}

func (h *HTTPHandler) SetupRoutes(router *gin.Engine) {
	router.GET("/", h.serveIndex)
	router.GET("/live/video.m3u8", h.getLivePlaylist)
	router.HEAD("/live/status", h.getStreamStatus)
	router.POST("/api/refresh-schedule", h.refreshSchedule)
	router.GET("/api/schedule", h.getSchedule)
	router.POST("/api/schedule", h.postSchedule)
	router.POST("/api/upload-video", h.uploadVideo)
	router.Static("/static", "./static")
}

func (h *HTTPHandler) serveIndex(c *gin.Context) {
	c.File("./index.html")
}

func (h *HTTPHandler) getLivePlaylist(c *gin.Context) {
	schedule := h.scheduleService.GetSchedule()

	playlist, err := h.streamingService.GenerateVODPlaylist(c.Request.Context(), schedule)
	if err != nil {
		log.Printf("プレイリスト生成エラー: %v", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.String(http.StatusOK, playlist)
}

func (h *HTTPHandler) getStreamStatus(c *gin.Context) {
	schedule := h.scheduleService.GetSchedule()
	status := h.streamingService.CheckStreamStatus(schedule)
	c.Status(status)
}

func (h *HTTPHandler) refreshSchedule(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.scheduleService.RefreshFromRepository(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	schedule := h.scheduleService.GetSchedule()
	c.JSON(http.StatusOK, gin.H{
		"message": "番組表を更新しました",
		"count":   len(schedule),
	})
}

func (h *HTTPHandler) getSchedule(c *gin.Context) {
	schedule := h.scheduleService.GetSchedule()
	c.JSON(http.StatusOK, gin.H{
		"schedule": schedule,
		"count":    len(schedule),
	})
}

func (h *HTTPHandler) postSchedule(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dateクエリパラメータが必要です"})
		return
	}

	var programItem domain.RequestProgramItem
	if err := c.ShouldBindJSON(&programItem); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "リクエストボディが不正です: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.scheduleService.AddProgramToSchedule(ctx, programItem, date); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "番組の追加に失敗しました: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "番組を追加しました",
		"program": programItem,
		"date":    date,
	})
}

// uploadVideo は動画ファイルをアップロードするエンドポイントです
func (h *HTTPHandler) uploadVideo(c *gin.Context) {
	// 1. ファイルサイズ制限の確認（例：100MB）
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 100*1024*1024) // 100MB制限

	// 2. マルチパートフォームからファイルを取得
	file, header, err := c.Request.FormFile("video")
	if err != nil {
		log.Printf("ファイル取得エラー: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "動画ファイルが見つかりません。'video'フィールドでファイルを送信してください",
		})
		return
	}
	defer file.Close()

	// 3. ファイル形式の検証
	contentType := header.Header.Get("Content-Type")
	if !isValidVideoType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("サポートされていないファイル形式です: %s", contentType),
		})
		return
	}

	// 4. ファイル名の取得とオブジェクト名の生成
	fileName := header.Filename
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ファイル名が指定されていません",
		})
		return
	}

	// 5. 必要なパラメータの取得
	date := c.PostForm("date")
	programName := c.PostForm("program_name")
	if date == "" || programName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "date と program_name パラメータが必要です",
		})
		return
	}

	// 6. ファイルデータを読み込み
	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Printf("ファイル読み込みエラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "ファイルの読み込みに失敗しました",
		})
		return
	}

	// 7. ファイルサイズのチェック
	if len(fileData) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "空のファイルはアップロードできません",
		})
		return
	}

	// 8. MediaServiceを使用してHLS変換とアップロード
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Minute) // HLS変換用に15分
	defer cancel()

	log.Printf("HLS変換を開始します: プログラム=%s, 日付=%s, サイズ=%d bytes", programName, date, len(fileData))

	if err := h.mediaService.ConvertAndUploadHLS(ctx, fileData, date, programName); err != nil {
		log.Printf("HLS変換・アップロードエラー: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "動画のHLS変換・アップロードに失敗しました",
		})
		return
	}

	// 9. 成功レスポンス
	log.Printf("HLS変換・アップロード成功: 日付=%s, プログラム=%s (元サイズ: %d bytes)", date, programName, len(fileData))
	c.JSON(http.StatusOK, gin.H{
		"message":       "動画のHLS変換・アップロードが完了しました",
		"date":          date,
		"program_name":  programName,
		"file_name":     fileName,
		"original_size": len(fileData),
		"content_type":  contentType,
		"hls_path":      fmt.Sprintf("%s/%s", date, programName),
	})
}

// isValidVideoType は動画ファイルの形式が有効かチェックします
func isValidVideoType(contentType string) bool {
	validTypes := []string{
		"video/mp4",
		"video/mpeg",
		"video/quicktime",
		"video/x-msvideo", // AVI
		"video/x-ms-wmv",  // WMV
	}

	for _, validType := range validTypes {
		if strings.EqualFold(contentType, validType) {
			return true
		}
	}

	// Content-Typeが設定されていない場合、ファイル拡張子でチェック
	if contentType == "" || contentType == "application/octet-stream" {
		return true // より詳細な検証は別途実装
	}

	return false
}
