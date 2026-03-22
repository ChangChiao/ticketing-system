package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
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

func (r *SeatRepository) GetAvailableSeatsInSection(ctx context.Context, eventID, sectionID string) ([]RowWithSeats, error) {
	query := `
		SELECT
			rw.id as row_id,
			rw.label as row_label,
			rw.sort_order,
			se.id as seat_id,
			se.number
		FROM rows rw
		JOIN seats se ON se.row_id = rw.id
		LEFT JOIN event_seats evt ON evt.seat_id = se.id AND evt.event_id = $1
		WHERE rw.section_id = $2
			AND (evt.status IS NULL OR evt.status = 'available')
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
		WHERE event_id = $1 AND seat_id = ANY($2)
	`
	_, err := r.db.ExecContext(ctx, query, eventID, seatIDs)
	return err
}

func (r *SeatRepository) ReleaseSeats(ctx context.Context, eventID string, seatIDs []string) error {
	query := `
		UPDATE event_seats
		SET status = 'available', locked_by = NULL, locked_at = NULL
		WHERE event_id = $1 AND seat_id = ANY($2)
	`
	_, err := r.db.ExecContext(ctx, query, eventID, seatIDs)
	return err
}
