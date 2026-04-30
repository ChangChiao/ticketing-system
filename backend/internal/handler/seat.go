package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	eventID := c.Param("id")
	userID := c.GetString("user_id")

	if h.queueSvc != nil {
		active, err := h.queueSvc.IsSelectionActive(c.Request.Context(), eventID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法驗證排隊狀態"})
			return
		}
		if !active {
			c.JSON(http.StatusForbidden, gin.H{"error": "尚未輪到您選位，請回到排隊頁面"})
			return
		}
	}

	var req struct {
		SectionID string `json:"section_id" binding:"required"`
		Quantity  int    `json:"quantity" binding:"required,min=1,max=4"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請選擇區域和張數 (1-4張)"})
		return
	}

	result, err := h.svc.AllocateSeats(c.Request.Context(), eventID, req.SectionID, userID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if h.queueSvc != nil {
		_ = h.queueSvc.EndSelection(c.Request.Context(), eventID, userID)
	}

	c.JSON(http.StatusOK, result)
}
