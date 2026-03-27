package service

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/ticketing-system/backend/internal/model"
	"github.com/ticketing-system/backend/internal/repository"
	pkgredis "github.com/ticketing-system/backend/pkg/redis"
)

type OrderService struct {
	repo    *repository.OrderRepository
	seatSvc *SeatService
	redis   *pkgredis.Client
}

func NewOrderService(repo *repository.OrderRepository, seatSvc *SeatService, redis *pkgredis.Client) *OrderService {
	return &OrderService{repo: repo, seatSvc: seatSvc, redis: redis}
}

// StartPaymentWarningWorker periodically checks for pending orders nearing the 8-minute mark
// and publishes a 2-minute countdown warning via Redis Pub/Sub.
func (s *OrderService) StartPaymentWarningWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			orders, err := s.repo.GetPendingOrdersNearExpiry(ctx)
			if err != nil {
				log.Printf("payment warning worker: %v", err)
				continue
			}
			for _, order := range orders {
				_ = s.redis.PublishPaymentWarning(ctx, pkgredis.PaymentWarningMessage{
					UserID:  order.UserID,
					OrderID: order.ID,
					EventID: order.EventID,
					Type:    "two_min_warning",
				})
			}
		}
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID, eventID string, seats []model.SeatInfo, pricePerSeat int) (*model.Order, error) {
	now := time.Now()
	order := &model.Order{
		ID:        uuid.New().String(),
		UserID:    userID,
		EventID:   eventID,
		Status:    "pending",
		Total:     pricePerSeat * len(seats),
		CreatedAt: now,
		UpdatedAt: now,
	}

	items := make([]model.OrderItem, len(seats))
	for i, seat := range seats {
		items[i] = model.OrderItem{
			ID:          uuid.New().String(),
			OrderID:     order.ID,
			EventSeatID: seat.EventSeatID,
			SectionName: seat.SectionName,
			RowLabel:    seat.RowLabel,
			SeatNumber:  seat.SeatNumber,
			Price:       pricePerSeat,
		}
	}

	if err := s.repo.Create(ctx, order, items); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (*model.Order, []model.OrderItem, error) {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.repo.GetOrderItems(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return order, items, nil
}

func (s *OrderService) ListOrders(ctx context.Context, userID string) ([]model.Order, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *OrderService) ConfirmOrder(ctx context.Context, orderID, transactionID string) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	items, err := s.repo.GetOrderItems(ctx, orderID)
	if err != nil {
		return err
	}

	// Mark seats as sold
	seatIDs := make([]string, len(items))
	for i, item := range items {
		seatIDs[i] = item.EventSeatID
	}
	if err := s.seatSvc.ConfirmSeats(ctx, order.EventID, seatIDs); err != nil {
		return err
	}

	// Create payment record
	payment := &model.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: transactionID,
		Method:        "linepay",
		Amount:        order.Total,
		Status:        "confirmed",
		CreatedAt:     time.Now(),
	}
	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return err
	}

	return s.repo.UpdateStatus(ctx, orderID, "confirmed")
}

// AreSeatsExpired checks if seat locks for an order have expired.
func (s *OrderService) AreSeatsExpired(ctx context.Context, orderID string) (bool, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return false, err
	}
	items, err := s.repo.GetOrderItems(ctx, orderID)
	if err != nil {
		return false, err
	}
	seatIDs := make([]string, len(items))
	for i, item := range items {
		seatIDs[i] = item.EventSeatID
	}
	return s.seatSvc.AreSeatLocksExpired(ctx, order.EventID, seatIDs)
}

// MarkPaymentPending sets order status to payment_pending for manual review.
func (s *OrderService) MarkPaymentPending(ctx context.Context, orderID string) error {
	return s.repo.UpdateStatus(ctx, orderID, "payment_pending")
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	items, err := s.repo.GetOrderItems(ctx, orderID)
	if err != nil {
		return err
	}

	seatIDs := make([]string, len(items))
	for i, item := range items {
		seatIDs[i] = item.EventSeatID
	}
	if err := s.seatSvc.ReleaseSeatsByEvent(ctx, order.EventID, seatIDs); err != nil {
		return err
	}

	return s.repo.UpdateStatus(ctx, orderID, "cancelled")
}
