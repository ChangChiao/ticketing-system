package service

import (
	"context"
	"errors"
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

		seatInfos := make([]model.SeatInfo, len(seats))
		for i, seat := range seats {
			seatInfos[i] = model.SeatInfo{
				EventSeatID: seat.SeatID,
				SectionName: sectionName,
				RowLabel:    seat.RowLabel,
				SeatNumber:  seat.Number,
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
	return s.repo.ReleaseSeats(ctx, eventID, seatIDs)
}

func (s *SeatService) ConfirmSeats(ctx context.Context, eventID string, seatIDs []string) error {
	return s.repo.MarkSeatsAsSold(ctx, eventID, seatIDs)
}

// AreSeatLocksExpired checks if any of the seat locks have expired in Redis.
func (s *SeatService) AreSeatLocksExpired(ctx context.Context, eventID string, seatIDs []string) (bool, error) {
	for _, seatID := range seatIDs {
		locked, err := s.redis.IsSeatLocked(ctx, eventID, seatID)
		if err != nil {
			return false, err
		}
		if !locked {
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

		// Check for locks in Redis
		availableSeats := make([]repository.RowWithSeats, 0, len(seats))
		for _, seat := range seats {
			locked, err := s.redis.IsSeatLocked(ctx, eventID, seat.SeatID)
			if err != nil {
				return nil, err
			}
			if !locked {
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
