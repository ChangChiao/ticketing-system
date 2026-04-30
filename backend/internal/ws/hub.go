package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/ticketing-system/backend/internal/middleware"
	pkgredis "github.com/ticketing-system/backend/pkg/redis"
)

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type QueueUpdate struct {
	Position      int64  `json:"position"`
	EstimatedWait string `json:"estimated_wait"`
	Status        string `json:"status"` // waiting, your_turn
}

type AvailabilityUpdate struct {
	SectionID string `json:"section_id"`
	Remaining int    `json:"remaining"`
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string
	rooms  map[string]bool // event IDs
}

type Hub struct {
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool // room -> clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan RoomMessage
	mu         sync.RWMutex
	upgrader   websocket.Upgrader
	jwtSecret  string
	redis      *pkgredis.Client
}

type RoomMessage struct {
	Room    string
	Message []byte
}

func NewHub(allowedOrigin, jwtSecret string) *Hub {
	// Parse the allowed origin to extract scheme + host for comparison
	parsed, _ := url.Parse(allowedOrigin)
	allowedHost := ""
	if parsed != nil {
		allowedHost = parsed.Scheme + "://" + parsed.Host
	}

	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan RoomMessage, 256),
		jwtSecret:  jwtSecret,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return false
				}
				return origin == allowedHost
			},
		},
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			middleware.ActiveWebSocketConnections.Inc()
			for room := range client.rooms {
				if h.rooms[room] == nil {
					h.rooms[room] = make(map[*Client]bool)
				}
				h.rooms[room][client] = true
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				middleware.ActiveWebSocketConnections.Dec()
				for room := range client.rooms {
					delete(h.rooms[room], client)
				}
				close(client.send)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.rooms[msg.Room]; ok {
				for client := range clients {
					select {
					case client.send <- msg.Message:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastToRoom(room string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws broadcast marshal error: %v", err)
		return
	}
	h.broadcast <- RoomMessage{Room: room, Message: data}
}

func (h *Hub) SendToUser(userID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID, err := h.userIDFromToken(r.URL.Query().Get("token"))
	if err != nil {
		http.Error(w, "invalid websocket token", http.StatusUnauthorized)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	eventID := r.URL.Query().Get("event_id")
	if eventID == "" {
		conn.Close()
		return
	}

	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
		rooms:  map[string]bool{"queue:" + eventID: true, "availability:" + eventID: true},
	}

	h.register <- client
	h.sendInitialQueueState(r.Context(), client, eventID)

	go client.writePump()
	go client.readPump()
}

func (h *Hub) userIDFromToken(rawToken string) (string, error) {
	if rawToken == "" {
		return "", nil
	}

	token, err := jwt.Parse(rawToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", jwt.ErrTokenInvalidClaims
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", jwt.ErrTokenInvalidClaims
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return "", jwt.ErrTokenInvalidClaims
	}

	return userID, nil
}

// SubscribeRedis listens to Redis Pub/Sub channels and bridges messages to WebSocket rooms.
func (h *Hub) SubscribeRedis(redisClient *pkgredis.Client) {
	h.redis = redisClient
	ctx := context.Background()

	// Availability updates
	go func() {
		sub := redisClient.SubscribeAvailability(ctx)
		defer sub.Close()
		ch := sub.Channel()
		for msg := range ch {
			var update pkgredis.AvailabilityMessage
			if err := json.Unmarshal([]byte(msg.Payload), &update); err != nil {
				log.Printf("ws: failed to unmarshal availability update: %v", err)
				continue
			}
			room := fmt.Sprintf("availability:%s", update.EventID)
			h.BroadcastToRoom(room, Message{
				Type: "availability_update",
				Data: AvailabilityUpdate{
					SectionID: update.SectionID,
					Remaining: update.Remaining,
				},
			})
		}
	}()

	// Payment countdown warnings
	go func() {
		sub := redisClient.SubscribePaymentWarning(ctx)
		defer sub.Close()
		ch := sub.Channel()
		for msg := range ch {
			var warning pkgredis.PaymentWarningMessage
			if err := json.Unmarshal([]byte(msg.Payload), &warning); err != nil {
				log.Printf("ws: failed to unmarshal payment warning: %v", err)
				continue
			}
			h.SendToUser(warning.UserID, Message{
				Type: "payment_warning",
				Data: map[string]string{
					"order_id": warning.OrderID,
					"type":     warning.Type,
					"message":  "付款倒數剩餘 2 分鐘",
				},
			})
		}
	}()
}

func (h *Hub) sendInitialQueueState(ctx context.Context, client *Client, eventID string) {
	if h.redis == nil || client.userID == "" {
		return
	}

	if admitted, err := h.redis.HasQueueAdmission(ctx, eventID, client.userID); err == nil && admitted {
		h.sendToClient(client, Message{
			Type: "queue_update",
			Data: map[string]interface{}{
				"event_id":       eventID,
				"position":       0,
				"estimated_wait": "即將輪到您",
				"status":         "your_turn",
				"entry_window":   60,
			},
		})
		return
	}

	if active, err := h.redis.HasSelectionSession(ctx, eventID, client.userID); err == nil && active {
		h.sendToClient(client, Message{
			Type: "queue_update",
			Data: map[string]interface{}{
				"event_id":       eventID,
				"position":       0,
				"estimated_wait": "即將輪到您",
				"status":         "your_turn",
			},
		})
		return
	}

	position, err := h.redis.QueuePosition(ctx, eventID, client.userID)
	if err != nil {
		return
	}
	total, _ := h.redis.QueueSize(ctx, eventID)
	h.sendToClient(client, Message{
		Type: "queue_update",
		Data: map[string]interface{}{
			"event_id":       eventID,
			"position":       position,
			"total_in_queue": total,
			"estimated_wait": estimateWait(position),
			"status":         "waiting",
		},
	})
}

func (h *Hub) sendToClient(client *Client, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case client.send <- data:
	default:
	}
}

func estimateWait(position int64) string {
	if position <= 0 {
		return "即將輪到您"
	}
	const batchSize int64 = 50
	const batchIntervalS int64 = 5
	seconds := (position / batchSize) * batchIntervalS
	if seconds < 60 {
		return fmt.Sprintf("約 %d 秒", seconds)
	}
	return fmt.Sprintf("約 %d 分鐘", seconds/60+1)
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			c.conn.WriteMessage(websocket.TextMessage, message)
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
