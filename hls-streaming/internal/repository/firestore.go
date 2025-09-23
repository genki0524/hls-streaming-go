package repository

import (
	"context"
	"sort"

	"cloud.google.com/go/firestore"
	"github.com/genki0524/hls_striming_go/internal/domain"
)

type FirestoreScheduleRepository struct {
	client *firestore.Client
}

func NewFirestoreScheduleRepository(client *firestore.Client) *FirestoreScheduleRepository {
	return &FirestoreScheduleRepository{
		client: client,
	}
}

func (r *FirestoreScheduleRepository) GetScheduleByDate(ctx context.Context, date string) (*domain.Schedule, error) {
	docRef := r.client.Collection("schedules").Doc(date)

	doc, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var data domain.Schedule
	if err := doc.DataTo(&data); err != nil {
		return nil, err
	}

	sort.Slice(data.Programs, func(i, j int) bool {
		return data.Programs[i].StartTime < data.Programs[j].StartTime
	})

	return &data, nil
}

func (r *FirestoreScheduleRepository) PostSchedule(ctx context.Context, request domain.RequestProgramItem, date string) error {
	docRef := r.client.Collection("schedules").Doc(date)

	// 既存のスケジュールを取得
	doc, err := docRef.Get(ctx)

	program := request2ProgramItem(request)

	if err != nil {
		// ドキュメントが存在しない場合は新規作成
		schedule := domain.Schedule{
			Programs: []domain.ProgramItem{program},
		}
		_, err = docRef.Set(ctx, schedule)
		return err
	}

	// 既存のスケジュールに番組を追加
	var existingSchedule domain.Schedule
	if err := doc.DataTo(&existingSchedule); err != nil {
		return err
	}

	existingSchedule.Programs = append(existingSchedule.Programs, program)

	_, err = docRef.Set(ctx, existingSchedule)
	return err
}

func request2ProgramItem(request domain.RequestProgramItem) domain.ProgramItem {
	return domain.ProgramItem{
		StartTime:    request.StartTime,
		DurationSec:  request.DurationSec,
		Type:         request.Type,
		PathTemplate: request.PathTemplate,
		Title:        request.Title,
	}
}
