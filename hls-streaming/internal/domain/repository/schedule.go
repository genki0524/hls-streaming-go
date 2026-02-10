package repository

import (
	"context"

	"github.com/genki0524/hls_striming_go/internal/domain"
)

type ScheduleRepository interface {
	GetScheduleByDate(ctx context.Context, date string) (*domain.Schedule, error)
	PostSchedule(ctx context.Context, request domain.RequestProgramItem, date string) error
}
