package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const SeatLockTTL = 10 * time.Minute

type Client struct {
	rdb *goredis.Client
}

func NewClient(rdb *goredis.Client) *Client {
	return &Client{rdb: rdb}
}

func (c *Client) Raw() *goredis.Client {
	return c.rdb
}

// Seat locking with Lua script for atomic multi-seat lock
var lockSeatsScript = goredis.NewScript(`
	for i, key in ipairs(KEYS) do
		if redis.call('EXISTS', key) == 1 then
			return 0
		end
	end
	for i, key in ipairs(KEYS) do
		redis.call('SET', key, ARGV[1], 'EX', ARGV[2])
	end
	return 1
`)

func (c *Client) LockSeats(ctx context.Context, eventID string, seatIDs []string, sessionID string) (bool, error) {
	keys := make([]string, len(seatIDs))
	for i, seatID := range seatIDs {
		keys[i] = fmt.Sprintf("seat_lock:%s:%s", eventID, seatID)
	}
	result, err := lockSeatsScript.Run(ctx, c.rdb, keys, sessionID, int(SeatLockTTL.Seconds())).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (c *Client) UnlockSeats(ctx context.Context, eventID string, seatIDs []string) error {
	keys := make([]string, len(seatIDs))
	for i, seatID := range seatIDs {
		keys[i] = fmt.Sprintf("seat_lock:%s:%s", eventID, seatID)
	}
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) IsSeatLocked(ctx context.Context, eventID, seatID string) (bool, error) {
	key := fmt.Sprintf("seat_lock:%s:%s", eventID, seatID)
	result, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// AreSeatsLocked checks lock status for multiple seats in a single Redis pipeline call.
// Returns a slice of booleans in the same order as seatIDs.
func (c *Client) AreSeatsLocked(ctx context.Context, eventID string, seatIDs []string) ([]bool, error) {
	pipe := c.rdb.Pipeline()
	cmds := make([]*goredis.IntCmd, len(seatIDs))
	for i, seatID := range seatIDs {
		key := fmt.Sprintf("seat_lock:%s:%s", eventID, seatID)
		cmds[i] = pipe.Exists(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]bool, len(seatIDs))
	for i, cmd := range cmds {
		results[i] = cmd.Val() > 0
	}
	return results, nil
}

// Queue operations
func (c *Client) QueueJoin(ctx context.Context, eventID, userToken string) error {
	key := fmt.Sprintf("queue:%s", eventID)
	score := float64(time.Now().UnixMicro())
	pipe := c.rdb.TxPipeline()
	pipe.ZAdd(ctx, key, goredis.Z{Score: score, Member: userToken})
	pipe.SAdd(ctx, "queue_events", eventID)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *Client) QueuePosition(ctx context.Context, eventID, userToken string) (int64, error) {
	key := fmt.Sprintf("queue:%s", eventID)
	rank, err := c.rdb.ZRank(ctx, key, userToken).Result()
	if err != nil {
		return -1, err
	}
	return rank, nil
}

func (c *Client) QueuePop(ctx context.Context, eventID string, count int64) ([]string, error) {
	key := fmt.Sprintf("queue:%s", eventID)
	result, err := c.rdb.ZPopMin(ctx, key, count).Result()
	if err != nil {
		return nil, err
	}
	tokens := make([]string, len(result))
	for i, z := range result {
		tokens[i] = z.Member.(string)
	}
	return tokens, nil
}

func (c *Client) QueueSize(ctx context.Context, eventID string) (int64, error) {
	key := fmt.Sprintf("queue:%s", eventID)
	return c.rdb.ZCard(ctx, key).Result()
}

func (c *Client) QueueMembers(ctx context.Context, eventID string) ([]string, error) {
	key := fmt.Sprintf("queue:%s", eventID)
	return c.rdb.ZRange(ctx, key, 0, -1).Result()
}

func (c *Client) QueueEventIDs(ctx context.Context) ([]string, error) {
	return c.rdb.SMembers(ctx, "queue_events").Result()
}

// Session tracking
func (c *Client) SetActiveSession(ctx context.Context, eventID, userID, sessionID string) (bool, error) {
	key := fmt.Sprintf("active_session:%s:%s", eventID, userID)
	return c.rdb.SetNX(ctx, key, sessionID, 15*time.Minute).Result()
}

func (c *Client) RemoveActiveSession(ctx context.Context, eventID, userID string) error {
	key := fmt.Sprintf("active_session:%s:%s", eventID, userID)
	return c.rdb.Del(ctx, key).Err()
}

func (c *Client) SetQueueAdmission(ctx context.Context, eventID, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("queue_admission:%s:%s", eventID, userID)
	return c.rdb.Set(ctx, key, "1", ttl).Err()
}

func (c *Client) HasQueueAdmission(ctx context.Context, eventID, userID string) (bool, error) {
	key := fmt.Sprintf("queue_admission:%s:%s", eventID, userID)
	result, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

var startSelectionSessionScript = goredis.NewScript(`
	local admissionKey = KEYS[1]
	local sessionKey = KEYS[2]
	local activeKey = KEYS[3]
	local userID = ARGV[1]
	local ttlSeconds = tonumber(ARGV[2])
	local expiresAt = tonumber(ARGV[3])

	if redis.call('EXISTS', sessionKey) == 0 and redis.call('EXISTS', admissionKey) == 0 then
		return 0
	end

	redis.call('DEL', admissionKey)
	redis.call('SET', sessionKey, '1', 'EX', ttlSeconds)
	redis.call('ZADD', activeKey, expiresAt, userID)
	return 1
`)

func (c *Client) StartSelectionSession(ctx context.Context, eventID, userID string, ttl time.Duration) (bool, error) {
	admissionKey := fmt.Sprintf("queue_admission:%s:%s", eventID, userID)
	sessionKey := fmt.Sprintf("selection_session:%s:%s", eventID, userID)
	activeKey := fmt.Sprintf("active_selection:%s", eventID)
	expiresAt := time.Now().Add(ttl).Unix()

	result, err := startSelectionSessionScript.Run(
		ctx,
		c.rdb,
		[]string{admissionKey, sessionKey, activeKey},
		userID,
		int(ttl.Seconds()),
		expiresAt,
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (c *Client) HasSelectionSession(ctx context.Context, eventID, userID string) (bool, error) {
	key := fmt.Sprintf("selection_session:%s:%s", eventID, userID)
	result, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (c *Client) EndSelectionSession(ctx context.Context, eventID, userID string) error {
	sessionKey := fmt.Sprintf("selection_session:%s:%s", eventID, userID)
	activeKey := fmt.Sprintf("active_selection:%s", eventID)
	pipe := c.rdb.TxPipeline()
	pipe.Del(ctx, sessionKey)
	pipe.ZRem(ctx, activeKey, userID)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *Client) PruneActiveSelections(ctx context.Context, eventID string) error {
	key := fmt.Sprintf("active_selection:%s", eventID)
	return c.rdb.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", time.Now().Unix())).Err()
}

func (c *Client) ActiveSelectionCount(ctx context.Context, eventID string) (int64, error) {
	key := fmt.Sprintf("active_selection:%s", eventID)
	return c.rdb.ZCard(ctx, key).Result()
}

// Availability cache
func (c *Client) SetSectionRemaining(ctx context.Context, eventID, sectionID string, remaining int) error {
	key := fmt.Sprintf("availability:%s:%s", eventID, sectionID)
	return c.rdb.Set(ctx, key, remaining, 5*time.Minute).Err()
}

func (c *Client) GetSectionRemaining(ctx context.Context, eventID, sectionID string) (int, error) {
	key := fmt.Sprintf("availability:%s:%s", eventID, sectionID)
	return c.rdb.Get(ctx, key).Int()
}

func (c *Client) DecrSectionRemaining(ctx context.Context, eventID, sectionID string, count int) error {
	key := fmt.Sprintf("availability:%s:%s", eventID, sectionID)
	return c.rdb.DecrBy(ctx, key, int64(count)).Err()
}

func (c *Client) IncrSectionRemaining(ctx context.Context, eventID, sectionID string, count int) error {
	key := fmt.Sprintf("availability:%s:%s", eventID, sectionID)
	return c.rdb.IncrBy(ctx, key, int64(count)).Err()
}

// Pub/Sub for real-time availability updates across pods
const AvailabilityChannel = "availability_updates"

type AvailabilityMessage struct {
	EventID   string `json:"event_id"`
	SectionID string `json:"section_id"`
	Remaining int    `json:"remaining"`
}

func (c *Client) PublishAvailability(ctx context.Context, eventID, sectionID string, remaining int) error {
	msg := AvailabilityMessage{
		EventID:   eventID,
		SectionID: sectionID,
		Remaining: remaining,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.rdb.Publish(ctx, AvailabilityChannel, data).Err()
}

func (c *Client) SubscribeAvailability(ctx context.Context) *goredis.PubSub {
	return c.rdb.Subscribe(ctx, AvailabilityChannel)
}

// Pub/Sub for payment countdown warnings
const PaymentWarningChannel = "payment_warnings"

type PaymentWarningMessage struct {
	UserID  string `json:"user_id"`
	OrderID string `json:"order_id"`
	EventID string `json:"event_id"`
	Type    string `json:"type"` // "two_min_warning"
}

func (c *Client) PublishPaymentWarning(ctx context.Context, msg PaymentWarningMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.rdb.Publish(ctx, PaymentWarningChannel, data).Err()
}

func (c *Client) SubscribePaymentWarning(ctx context.Context) *goredis.PubSub {
	return c.rdb.Subscribe(ctx, PaymentWarningChannel)
}

// Rate limiting with fixed-window counter (atomic via Lua script)
var rateLimitScript = goredis.NewScript(`
	local key = KEYS[1]
	local max = tonumber(ARGV[1])
	local window = tonumber(ARGV[2])
	local current = redis.call('INCR', key)
	if current == 1 then
		redis.call('EXPIRE', key, window)
	end
	if current > max then
		return 0
	end
	return 1
`)

// CheckRateLimit returns true if the request is allowed, false if rate limited.
// key: unique identifier (e.g. "rl:ip:1.2.3.4"), maxRequests: max count in window, window: time window duration.
func (c *Client) CheckRateLimit(ctx context.Context, key string, maxRequests int, window time.Duration) (bool, error) {
	windowSec := int(window.Seconds())
	if windowSec < 1 {
		windowSec = 1
	}
	result, err := rateLimitScript.Run(ctx, c.rdb, []string{key}, maxRequests, windowSec).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
