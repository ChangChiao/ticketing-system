package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ticketing-system/backend/internal/middleware"
	"github.com/ticketing-system/backend/internal/service"
)

type AdminHandler struct {
	orderSvc *service.OrderService
}

func NewAdminHandler(orderSvc *service.OrderService) *AdminHandler {
	return &AdminHandler{orderSvc: orderSvc}
}

func (h *AdminHandler) ListPaymentPendingOrders(c *gin.Context) {
	orders, err := h.orderSvc.ListPaymentPendingOrders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法取得待審核訂單"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (h *AdminHandler) ConfirmPaymentPendingOrder(c *gin.Context) {
	orderID := c.Param("id")
	var req struct {
		TransactionID string `json:"transaction_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少交易編號"})
		return
	}

	if err := h.orderSvc.ConfirmOrder(c.Request.Context(), orderID, req.TransactionID); err != nil {
		middleware.ErrorsTotal.WithLabelValues("manual_confirm_failed").Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "手動確認訂單失敗"})
		return
	}
	middleware.PaymentTotal.WithLabelValues("manual_confirmed").Inc()
	c.JSON(http.StatusOK, gin.H{"status": "confirmed"})
}

func (h *AdminHandler) CancelPaymentPendingOrder(c *gin.Context) {
	orderID := c.Param("id")
	if err := h.orderSvc.CancelOrder(c.Request.Context(), orderID); err != nil {
		middleware.ErrorsTotal.WithLabelValues("manual_cancel_failed").Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "手動取消訂單失敗"})
		return
	}
	middleware.PaymentTotal.WithLabelValues("manual_cancelled").Inc()
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}
