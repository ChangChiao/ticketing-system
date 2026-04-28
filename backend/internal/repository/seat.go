package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/ticketing-system/backend/internal/model"
)

type SeatRepository struct {
	db *sqlx.DB
}

func NewSeatRepository(db *sqlx.DB) *SeatRepository {
	return &SeatRepository{db: db}
}

func (r *SeatRepository) GetAvailability(ctx context.Context, eventID string) ([]model.SectionAvailability, error) {
	query := `
		SELECT
			s.id as section_id,
			s.name,
			es.quota,
			COALESCE(es.quota - COUNT(CASE WHEN evt.status IN ('locked','sold') THEN 1 END), es.quota) as remaining
		FROM event_sections es
		JOIN sections s ON s.id = es.section_id
		LEFT JOIN event_seats evt ON evt.event_id = es.event_id
			AND evt.seat_id IN (SELECT se.id FROM seats se JOIN rows r ON r.id = se.row_id WHERE r.section_id = s.id)
		WHERE es.event_id = $1
		GROUP BY s.id, s.name, es.quota
		ORDER BY s.sort_order
	`
	var result []model.SectionAvailability
	err := r.db.SelectContext(ctx, &result, query, eventID)
	return result, err
}

type RowWithSeats struct {
	RowID     string `db:"row_id"`
	RowLabel  string `db:"row_label"`
	SortOrder int    `db:"sort_order"`
	SeatID    string `db:"seat_id"`
	Number    int    `db:"number"`
}

type LockedSeatForOrder struct {
	EventSeatID string `db:"event_seat_id"`
	SectionName string `db:"section_name"`
	RowLabel    string `db:"row_label"`
	SeatNumber  int    `db:"seat_number"`
	Price       int    `db:"price"`
}

func (r *SeatRepository) GetAvailableSeatsInSection(ctx context.Context, eventID, sectionID string) ([]RowWithSeats, error) {
	query := `
		SELECT
			rw.id as row_id,
			rw.label as row_label,
			rw.sort_order,
			evt.id as seat_id,
			se.number
		FROM rows rw
		JOIN seats se ON se.row_id = rw.id
		JOIN event_seats evt ON evt.seat_id = se.id AND evt.event_id = $1
		WHERE rw.section_id = $2
			AND evt.status = 'available'
		ORDER BY rw.sort_order, se.number
	`
	var seats []RowWithSeats
	err := r.db.SelectContext(ctx, &seats, query, eventID, sectionID)
	return seats, err
}

func (r *SeatRepository) GetSectionInfo(ctx context.Context, eventID, sectionID string) (*model.EventSection, string, error) {
	var result struct {
		model.EventSection
		SectionName string `db:"section_name"`
	}
	query := `
		SELECT es.*, s.name as section_name
		FROM event_sections es
		JOIN sections s ON s.id = es.section_id
		WHERE es.event_id = $1 AND es.section_id = $2
	`
	err := r.db.GetContext(ctx, &result, query, eventID, sectionID)
	if err != nil {
		return nil, "", err
	}
	return &result.EventSection, result.SectionName, nil
}

func (r *SeatRepository) MarkSeatsAsSold(ctx context.Context, eventID string, seatIDs []string) error {
	query := `
		UPDATE event_seats
		SET status = 'sold'
		WHERE event_id = $1 AND id = ANY($2)
	`
	_, err := r.db.ExecContext(ctx, query, eventID, pq.Array(seatIDs))
	return err
}

func (r *SeatRepository) MarkSeatsAsLocked(ctx context.Context, eventID string, eventSeatIDs []string, userID string) error {
	query := `
		UPDATE event_seats
		SET status = 'locked', locked_by = $3, locked_at = NOW()
		WHERE event_id = $1 AND id = ANY($2) AND status = 'available'
	`
	result, err := r.db.ExecContext(ctx, query, eventID, pq.Array(eventSeatIDs), userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != int64(len(eventSeatIDs)) {
		return fmt.Errorf("locked %d of %d requested seats", affected, len(eventSeatIDs))
	}
	return nil
}

func (r *SeatRepository) ReleaseSeats(ctx context.Context, eventID string, seatIDs []string) error {
	query := `
		UPDATE event_seats
		SET status = 'available', locked_by = NULL, locked_at = NULL
		WHERE event_id = $1 AND id = ANY($2) AND status = 'locked'
	`
	_, err := r.db.ExecContext(ctx, query, eventID, pq.Array(seatIDs))
	return err
}

func (r *SeatRepository) ReleaseExpiredLocks(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE event_seats
		SET status = 'available', locked_by = NULL, locked_at = NULL
		WHERE status = 'locked'
			AND locked_at < NOW() - INTERVAL '10 minutes'
	`)
	return err
}

func (r *SeatRepository) GetLockedSeatsForOrder(ctx context.Context, eventID, userID string, eventSeatIDs []string) ([]LockedSeatForOrder, error) {
	var seats []LockedSeatForOrder
	err := r.db.SelectContext(ctx, &seats, `
		SELECT
			evt.id AS event_seat_id,
			s.name AS section_name,
			rw.label AS row_label,
			se.number AS seat_number,
			es.price AS price
		FROM event_seats evt
		JOIN seats se ON se.id = evt.seat_id
		JOIN rows rw ON rw.id = se.row_id
		JOIN sections s ON s.id = rw.section_id
		JOIN event_sections es ON es.event_id = evt.event_id AND es.section_id = s.id
		WHERE evt.event_id = $1
			AND evt.id = ANY($2)
			AND evt.status = 'locked'
			AND evt.locked_by = $3
		ORDER BY s.sort_order, rw.sort_order, se.number
	`, eventID, pq.Array(eventSeatIDs), userID)
	return seats, err
}
