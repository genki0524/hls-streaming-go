package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/genki0524/hls_striming_go/internal/domain"
)

type ScheduleService struct {
	schedule   []domain.ProgramItem
	mutex      sync.RWMutex
	repository domain.ScheduleRepository
}

func NewScheduleService(repository domain.ScheduleRepository) *ScheduleService {
	return &ScheduleService{
		schedule:   make([]domain.ProgramItem, 0),
		repository: repository,
	}
}

func (s *ScheduleService) GetSchedule() []domain.ProgramItem {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make([]domain.ProgramItem, len(s.schedule))
	copy(result, s.schedule)
	return result
}

func (s *ScheduleService) UpdateSchedule(newSchedule []domain.ProgramItem) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.schedule = newSchedule
	log.Printf("番組表を更新しました。番組数: %d", len(newSchedule))
}

func (s *ScheduleService) RefreshFromRepository(ctx context.Context) error {
	jst := time.FixedZone("JST", 9*60*60)
	todayString := time.Now().In(jst).Format("2006-01-02")

	schedule, err := s.repository.GetScheduleByDate(ctx, todayString)
	if err != nil {
		log.Printf("Repositoryからの取得に失敗: %v", err)
		return err
	}

	s.UpdateSchedule(schedule.Programs)
	return nil
}

func (s *ScheduleService) StartPeriodicRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("番組表の定期更新を停止します")
			return
		case <-ticker.C:
			log.Println("番組表を定期更新中...")
			if err := s.RefreshFromRepository(ctx); err != nil {
				log.Printf("定期更新でエラーが発生: %v", err)
			}
		}
	}
}