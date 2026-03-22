package service

import (
	"context"

	"github.com/ticketing-system/backend/internal/model"
	"github.com/ticketing-system/backend/internal/repository"
)

type EventService struct {
	repo *repository.EventRepository
}

func NewEventService(repo *repository.EventRepository) *EventService {
	return &EventService{repo: repo}
}

func (s *EventService) ListEvents(ctx context.Context) ([]model.EventListItem, error) {
	return s.repo.ListEvents(ctx)
}

func (s *EventService) GetEvent(ctx context.Context, id string) (*model.EventDetail, error) {
	return s.repo.GetEvent(ctx, id)
}
