package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/ticketing-system/backend/internal/model"
)

type OrderRepository struct {
	db *sqlx.DB
}

func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, order *model.Order, items []model.OrderItem) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.NamedExecContext(ctx, `
		INSERT INTO orders (id, user_id, event_id, status, total, callback_token, created_at, updated_at)
		VALUES (:id, :user_id, :event_id, :status, :total, :callback_token, :created_at, :updated_at)
	`, order)
	if err != nil {
		return err
	}

	for _, item := range items {
		_, err = tx.NamedExecContext(ctx, `
			INSERT INTO order_items (id, order_id, event_seat_id, section_name, row_label, seat_number, price)
			VALUES (:id, :order_id, :event_seat_id, :section_name, :row_label, :seat_number, :price)
		`, item)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (*model.Order, error) {
	var order model.Order
	err := r.db.GetContext(ctx, &order, "SELECT * FROM orders WHERE id = $1", id)
	return &order, err
}

func (r *OrderRepository) GetByIDForUser(ctx context.Context, id, userID string) (*model.Order, error) {
	var order model.Order
	err := r.db.GetContext(ctx, &order, "SELECT * FROM orders WHERE id = $1 AND user_id = $2", id, userID)
	return &order, err
}

func (r *OrderRepository) GetOrderItems(ctx context.Context, orderID string) ([]model.OrderItem, error) {
	var items []model.OrderItem
	err := r.db.SelectContext(ctx, &items, "SELECT * FROM order_items WHERE order_id = $1", orderID)
	return items, err
}

func (r *OrderRepository) ListByUser(ctx context.Context, userID string) ([]model.Order, error) {
	var orders []model.Order
	err := r.db.SelectContext(ctx, &orders, "SELECT * FROM orders WHERE user_id = $1 ORDER BY created_at DESC", userID)
	return orders, err
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2", status, id)
	return err
}

func (r *OrderRepository) UpdateStatusIfCurrent(ctx context.Context, id, status string, currentStatuses []string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = ANY($3)
	`, status, id, pq.Array(currentStatuses))
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *OrderRepository) CreatePayment(ctx context.Context, payment *model.Payment) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO payments (id, order_id, transaction_id, method, amount, status, created_at)
		VALUES (:id, :order_id, :transaction_id, :method, :amount, :status, :created_at)
	`, payment)
	return err
}

func (r *OrderRepository) UpdatePaymentStatus(ctx context.Context, orderID, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments SET status = $1, confirmed_at = NOW() WHERE order_id = $2
	`, status, orderID)
	return err
}

// ConfirmOrderTx marks seats as sold, creates a payment record, and updates order status in a single transaction.
func (r *OrderRepository) ConfirmOrderTx(ctx context.Context, orderID, eventID string, seatIDs []string, payment *model.Payment) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Mark seats as sold
	_, err = tx.ExecContext(ctx, `
		UPDATE event_seats SET status = 'sold'
		WHERE event_id = $1 AND id = ANY($2)
	`, eventID, pq.Array(seatIDs))
	if err != nil {
		return err
	}

	// Create payment record
	_, err = tx.NamedExecContext(ctx, `
		INSERT INTO payments (id, order_id, transaction_id, method, amount, status, created_at)
		VALUES (:id, :order_id, :transaction_id, :method, :amount, :status, :created_at)
	`, payment)
	if err != nil {
		return err
	}

	// Update order status
	_, err = tx.ExecContext(ctx, `
		UPDATE orders SET status = 'confirmed', updated_at = NOW()
		WHERE id = $1
	`, orderID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ValidateCallbackToken checks if the given token matches the order's callback_token.
func (r *OrderRepository) ValidateCallbackToken(ctx context.Context, orderID, token string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM orders WHERE id = $1 AND callback_token = $2", orderID, token)
	return count > 0, err
}

// GetPendingOrdersNearExpiry returns pending orders created between 7.5 and 8.5 minutes ago (the 2-minute warning window).
func (r *OrderRepository) GetPendingOrdersNearExpiry(ctx context.Context) ([]model.Order, error) {
	var orders []model.Order
	err := r.db.SelectContext(ctx, &orders, `
		SELECT * FROM orders
		WHERE status = 'pending'
		AND created_at <= NOW() - INTERVAL '7 minutes 30 seconds'
		AND created_at > NOW() - INTERVAL '8 minutes 30 seconds'
	`)
	return orders, err
}

func (r *OrderRepository) GetExpiredPendingOrders(ctx context.Context) ([]model.Order, error) {
	var orders []model.Order
	err := r.db.SelectContext(ctx, &orders, `
		SELECT * FROM orders
		WHERE status = 'pending'
		AND created_at <= NOW() - INTERVAL '10 minutes'
	`)
	return orders, err
}
