package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/ticketing-system/backend/internal/model"
	"github.com/ticketing-system/backend/internal/repository"
	pkgredis "github.com/ticketing-system/backend/pkg/redis"
)

var (
	ErrNoConsecutiveSeats = errors.New("此區域已無連續座位，請減少張數或選擇其他區域")
	ErrSeatLockFailed     = errors.New("此區域座位搶購中，請稍後重試或選擇其他區域")
	ErrInvalidSeatLock    = errors.New("座位鎖定已失效，請重新選位")
)

const maxRetries = 3

type SeatService struct {
	repo  *repository.SeatRepository
	redis *pkgredis.Client
}

func NewSeatService(repo *repository.SeatRepository, redis *pkgredis.Client) *SeatService {
	return &SeatService{repo: repo, redis: redis}
}

func (s *SeatService) GetAvailability(ctx context.Context, eventID string) ([]model.SectionAvailability, error) {
	return s.repo.GetAvailability(ctx, eventID)
}

func (s *SeatService) GetLockedSeatsForOrder(ctx context.Context, eventID, userID string, requestedSeats []model.SeatInfo) ([]repository.LockedSeatForOrder, error) {
	eventSeatIDs := make([]string, 0, len(requestedSeats))
	for _, seat := range requestedSeats {
		if seat.EventSeatID == "" {
			return nil, ErrInvalidSeatLock
		}
		eventSeatIDs = append(eventSeatIDs, seat.EventSeatID)
	}
	if len(eventSeatIDs) == 0 {
		return nil, ErrInvalidSeatLock
	}

	seats, err := s.repo.GetLockedSeatsForOrder(ctx, eventID, userID, eventSeatIDs)
	if err != nil {
		return nil, err
	}
	if len(seats) != len(eventSeatIDs) {
		return nil, fmt.Errorf("%w: expected %d locked seats, got %d", ErrInvalidSeatLock, len(eventSeatIDs), len(seats))
	}
	return seats, nil
}

func (s *SeatService) StartExpiredLockCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.repo.ReleaseExpiredLocks(ctx); err != nil {
				log.Printf("seat lock cleanup worker: %v", err)
			}
		}
	}
}

func (s *SeatService) AllocateSeats(ctx context.Context, eventID, sectionID, userID string, quantity int) (*model.AllocatedSeats, error) {
	_, sectionName, err := s.repo.GetSectionInfo(ctx, eventID, sectionID)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		seats, err := s.findConsecutiveSeats(ctx, eventID, sectionID, quantity)
		if err != nil {
			return nil, err
		}

		sessionID := uuid.New().String()
		seatIDs := make([]string, len(seats))
		for i, seat := range seats {
			seatIDs[i] = seat.SeatID
		}

		locked, err := s.redis.LockSeats(ctx, eventID, seatIDs, sessionID)
		if err != nil {
			return nil, err
		}
		if !locked {
			continue // retry with different seats
		}

		if err := s.repo.MarkSeatsAsLocked(ctx, eventID, seatIDs, userID); err != nil {
			_ = s.redis.UnlockSeats(ctx, eventID, seatIDs)
			return nil, err
		}

		seatInfos := make([]model.SeatInfo, len(seats))
		for i, seat := range seats {
			seatInfos[i] = model.SeatInfo{
				EventSeatID: seat.SeatID,
				SectionName: sectionName,
				RowLabel:    seat.RowLabel,
				SeatNumber:  seat.Number,
			}
		}

		// Publish availability update via Redis Pub/Sub
		availability, _ := s.repo.GetAvailability(ctx, eventID)
		for _, a := range availability {
			if a.SectionID == sectionID {
				_ = s.redis.PublishAvailability(ctx, eventID, sectionID, a.Remaining)
				break
			}
		}

		return &model.AllocatedSeats{
			SessionID: sessionID,
			Seats:     seatInfos,
			ExpiresAt: time.Now().Add(pkgredis.SeatLockTTL),
		}, nil
	}

	return nil, ErrSeatLockFailed
}

func (s *SeatService) ReleaseSeatsByEvent(ctx context.Context, eventID string, seatIDs []string) error {
	if err := s.redis.UnlockSeats(ctx, eventID, seatIDs); err != nil {
		return err
	}
	if err := s.repo.ReleaseSeats(ctx, eventID, seatIDs); err != nil {
		return err
	}

	// Publish availability updates for affected sections
	availability, _ := s.repo.GetAvailability(ctx, eventID)
	for _, a := range availability {
		_ = s.redis.PublishAvailability(ctx, eventID, a.SectionID, a.Remaining)
	}
	return nil
}

