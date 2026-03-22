package redis

import (
	"context"
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

// Queue operations
func (c *Client) QueueJoin(ctx context.Context, eventID, userToken string) error {
	key := fmt.Sprintf("queue:%s", eventID)
	score := float64(time.Now().UnixMicro())
	return c.rdb.ZAdd(ctx, key, goredis.Z{Score: score, Member: userToken}).Err()
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

// Session tracking
func (c *Client) SetActiveSession(ctx context.Context, eventID, userID, sessionID string) (bool, error) {
	key := fmt.Sprintf("active_session:%s:%s", eventID, userID)
	return c.rdb.SetNX(ctx, key, sessionID, 15*time.Minute).Result()
}

func (c *Client) RemoveActiveSession(ctx context.Context, eventID, userID string) error {
	key := fmt.Sprintf("active_session:%s:%s", eventID, userID)
	return c.rdb.Del(ctx, key).Err()
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
