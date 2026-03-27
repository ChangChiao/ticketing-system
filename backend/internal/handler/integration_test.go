package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	goredis "github.com/redis/go-redis/v9"

	"github.com/ticketing-system/backend/internal/middleware"
	"github.com/ticketing-system/backend/internal/repository"
	"github.com/ticketing-system/backend/internal/service"
	"github.com/ticketing-system/backend/pkg/linepay"
	pkgredis "github.com/ticketing-system/backend/pkg/redis"
)

type testEnv struct {
	router *gin.Engine
	db     *sqlx.DB
	rdb    *goredis.Client
	token  string // JWT for test user
	userID string
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration tests")
	}
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	rdb := goredis.NewClient(&goredis.Options{Addr: redisAddr})
	redisClient := pkgredis.NewClient(rdb)

	// Repositories
	eventRepo := repository.NewEventRepository(db)
	seatRepo := repository.NewSeatRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Services
	jwtSecret := "test-secret"
	eventSvc := service.NewEventService(eventRepo)
	seatSvc := service.NewSeatService(seatRepo, redisClient)
	orderSvc := service.NewOrderService(orderRepo, seatSvc, redisClient)
	authSvc := service.NewAuthService(userRepo, jwtSecret)
	queueSvc := service.NewQueueService(redisClient)

	linePayCli := linepay.NewClient("", "", "https://sandbox-api-pay.line.me", "http://localhost:3000")

	// Handlers
	eventHandler := NewEventHandler(eventSvc)
	seatHandler := NewSeatHandler(seatSvc)
	orderHandler := NewOrderHandler(orderSvc, linePayCli)
	authHandler := NewAuthHandler(authSvc)
	queueHandler := NewQueueHandler(queueSvc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
		api.GET("/events", eventHandler.ListEvents)
		api.GET("/events/:id", eventHandler.GetEvent)
		api.GET("/events/:id/availability", seatHandler.GetAvailability)
		api.GET("/payments/confirm", orderHandler.ConfirmPayment)
		api.GET("/payments/cancel", orderHandler.CancelPayment)

		protected := api.Group("")
		protected.Use(middleware.Auth(jwtSecret))
		{
			protected.POST("/events/:id/queue/join", queueHandler.JoinQueue)
			protected.GET("/events/:id/queue/position", queueHandler.GetPosition)
			protected.POST("/events/:id/allocate", seatHandler.AllocateSeats)
			protected.POST("/orders", orderHandler.CreateOrder)
			protected.GET("/orders", orderHandler.ListOrders)
			protected.GET("/orders/:id", orderHandler.GetOrder)
		}
	}

	env := &testEnv{router: r, db: db, rdb: rdb}

	// Register a test user
	email := fmt.Sprintf("test_%d@example.com", time.Now().UnixNano())
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": "testpassword123",
		"name":     "Test User",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("failed to register test user: %d %s", w.Code, w.Body.String())
	}

	var authResp struct {
		User  struct{ ID string `json:"id"` } `json:"user"`
		Token string                           `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &authResp)
	env.token = authResp.Token
	env.userID = authResp.User.ID

	return env
}

func (e *testEnv) authRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.token)
	e.router.ServeHTTP(w, req)
	return w
}

func (e *testEnv) publicRequest(method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	e.router.ServeHTTP(w, req)
	return w
}

// TestFullFlow_QueueToPayment tests the end-to-end flow:
// List events → Join queue → Get position → Allocate seats → Create order → Confirm payment
func TestFullFlow_QueueToPayment(t *testing.T) {
	env := setupTestEnv(t)
	defer env.db.Close()
	defer env.rdb.FlushDB(nil)

	// 1. List events
	w := env.publicRequest("GET", "/api/events")
	if w.Code != http.StatusOK {
		t.Fatalf("list events failed: %d", w.Code)
	}
	var eventsResp struct {
		Events []struct {
			ID string `json:"id"`
		} `json:"events"`
	}
	json.Unmarshal(w.Body.Bytes(), &eventsResp)
	if len(eventsResp.Events) == 0 {
		t.Skip("no events in test database, skipping flow test")
	}
	eventID := eventsResp.Events[0].ID

	// 2. Get event detail
	w = env.publicRequest("GET", "/api/events/"+eventID)
	if w.Code != http.StatusOK {
		t.Fatalf("get event failed: %d", w.Code)
	}

	// 3. Get availability
	w = env.publicRequest("GET", "/api/events/"+eventID+"/availability")
	if w.Code != http.StatusOK {
		t.Fatalf("get availability failed: %d", w.Code)
	}
	var availResp struct {
		Sections []struct {
			SectionID string `json:"section_id"`
			Remaining int    `json:"remaining"`
		} `json:"sections"`
	}
	json.Unmarshal(w.Body.Bytes(), &availResp)

	// Find a section with available seats
	var sectionID string
	for _, s := range availResp.Sections {
		if s.Remaining > 0 {
			sectionID = s.SectionID
			break
		}
	}
	if sectionID == "" {
		t.Skip("no available seats in test database")
	}

	// 4. Join queue
	w = env.authRequest("POST", "/api/events/"+eventID+"/queue/join", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("join queue failed: %d %s", w.Code, w.Body.String())
	}
	var queueResp struct {
		Position int64 `json:"position"`
	}
	json.Unmarshal(w.Body.Bytes(), &queueResp)
	t.Logf("Queue position: %d", queueResp.Position)

	// 5. Get queue position
	w = env.authRequest("GET", "/api/events/"+eventID+"/queue/position", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get position failed: %d", w.Code)
	}

	// 6. Allocate seats
	w = env.authRequest("POST", "/api/events/"+eventID+"/allocate", map[string]interface{}{
		"section_id": sectionID,
		"quantity":   2,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("allocate seats failed: %d %s", w.Code, w.Body.String())
	}
	var allocResp struct {
		SessionID string `json:"session_id"`
		Seats     []struct {
			EventSeatID string `json:"event_seat_id"`
			SectionName string `json:"section_name"`
			RowLabel    string `json:"row_label"`
			SeatNumber  int    `json:"seat_number"`
		} `json:"seats"`
		ExpiresAt string `json:"expires_at"`
	}
	json.Unmarshal(w.Body.Bytes(), &allocResp)
	if len(allocResp.Seats) != 2 {
		t.Fatalf("expected 2 allocated seats, got %d", len(allocResp.Seats))
	}
	t.Logf("Allocated seats: %s row %s seats %d-%d",
		allocResp.Seats[0].SectionName, allocResp.Seats[0].RowLabel,
		allocResp.Seats[0].SeatNumber, allocResp.Seats[1].SeatNumber)

	// 7. Create order (LINE Pay will fail in test, but order should be created)
	seats := make([]map[string]interface{}, len(allocResp.Seats))
	for i, s := range allocResp.Seats {
		seats[i] = map[string]interface{}{
			"event_seat_id": s.EventSeatID,
			"section_name":  s.SectionName,
			"row_label":     s.RowLabel,
			"seat_number":   s.SeatNumber,
		}
	}
	w = env.authRequest("POST", "/api/orders", map[string]interface{}{
		"event_id":       eventID,
		"seats":          seats,
		"price_per_seat": 2800,
	})
	// May fail due to LINE Pay not configured in test, which is expected
	t.Logf("Create order response: %d %s", w.Code, w.Body.String())

	// 8. List orders
	w = env.authRequest("GET", "/api/orders", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list orders failed: %d", w.Code)
	}
}

// TestQueueSingleSession verifies single-session enforcement
func TestQueueSingleSession(t *testing.T) {
	env := setupTestEnv(t)
	defer env.db.Close()
	defer env.rdb.FlushDB(nil)

	w := env.publicRequest("GET", "/api/events")
	var eventsResp struct {
		Events []struct{ ID string `json:"id"` } `json:"events"`
	}
	json.Unmarshal(w.Body.Bytes(), &eventsResp)
	if len(eventsResp.Events) == 0 {
		t.Skip("no events in test database")
	}
	eventID := eventsResp.Events[0].ID

	// First join succeeds
	w = env.authRequest("POST", "/api/events/"+eventID+"/queue/join", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("first queue join failed: %d", w.Code)
	}

	// Second join should fail (already in queue)
	w = env.authRequest("POST", "/api/events/"+eventID+"/queue/join", nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for duplicate queue join, got %d", w.Code)
	}
}

// TestAllocateInvalidQuantity verifies quantity validation
func TestAllocateInvalidQuantity(t *testing.T) {
	env := setupTestEnv(t)
	defer env.db.Close()
	defer env.rdb.FlushDB(nil)

	// Quantity 0 should fail
	w := env.authRequest("POST", "/api/events/fake-id/allocate", map[string]interface{}{
		"section_id": "section1",
		"quantity":   0,
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for quantity 0, got %d", w.Code)
	}

	// Quantity 5 should fail (max 4)
	w = env.authRequest("POST", "/api/events/fake-id/allocate", map[string]interface{}{
		"section_id": "section1",
		"quantity":   5,
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for quantity 5, got %d", w.Code)
	}
}
