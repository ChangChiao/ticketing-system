package service

import (
	"context"
	"errors"
	"fmt"

	pkgredis "github.com/ticketing-system/backend/pkg/redis"
)

var ErrAlreadyInQueue = errors.New("您已在排隊中，請回到原視窗")

const (
	BatchSize      = 50
	MaxConcurrent  = 500
	BatchIntervalS = 5
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
	return tokens, nil
}
