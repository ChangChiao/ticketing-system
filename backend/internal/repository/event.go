package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ticketing-system/backend/internal/model"
)

type EventRepository struct {
	db *sqlx.DB
}

func NewEventRepository(db *sqlx.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) ListEvents(ctx context.Context) ([]model.EventListItem, error) {
	query := `
		SELECT
			e.id, e.title, e.event_date, e.sale_start, e.status, e.image_url,
			v.name as venue_name,
			CONCAT(MIN(es.price), ' - ', MAX(es.price)) as price_range
		FROM events e
		JOIN venues v ON v.id = e.venue_id
		LEFT JOIN event_sections es ON es.event_id = e.id
		WHERE e.status != 'draft'
		GROUP BY e.id, v.name
		ORDER BY e.event_date ASC
	`
	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.EventListItem
	now := time.Now()
	for rows.Next() {
		var item struct {
			model.EventListItem
			Status string `db:"status"`
		}
		if err := rows.StructScan(&item); err != nil {
			return nil, err
		}
		item.EventListItem.SaleStatus = deriveSaleStatus(item.Status, item.SaleStart, now)
		events = append(events, item.EventListItem)
	}
	return events, nil
}

func (r *EventRepository) GetEvent(ctx context.Context, id string) (*model.EventDetail, error) {
	var event model.Event
	err := r.db.GetContext(ctx, &event, "SELECT * FROM events WHERE id = $1", id)
	if err != nil {
		return nil, err
	}

	var venue model.Venue
	err = r.db.GetContext(ctx, &venue, "SELECT * FROM venues WHERE id = $1", event.VenueID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT es.id, es.event_id, es.section_id, es.price, es.quota,
			s.name as section_name, s.polygon,
			COALESCE(es.quota - COUNT(CASE WHEN evt.status IN ('locked','sold') THEN 1 END), es.quota) as remaining
		FROM event_sections es
		JOIN sections s ON s.id = es.section_id
		LEFT JOIN event_seats evt ON evt.event_id = es.event_id
			AND evt.seat_id IN (SELECT se.id FROM seats se JOIN rows r ON r.id = se.row_id WHERE r.section_id = s.id)
		WHERE es.event_id = $1
		GROUP BY es.id, s.name, s.polygon
		ORDER BY s.sort_order
	`
	var sections []model.EventSectionDetail
	err = r.db.SelectContext(ctx, &sections, query, id)
	if err != nil {
		return nil, err
	}

	return &model.EventDetail{
		Event:      event,
		VenueName:  venue.Name,
		LayoutData: venue.LayoutData,
		Sections:   sections,
	}, nil
}

func deriveSaleStatus(status string, saleStart time.Time, now time.Time) string {
	switch status {
	case "sold_out":
		return "已售完"
	case "ended":
		return "已結束"
	default:
		if now.Before(saleStart) {
			return "即將開賣"
		}
		return "熱賣中"
	}
}
