package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ticketing-system/backend/internal/model"
	"github.com/ticketing-system/backend/internal/service"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		EventID      string           `json:"event_id" binding:"required"`
		Seats        []model.SeatInfo `json:"seats" binding:"required"`
		PricePerSeat int              `json:"price_per_seat" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "訂單資料不完整"})
		return
	}

	order, err := h.svc.CreateOrder(c.Request.Context(), userID, req.EventID, req.Seats, req.PricePerSeat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "建立訂單失敗"})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID := c.GetString("user_id")
	orders, err := h.svc.ListOrders(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法取得訂單列表"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	order, items, err := h.svc.GetOrder(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到此訂單"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": order, "items": items})
}

func (h *OrderHandler) ConfirmPayment(c *gin.Context) {
	transactionID := c.Query("transactionId")
	orderID := c.Query("orderId")

	if transactionID == "" || orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少付款參數"})
		return
	}

	if err := h.svc.ConfirmOrder(c.Request.Context(), orderID, transactionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "付款確認失敗"})
		return
	}

	// Redirect to success page
	c.Redirect(http.StatusFound, "/orders/"+orderID+"/confirmation")
}

func (h *OrderHandler) CancelPayment(c *gin.Context) {
	orderID := c.Query("orderId")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少訂單參數"})
		return
	}

	if err := h.svc.CancelOrder(c.Request.Context(), orderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消訂單失敗"})
		return
	}

	c.Redirect(http.StatusFound, "/events")
}
