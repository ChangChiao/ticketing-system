package model

import (
	"encoding/json"
	"time"
)

type Venue struct {
	ID         string          `db:"id" json:"id"`
	Name       string          `db:"name" json:"name"`
	Address    string          `db:"address" json:"address"`
	LayoutData json.RawMessage `db:"layout_data" json:"layout_data"`
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`
}

type Section struct {
	ID        string          `db:"id" json:"id"`
	VenueID   string          `db:"venue_id" json:"venue_id"`
	Name      string          `db:"name" json:"name"`
	Capacity  int             `db:"capacity" json:"capacity"`
	Polygon   json.RawMessage `db:"polygon" json:"polygon"`
	SortOrder int             `db:"sort_order" json:"sort_order"`
}

type Row struct {
	ID        string `db:"id" json:"id"`
	SectionID string `db:"section_id" json:"section_id"`
	Label     string `db:"label" json:"label"`
	SortOrder int    `db:"sort_order" json:"sort_order"`
}

type Seat struct {
	ID     string `db:"id" json:"id"`
	RowID  string `db:"row_id" json:"row_id"`
	Number int    `db:"number" json:"number"`
}

type Event struct {
	ID        string    `db:"id" json:"id"`
	VenueID   string    `db:"venue_id" json:"venue_id"`
	Title     string    `db:"title" json:"title"`
	EventDate time.Time `db:"event_date" json:"event_date"`
	SaleStart time.Time `db:"sale_start" json:"sale_start"`
	SaleEnd   time.Time `db:"sale_end" json:"sale_end"`
	Status    string    `db:"status" json:"status"` // draft, on_sale, sold_out, ended
	ImageURL  string    `db:"image_url" json:"image_url"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type EventSection struct {
	ID        string `db:"id" json:"id"`
	EventID   string `db:"event_id" json:"event_id"`
	SectionID string `db:"section_id" json:"section_id"`
	Price     int    `db:"price" json:"price"` // in TWD
	Quota     int    `db:"quota" json:"quota"`
}

type EventSeat struct {
	ID       string     `db:"id" json:"id"`
	EventID  string     `db:"event_id" json:"event_id"`
	SeatID   string     `db:"seat_id" json:"seat_id"`
	Status   string     `db:"status" json:"status"` // available, locked, sold
	LockedBy *string    `db:"locked_by" json:"locked_by"`
	LockedAt *time.Time `db:"locked_at" json:"locked_at"`
}

type User struct {
	ID           string    `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Name         string    `db:"name" json:"name"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type Order struct {
	ID        string    `db:"id" json:"id"`
	UserID    string    `db:"user_id" json:"user_id"`
	EventID   string    `db:"event_id" json:"event_id"`
	Status    string    `db:"status" json:"status"` // pending, confirmed, cancelled, payment_pending
	Total     int       `db:"total" json:"total"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type OrderItem struct {
	ID          string `db:"id" json:"id"`
	OrderID     string `db:"order_id" json:"order_id"`
	EventSeatID string `db:"event_seat_id" json:"event_seat_id"`
	SectionName string `db:"section_name" json:"section_name"`
	RowLabel    string `db:"row_label" json:"row_label"`
	SeatNumber  int    `db:"seat_number" json:"seat_number"`
	Price       int    `db:"price" json:"price"`
}

type Payment struct {
	ID            string    `db:"id" json:"id"`
	OrderID       string    `db:"order_id" json:"order_id"`
	TransactionID string    `db:"transaction_id" json:"transaction_id"`
	Method        string    `db:"method" json:"method"` // linepay
	Amount        int       `db:"amount" json:"amount"`
	Status        string    `db:"status" json:"status"` // pending, confirmed, failed
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	ConfirmedAt   *time.Time `db:"confirmed_at" json:"confirmed_at"`
}

// API response types

type EventListItem struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	EventDate  time.Time `json:"event_date"`
	VenueName  string    `json:"venue_name"`
	PriceRange string    `json:"price_range"`
	SaleStatus string    `json:"sale_status"`
	SaleStart  time.Time `json:"sale_start"`
	ImageURL   string    `json:"image_url"`
}

type EventDetail struct {
	Event
	VenueName  string               `json:"venue_name"`
	LayoutData json.RawMessage      `json:"layout_data"`
	Sections   []EventSectionDetail `json:"sections"`
}

type EventSectionDetail struct {
	EventSection
	SectionName string `json:"section_name"`
	Polygon     json.RawMessage `json:"polygon"`
	Remaining   int    `json:"remaining"`
}

type SectionAvailability struct {
	SectionID string `json:"section_id"`
	Name      string `json:"name"`
	Remaining int    `json:"remaining"`
	Quota     int    `json:"quota"`
}

type AllocatedSeats struct {
	SessionID string     `json:"session_id"`
	Seats     []SeatInfo `json:"seats"`
	ExpiresAt time.Time  `json:"expires_at"`
}

type SeatInfo struct {
	EventSeatID string `json:"event_seat_id"`
	SectionName string `json:"section_name"`
	RowLabel    string `json:"row_label"`
	SeatNumber  int    `json:"seat_number"`
}
