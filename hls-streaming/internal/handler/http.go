package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/genki0524/hls_striming_go/internal/service"
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	scheduleService  *service.ScheduleService
	streamingService *service.StreamingService
}

func NewHTTPHandler(scheduleService *service.ScheduleService, streamingService *service.StreamingService) *HTTPHandler {
	return &HTTPHandler{
		scheduleService:  scheduleService,
		streamingService: streamingService,
	}
}

func (h *HTTPHandler) SetupRoutes(router *gin.Engine) {
	router.GET("/", h.serveIndex)
	router.GET("/live/video.m3u8", h.getLivePlaylist)
	router.HEAD("/live/status", h.getStreamStatus)
	router.POST("/api/refresh-schedule", h.refreshSchedule)
	router.GET("/api/schedule", h.getSchedule)
	router.Static("/static", "./static")
}

func (h *HTTPHandler) serveIndex(c *gin.Context) {
	c.File("./index.html")
}

func (h *HTTPHandler) getLivePlaylist(c *gin.Context) {
	schedule := h.scheduleService.GetSchedule()

	playlist, err := h.streamingService.GenerateVODPlaylist(schedule)
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