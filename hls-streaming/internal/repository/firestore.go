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