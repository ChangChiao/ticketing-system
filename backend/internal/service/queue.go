package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	pkgredis "github.com/ticketing-system/backend/pkg/redis"
)

var ErrAlreadyInQueue = errors.New("您已在排隊中，請回到原視窗")

const (
	BatchSize          = 50
	MaxConcurrent      = 500
	BatchIntervalS     = 5
	EntryWindowSeconds = 60
)

type QueueService struct {
	redis *pkgredis.Client
}

func NewQueueService(redis *pkgredis.Client) *QueueService {
	return &QueueService{redis: redis}
}

func (s *QueueService) JoinQueue(ctx context.Context, eventID, userID string) (int64, error) {
	// Enforce single session
	ok, err := s.redis.SetActiveSession(ctx, eventID, userID, userID)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrAlreadyInQueue
	}

	if err := s.redis.QueueJoin(ctx, eventID, userID); err != nil {
		return 0, err
	}

	return s.redis.QueuePosition(ctx, eventID, userID)
}

func (s *QueueService) GetPosition(ctx context.Context, eventID, userID string) (int64, error) {
	return s.redis.QueuePosition(ctx, eventID, userID)
}

func (s *QueueService) EstimateWait(position int64) string {
	if position <= 0 {
		return "即將輪到您"
	}
	batches := position / BatchSize
	seconds := batches * BatchIntervalS
	if seconds < 60 {
		return fmt.Sprintf("約 %d 秒", seconds)
	}
	return fmt.Sprintf("約 %d 分鐘", seconds/60+1)
}

func (s *QueueService) AdmitNextBatch(ctx context.Context, eventID string) ([]string, error) {
	tokens, err := s.redis.QueuePop(ctx, eventID, BatchSize)
	if err != nil {
		return nil, err
	}
	for _, token := range tokens {
		if err := s.redis.SetQueueAdmission(ctx, eventID, token, EntryWindowSeconds*time.Second); err != nil {
			return nil, err
		}
	}
	return tokens, nil
}

func (s *QueueService) IsAdmitted(ctx context.Context, eventID, userID string) (bool, error) {
	return s.redis.HasQueueAdmission(ctx, eventID, userID)
}

func (s *QueueService) ActiveEventIDs(ctx context.Context) ([]string, error) {
	return s.redis.QueueEventIDs(ctx)
}

func (s *QueueService) QueueMembers(ctx context.Context, eventID string) ([]string, error) {
	return s.redis.QueueMembers(ctx, eventID)
}

func (s *QueueService) QueueSize(ctx context.Context, eventID string) (int64, error) {
	return s.redis.QueueSize(ctx, eventID)
}

func (s *QueueService) StartAdmissionWorker(ctx context.Context, notify func(eventID, userID string)) {
	ticker := time.NewTicker(BatchIntervalS * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			eventIDs, err := s.ActiveEventIDs(ctx)
			if err != nil {
				log.Printf("queue admission worker: list events: %v", err)
				continue
			}
			for _, eventID := range eventIDs {
				users, err := s.AdmitNextBatch(ctx, eventID)
				if err != nil {
					log.Printf("queue admission worker: admit %s: %v", eventID, err)
					continue
				}
				for _, userID := range users {
					notify(eventID, userID)
				}
			}
		}
	}
}

func (s *QueueService) StartPositionUpdateWorker(ctx context.Context, notify func(eventID, userID string, position, total int64, estimatedWait string)) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			eventIDs, err := s.ActiveEventIDs(ctx)
			if err != nil {
				log.Printf("queue position worker: list events: %v", err)
				continue
			}
			for _, eventID := range eventIDs {
				users, err := s.QueueMembers(ctx, eventID)
				if err != nil {
					log.Printf("queue position worker: members %s: %v", eventID, err)
					continue
				}
				total := int64(len(users))
				for position, userID := range users {
					pos := int64(position)
					notify(eventID, userID, pos, total, s.EstimateWait(pos))
				}
			}
		}
	}
}
