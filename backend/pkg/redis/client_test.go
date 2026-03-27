package redis

import (
	"context"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func getTestRedis(t *testing.T) (*Client, func()) {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	rdb := goredis.NewClient(&goredis.Options{Addr: addr})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at %s: %v", addr, err)
	}
	client := NewClient(rdb)
	cleanup := func() {
		rdb.FlushDB(ctx)
		rdb.Close()
	}
	return client, cleanup
}

func TestLockSeats_AllAvailable(t *testing.T) {
	client, cleanup := getTestRedis(t)
	defer cleanup()
	ctx := context.Background()

	locked, err := client.LockSeats(ctx, "evt1", []string{"seat1", "seat2", "seat3"}, "session1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !locked {
		t.Fatal("expected lock to succeed")
	}

	// Verify all keys exist
	for _, seatID := range []string{"seat1", "seat2", "seat3"} {
		exists, err := client.IsSeatLocked(ctx, "evt1", seatID)
		if err != nil {
			t.Fatalf("unexpected error checking lock: %v", err)
		}
		if !exists {
			t.Errorf("expected seat %s to be locked", seatID)
		}
	}
}

func TestLockSeats_AlreadyLocked(t *testing.T) {
	client, cleanup := getTestRedis(t)
	defer cleanup()
	ctx := context.Background()

	// First lock succeeds
	locked, err := client.LockSeats(ctx, "evt1", []string{"seat1", "seat2"}, "session1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !locked {
		t.Fatal("expected first lock to succeed")
	}

	// Second lock fails (seat1 already locked)
	locked, err = client.LockSeats(ctx, "evt1", []string{"seat1", "seat3"}, "session2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if locked {
		t.Fatal("expected second lock to fail due to conflict")
	}

	// seat3 should NOT be locked (atomic all-or-nothing)
	exists, _ := client.IsSeatLocked(ctx, "evt1", "seat3")
	if exists {
		t.Error("seat3 should not be locked after failed atomic lock")
	}
}

func TestLockSeats_TTLExpiry(t *testing.T) {
	client, cleanup := getTestRedis(t)
	defer cleanup()
	ctx := context.Background()

	// Lock with a short TTL by setting key directly
	key := "seat_lock:evt1:seat_ttl"
	client.rdb.Set(ctx, key, "session1", 1*time.Second)

	// Initially locked
	exists, _ := client.IsSeatLocked(ctx, "evt1", "seat_ttl")
	if !exists {
		t.Fatal("expected seat to be locked initially")
	}

	// Wait for TTL
	time.Sleep(1100 * time.Millisecond)

	exists, _ = client.IsSeatLocked(ctx, "evt1", "seat_ttl")
	if exists {
		t.Fatal("expected seat lock to expire after TTL")
	}
}

func TestLockSeats_ConcurrentAtomicity(t *testing.T) {
	client, cleanup := getTestRedis(t)
	defer cleanup()
	ctx := context.Background()

	// Simulate two concurrent lock attempts on overlapping seats
	seats1 := []string{"seat_a", "seat_b", "seat_c"}
	seats2 := []string{"seat_b", "seat_c", "seat_d"}

	locked1, _ := client.LockSeats(ctx, "evt1", seats1, "session1")
	locked2, _ := client.LockSeats(ctx, "evt1", seats2, "session2")

	// Exactly one should succeed
	if locked1 == locked2 {
		t.Fatalf("expected exactly one lock to succeed: locked1=%v, locked2=%v", locked1, locked2)
	}

	// The successful session should hold all its seats
	if locked1 {
		for _, s := range seats1 {
			exists, _ := client.IsSeatLocked(ctx, "evt1", s)
			if !exists {
				t.Errorf("session1 won but seat %s not locked", s)
			}
		}
	}
}

func TestUnlockSeats(t *testing.T) {
	client, cleanup := getTestRedis(t)
	defer cleanup()
	ctx := context.Background()

	client.LockSeats(ctx, "evt1", []string{"seat1", "seat2"}, "session1")
	client.UnlockSeats(ctx, "evt1", []string{"seat1", "seat2"})

	for _, seatID := range []string{"seat1", "seat2"} {
		exists, _ := client.IsSeatLocked(ctx, "evt1", seatID)
		if exists {
			t.Errorf("seat %s should be unlocked", seatID)
		}
	}
}

func TestQueueOperations(t *testing.T) {
	client, cleanup := getTestRedis(t)
	defer cleanup()
	ctx := context.Background()

	// Join queue
	if err := client.QueueJoin(ctx, "evt1", "user1"); err != nil {
		t.Fatalf("queue join failed: %v", err)
	}
	time.Sleep(1 * time.Millisecond) // ensure different timestamps
	if err := client.QueueJoin(ctx, "evt1", "user2"); err != nil {
		t.Fatalf("queue join failed: %v", err)
	}

	// Check positions (FIFO order)
	pos1, _ := client.QueuePosition(ctx, "evt1", "user1")
	pos2, _ := client.QueuePosition(ctx, "evt1", "user2")
	if pos1 != 0 {
		t.Errorf("user1 should be at position 0, got %d", pos1)
	}
	if pos2 != 1 {
		t.Errorf("user2 should be at position 1, got %d", pos2)
	}

	// Queue size
	size, _ := client.QueueSize(ctx, "evt1")
	if size != 2 {
		t.Errorf("expected queue size 2, got %d", size)
	}

	// Pop first user
	popped, _ := client.QueuePop(ctx, "evt1", 1)
	if len(popped) != 1 || popped[0] != "user1" {
		t.Errorf("expected to pop user1, got %v", popped)
	}
}

func TestActiveSession(t *testing.T) {
	client, cleanup := getTestRedis(t)
	defer cleanup()
	ctx := context.Background()

	// First session succeeds
	ok, err := client.SetActiveSession(ctx, "evt1", "user1", "sess1")
	if err != nil || !ok {
		t.Fatal("expected first session to succeed")
	}

	// Second session fails (already active)
	ok, err = client.SetActiveSession(ctx, "evt1", "user1", "sess2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected second session to fail")
	}

	// Remove session
	client.RemoveActiveSession(ctx, "evt1", "user1")

	// Now new session succeeds
	ok, _ = client.SetActiveSession(ctx, "evt1", "user1", "sess3")
	if !ok {
		t.Fatal("expected new session to succeed after removal")
	}
}
