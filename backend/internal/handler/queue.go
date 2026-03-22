package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ticketing-system/backend/internal/service"
)

type QueueHandler struct {
	svc *service.QueueService
}

func NewQueueHandler(svc *service.QueueService) *QueueHandler {
	return &QueueHandler{svc: svc}
}

func (h *QueueHandler) JoinQueue(c *gin.Context) {
	eventID := c.Param("id")
	userID := c.GetString("user_id")

	position, err := h.svc.JoinQueue(c.Request.Context(), eventID, userID)
	if err != nil {
		if err == service.ErrAlreadyInQueue {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加入排隊失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"position":       position,
		"estimated_wait": h.svc.EstimateWait(position),
	})
}

func (h *QueueHandler) GetPosition(c *gin.Context) {
	eventID := c.Param("id")
	userID := c.GetString("user_id")

	position, err := h.svc.GetPosition(c.Request.Context(), eventID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未在排隊中"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"position":       position,
		"estimated_wait": h.svc.EstimateWait(position),
	})
}
