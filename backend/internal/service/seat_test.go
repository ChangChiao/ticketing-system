package service

import (
	"testing"

	"github.com/ticketing-system/backend/internal/repository"
)

func TestFindConsecutiveInRow_Basic(t *testing.T) {
	seats := []repository.RowWithSeats{
		{SeatID: "s1", Number: 1},
		{SeatID: "s2", Number: 2},
		{SeatID: "s3", Number: 3},
		{SeatID: "s4", Number: 4},
		{SeatID: "s5", Number: 5},
	}

	result := findConsecutiveInRow(seats, 3)
	if result == nil {
		t.Fatal("expected 3 consecutive seats, got nil")
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 seats, got %d", len(result))
	}
	if result[0].Number != 1 || result[1].Number != 2 || result[2].Number != 3 {
		t.Errorf("expected seats 1,2,3 got %d,%d,%d", result[0].Number, result[1].Number, result[2].Number)
	}
}

func TestFindConsecutiveInRow_WithGaps(t *testing.T) {
	seats := []repository.RowWithSeats{
		{SeatID: "s1", Number: 1},
		{SeatID: "s3", Number: 3},
		{SeatID: "s4", Number: 4},
		{SeatID: "s5", Number: 5},
		{SeatID: "s6", Number: 6},
	}

	result := findConsecutiveInRow(seats, 3)
	if result == nil {
		t.Fatal("expected 3 consecutive seats, got nil")
	}
	if result[0].Number != 3 || result[1].Number != 4 || result[2].Number != 5 {
		t.Errorf("expected seats 3,4,5 got %d,%d,%d", result[0].Number, result[1].Number, result[2].Number)
	}
}

func TestFindConsecutiveInRow_NotEnough(t *testing.T) {
	seats := []repository.RowWithSeats{
		{SeatID: "s1", Number: 1},
		{SeatID: "s3", Number: 3},
		{SeatID: "s5", Number: 5},
	}

	result := findConsecutiveInRow(seats, 3)
	if result != nil {
		t.Fatal("expected nil, got seats")
	}
}

func TestFindConsecutiveInRow_ExactFit(t *testing.T) {
	seats := []repository.RowWithSeats{
		{SeatID: "s10", Number: 10},
		{SeatID: "s11", Number: 11},
	}

	result := findConsecutiveInRow(seats, 2)
	if result == nil {
		t.Fatal("expected 2 consecutive seats")
	}
	if result[0].Number != 10 || result[1].Number != 11 {
		t.Errorf("expected seats 10,11")
	}
}

func TestFindConsecutiveInRow_SingleSeat(t *testing.T) {
	seats := []repository.RowWithSeats{
		{SeatID: "s5", Number: 5},
	}

	result := findConsecutiveInRow(seats, 1)
	if result == nil || len(result) != 1 {
		t.Fatal("expected 1 seat")
	}
}

func TestFindConsecutiveInRow_Empty(t *testing.T) {
	result := findConsecutiveInRow(nil, 1)
	if result != nil {
		t.Fatal("expected nil for empty input")
	}
}

func TestAbs(t *testing.T) {
	if abs(-5) != 5 {
		t.Error("abs(-5) should be 5")
	}
	if abs(5) != 5 {
		t.Error("abs(5) should be 5")
	}
	if abs(0) != 0 {
		t.Error("abs(0) should be 0")
	}
}
