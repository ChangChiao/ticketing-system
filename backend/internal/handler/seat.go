package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ticketing-system/backend/internal/middleware"
	"github.com/ticketing-system/backend/internal/service"
)

type SeatHandler struct {
	svc      *service.SeatService
	queueSvc *service.QueueService
}

func NewSeatHandler(svc *service.SeatService, queueSvc ...*service.QueueService) *SeatHandler {
	h := &SeatHandler{svc: svc}
	if len(queueSvc) > 0 {
		h.queueSvc = queueSvc[0]
	}
	return h
}

func (h *SeatHandler) GetAvailability(c *gin.Context) {
	eventID := c.Param("id")
	availability, err := h.svc.GetAvailability(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法取得剩餘票數"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sections": availability})
}

func (h *SeatHandler) AllocateSeats(c *gin.Context) {
	start := time.Now()
	eventID := c.Param("id")
	userID := c.GetString("user_id")
	resultLabel := "error"
	defer func() {
		middleware.SeatAllocationAttempts.WithLabelValues(eventID, resultLabel).Inc()
		middleware.SeatAllocationDuration.WithLabelValues(eventID).Observe(time.Since(start).Seconds())
	}()

	if h.queueSvc != nil {
		active, err := h.queueSvc.IsSelectionActive(c.Request.Context(), eventID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法驗證排隊狀態"})
			return
		}
		if !active {
			resultLabel = "conflict"
			c.JSON(http.StatusForbidden, gin.H{"error": "尚未輪到您選位，請回到排隊頁面"})
			return
		}
	}

	var req struct {
		SectionID string `json:"section_id" binding:"required"`
		Quantity  int    `json:"quantity" binding:"required,min=1,max=4"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resultLabel = "error"
		c.JSON(http.StatusBadRequest, gin.H{"error": "請選擇區域和張數 (1-4張)"})
		return
	}

	result, err := h.svc.AllocateSeats(c.Request.Context(), eventID, req.SectionID, userID, req.Quantity)
	if err != nil {
		resultLabel = "conflict"
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if h.queueSvc != nil {
		_ = h.queueSvc.EndSelection(c.Request.Context(), eventID, userID)
	}

	resultLabel = "success"
	c.JSON(http.StatusOK, result)
}

func (h *SeatHandler) ReleaseAllocation(c *gin.Context) {
	eventID := c.Param("id")
	userID := c.GetString("user_id")

	var req struct {
		Seats []struct {
			EventSeatID string `json:"event_seat_id" binding:"required"`
		} `json:"seats" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少要釋放的座位"})
		return
	}

	seatIDs := make([]string, 0, len(req.Seats))
	for _, seat := range req.Seats {
		seatIDs = append(seatIDs, seat.EventSeatID)
	}

	if err := h.svc.ReleaseLockedSeatsForUser(c.Request.Context(), eventID, userID, seatIDs); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "座位已失效或無法釋放"})
		return
	}

	if h.queueSvc != nil {
		_ = h.queueSvc.RestoreSelection(c.Request.Context(), eventID, userID)
	}
	c.JSON(http.StatusOK, gin.H{"status": "released"})
}