func (s *SeatService) ReleaseLockedSeatsForUser(ctx context.Context, eventID, userID string, seatIDs []string) error {
	if len(seatIDs) == 0 {
		return nil
	}
	seats, err := s.repo.GetLockedSeatsForOrder(ctx, eventID, userID, seatIDs)
	if err != nil {
		return err
	}
	if len(seats) != len(seatIDs) {
		return ErrInvalidSeatLock
	}
	released, err := s.repo.ReleaseSeatsForUser(ctx, eventID, userID, seatIDs)
	if err != nil {
		return err
	}
	if released != int64(len(seatIDs)) {
		return ErrInvalidSeatLock
	}
	if err := s.redis.UnlockSeats(ctx, eventID, seatIDs); err != nil {
		return err
	}

	availability, _ := s.repo.GetAvailability(ctx, eventID)
	for _, a := range availability {
		_ = s.redis.PublishAvailability(ctx, eventID, a.SectionID, a.Remaining)
	}
	return nil
}

func (s *SeatService) ConfirmSeats(ctx context.Context, eventID string, seatIDs []string) error {
	if err := s.repo.MarkSeatsAsSold(ctx, eventID, seatIDs); err != nil {
		return err
	}

	// Publish availability updates for affected sections
	availability, _ := s.repo.GetAvailability(ctx, eventID)
	for _, a := range availability {
		_ = s.redis.PublishAvailability(ctx, eventID, a.SectionID, a.Remaining)
	}
	return nil
}

// PublishAvailabilityUpdate fetches current availability and publishes updates via Redis Pub/Sub.
func (s *SeatService) PublishAvailabilityUpdate(ctx context.Context, eventID string) {
	availability, _ := s.repo.GetAvailability(ctx, eventID)
	for _, a := range availability {
		_ = s.redis.PublishAvailability(ctx, eventID, a.SectionID, a.Remaining)
	}
}

// AreSeatLocksExpired checks if any of the seat locks have expired in Redis.
func (s *SeatService) AreSeatLocksExpired(ctx context.Context, eventID string, seatIDs []string) (bool, error) {
	locked, err := s.redis.AreSeatsLocked(ctx, eventID, seatIDs)
	if err != nil {
		return false, err
	}
	for _, isLocked := range locked {
		if !isLocked {
			return true, nil
		}
	}
	return false, nil
}

func (s *SeatService) findConsecutiveSeats(ctx context.Context, eventID, sectionID string, quantity int) ([]repository.RowWithSeats, error) {
	allSeats, err := s.repo.GetAvailableSeatsInSection(ctx, eventID, sectionID)
	if err != nil {
		return nil, err
	}

	// Group seats by row
	rowSeats := make(map[string][]repository.RowWithSeats)
	rowOrders := make(map[string]int)
	for _, seat := range allSeats {
		rowSeats[seat.RowID] = append(rowSeats[seat.RowID], seat)
		rowOrders[seat.RowID] = seat.SortOrder
	}

	// Sort rows by middle-first strategy
	rowIDs := make([]string, 0, len(rowSeats))
	for rowID := range rowSeats {
		rowIDs = append(rowIDs, rowID)
	}
	totalRows := len(rowIDs)
	sort.Slice(rowIDs, func(i, j int) bool {
		mid := totalRows / 2
		distI := abs(rowOrders[rowIDs[i]] - mid)
		distJ := abs(rowOrders[rowIDs[j]] - mid)
		return distI < distJ
	})

	// Sliding window to find consecutive seats in each row
	for _, rowID := range rowIDs {
		seats := rowSeats[rowID]
		// Sort by seat number
		sort.Slice(seats, func(i, j int) bool {
			return seats[i].Number < seats[j].Number
		})

		// Batch check locks in Redis via pipeline
		seatIDs := make([]string, len(seats))
		for i, seat := range seats {
			seatIDs[i] = seat.SeatID
		}
		locked, err := s.redis.AreSeatsLocked(ctx, eventID, seatIDs)
		if err != nil {
			return nil, err
		}
		availableSeats := make([]repository.RowWithSeats, 0, len(seats))
		for i, seat := range seats {
			if !locked[i] {
				availableSeats = append(availableSeats, seat)
			}
		}

		result := findConsecutiveInRow(availableSeats, quantity)
		if result != nil {
			return result, nil
		}
	}

	return nil, ErrNoConsecutiveSeats
}

func findConsecutiveInRow(seats []repository.RowWithSeats, quantity int) []repository.RowWithSeats {
	if len(seats) < quantity {
		return nil
	}

	for i := 0; i <= len(seats)-quantity; i++ {
		consecutive := true
		for j := 1; j < quantity; j++ {
			if seats[i+j].Number != seats[i+j-1].Number+1 {
				consecutive = false
				break
			}
		}
		if consecutive {
			result := make([]repository.RowWithSeats, quantity)
			copy(result, seats[i:i+quantity])
			return result
		}
	}
	return nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
