package domain

import (
	"context"
	"time"
)

type ProgramItem struct {
	StartTime    string `firestore:"start_time"`
	DurationSec  int32  `firestore:"duration_sec"`
	Type         string `firestore:"type"`
	PathTemplate string `firestore:"path_template"`
	Title        string `firestore:"title"`
}

type Schedule struct {
	Programs []ProgramItem `firestore:"programs"`
}

type ScheduleRepository interface {
	GetScheduleByDate(ctx context.Context, date string) (*Schedule, error)
}

func (p *ProgramItem) GetStartTime() (time.Time, error) {
	return time.Parse(time.RFC3339, p.StartTime)
}

func (p *ProgramItem) GetEndTime(jst *time.Location) (time.Time, error) {
	startTime, err := p.GetStartTime()
	if err != nil {
		return time.Time{}, err
	}

	startTimeJST := startTime.In(jst)
	return startTimeJST.Add(time.Duration(p.DurationSec) * time.Second), nil
}

func (p *ProgramItem) IsCurrentlyAiring(currentTime time.Time, jst *time.Location) bool {
	startTime, err := p.GetStartTime()
	if err != nil {
		return false
	}

	startTimeJST := startTime.In(jst)
	endTime, err := p.GetEndTime(jst)
	if err != nil {
		return false
	}

	return currentTime.After(startTimeJST) && currentTime.Before(endTime)
}

func FindCurrentProgram(schedule []ProgramItem, currentTime time.Time, jst *time.Location) (*ProgramItem, int) {
	for index, program := range schedule {
		if program.IsCurrentlyAiring(currentTime, jst) {
			return &program, index
		}
	}
	return nil, -1
}

func FindNextProgram(schedule []ProgramItem, currentTime time.Time, jst *time.Location) *ProgramItem {
	for _, program := range schedule {
		startTime, err := program.GetStartTime()
		if err != nil {
			continue
		}

		startTimeJST := startTime.In(jst)
		if startTimeJST.After(currentTime) {
			return &program
		}
	}
	return nil
}