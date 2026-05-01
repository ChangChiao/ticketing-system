package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/ticketing-system/backend/internal/middleware"
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

func (s *OrderService) StartPaymentTimeoutWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			orders, err := s.repo.GetExpiredPendingOrders(ctx)
			if err != nil {
				log.Printf("payment timeout worker: %v", err)
				continue
			}
			for _, order := range orders {
				if err := s.CancelOrder(ctx, order.ID); err != nil {
					log.Printf("payment timeout worker: cancel order %s: %v", order.ID, err)
					continue
				}
				middleware.PaymentTotal.WithLabelValues("timeout").Inc()
				middleware.ErrorsTotal.WithLabelValues("payment_timeout").Inc()
				_ = s.redis.PublishPaymentWarning(ctx, pkgredis.PaymentWarningMessage{
					UserID:  order.UserID,
					OrderID: order.ID,
					EventID: order.EventID,
					Type:    "timeout",
				})
			}
		}
	}
}

// generateCallbackToken creates a cryptographically random 32-byte hex token.
func generateCallbackToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *OrderService) CreateOrder(ctx context.Context, userID, eventID string, seats []model.SeatInfo) (*model.Order, error) {
	lockedSeats, err := s.seatSvc.GetLockedSeatsForOrder(ctx, eventID, userID, seats)
	if err != nil {
		return nil, err
	}

	callbackToken, err := generateCallbackToken()
	if err != nil {
		return nil, err
	}

	total := 0
	for _, seat := range lockedSeats {
		total += seat.Price
	}

	now := time.Now()
	order := &model.Order{
		ID:            uuid.New().String(),
		UserID:        userID,
		EventID:       eventID,
		Status:        "pending",
		Total:         total,
		CallbackToken: callbackToken,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	items := make([]model.OrderItem, len(lockedSeats))
	for i, seat := range lockedSeats {
		items[i] = model.OrderItem{
			ID:          uuid.New().String(),
			OrderID:     order.ID,
			EventSeatID: seat.EventSeatID,
			SectionName: seat.SectionName,
			RowLabel:    seat.RowLabel,
			SeatNumber:  seat.SeatNumber,
			Price:       seat.Price,
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

func (s *OrderService) GetUserOrder(ctx context.Context, id, userID string) (*model.Order, []model.OrderItem, error) {
	order, err := s.repo.GetByIDForUser(ctx, id, userID)
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

func (s *OrderService) ListPaymentPendingOrders(ctx context.Context) ([]model.Order, error) {
	return s.repo.ListByStatus(ctx, "payment_pending")
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

	seatIDs := make([]string, len(items))
	for i, item := range items {
		seatIDs[i] = item.EventSeatID
	}

	payment := &model.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: transactionID,
		Method:        "linepay",
		Amount:        order.Total,
		Status:        "confirmed",
		CreatedAt:     time.Now(),
	}

	// All DB writes in a single transaction: mark seats sold + create payment + update order
	if err := s.repo.ConfirmOrderTx(ctx, orderID, order.EventID, seatIDs, payment); err != nil {
		return err
	}

	// Publish availability updates via Redis (after successful commit)
	s.seatSvc.PublishAvailabilityUpdate(ctx, order.EventID)

	return nil
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

// ValidateCallbackToken verifies that the provided token matches the order's stored callback_token.
func (s *OrderService) ValidateCallbackToken(ctx context.Context, orderID, token string) (bool, error) {
	return s.repo.ValidateCallbackToken(ctx, orderID, token)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != "pending" && order.Status != "payment_pending" {
		return nil
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
	_, err = s.repo.UpdateStatusIfCurrent(ctx, orderID, "cancelled", []string{"pending", "payment_pending"})
	if err != nil {
		return err
	}
	return nil
}
