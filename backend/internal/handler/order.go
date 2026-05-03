package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ticketing-system/backend/internal/middleware"
	"github.com/ticketing-system/backend/internal/model"
	"github.com/ticketing-system/backend/internal/service"
	"github.com/ticketing-system/backend/pkg/linepay"
)

type OrderHandler struct {
	svc        *service.OrderService
	linePayCli *linepay.Client
	queueSvc   *service.QueueService
}

func NewOrderHandler(svc *service.OrderService, linePayCli *linepay.Client, queueSvc ...*service.QueueService) *OrderHandler {
	h := &OrderHandler{svc: svc, linePayCli: linePayCli}
	if len(queueSvc) > 0 {
		h.queueSvc = queueSvc[0]
	}
	return h
}

func (h *OrderHandler) requestLinePay(order *model.Order) (*linepay.RequestPaymentOutput, error) {
	return h.linePayCli.RequestPayment(linepay.RequestPaymentInput{
		OrderID:       order.ID,
		Amount:        order.Total,
		ProductName:   "演唱會門票",
		Quantity:      1,
		Price:         order.Total,
		CallbackToken: order.CallbackToken,
	})
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		EventID      string           `json:"event_id" binding:"required"`
		Seats        []model.SeatInfo `json:"seats" binding:"required"`
		PricePerSeat int              `json:"price_per_seat"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "訂單資料不完整"})
		return
	}

	order, err := h.svc.CreateOrder(c.Request.Context(), userID, req.EventID, req.Seats)
	if err != nil {
		if errors.Is(err, service.ErrInvalidSeatLock) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "建立訂單失敗"})
		return
	}

	paymentOutput, err := h.requestLinePay(order)
	if err != nil {
		log.Printf("LINE Pay request failed for order %s: %v", order.ID, err)
		middleware.PaymentTotal.WithLabelValues("failed").Inc()
		middleware.ErrorsTotal.WithLabelValues("line_pay_request_error").Inc()
		c.JSON(http.StatusCreated, gin.H{
			"id":            order.ID,
			"status":        order.Status,
			"total":         order.Total,
			"payment_error": "付款服務暫時無法使用，請稍後再試",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":             order.ID,
		"status":         order.Status,
		"total":          order.Total,
		"payment_url":    paymentOutput.PaymentURL,
		"transaction_id": paymentOutput.TransactionID,
	})
}

func (h *OrderHandler) CreatePayment(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetString("user_id")

	order, _, err := h.svc.GetUserOrder(c.Request.Context(), orderID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到此訂單"})
		return
	}
	if order.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "此訂單目前無法付款"})
		return
	}

	expired, err := h.svc.AreSeatsExpired(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法確認座位保留狀態"})
		return
	}
	if expired {
		_ = h.svc.CancelOrder(c.Request.Context(), orderID)
		middleware.PaymentTotal.WithLabelValues("timeout").Inc()
		middleware.ErrorsTotal.WithLabelValues("payment_timeout").Inc()
		c.JSON(http.StatusConflict, gin.H{"error": "付款逾時，座位已釋出"})
		return
	}

	paymentOutput, err := h.requestLinePay(order)
	if err != nil {
		log.Printf("LINE Pay retry request failed for order %s: %v", order.ID, err)
		middleware.PaymentTotal.WithLabelValues("failed").Inc()
		middleware.ErrorsTotal.WithLabelValues("line_pay_request_error").Inc()
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "付款服務暫時無法使用，請稍後再試"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             order.ID,
		"status":         order.Status,
		"total":          order.Total,
		"payment_url":    paymentOutput.PaymentURL,
		"transaction_id": paymentOutput.TransactionID,
	})
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
	userID := c.GetString("user_id")
	order, items, err := h.svc.GetUserOrder(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到此訂單"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": order, "items": items})
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	userID := c.GetString("user_id")

	order, err := h.svc.CancelUserOrder(c.Request.Context(), orderID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到此訂單"})
		return
	}
	h.restoreSelection(c, order.EventID, order.UserID)
	middleware.PaymentTotal.WithLabelValues("cancelled").Inc()
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

func (h *OrderHandler) ConfirmPayment(c *gin.Context) {
	transactionID := c.Query("transactionId")
	orderID := c.Query("orderId")
	callbackToken := c.Query("token")

	if transactionID == "" || orderID == "" || callbackToken == "" {
		c.Redirect(http.StatusFound, "/orders?error=missing_params")
		return
	}

	// Validate callback token
	valid, err := h.svc.ValidateCallbackToken(c.Request.Context(), orderID, callbackToken)
	if err != nil {
		log.Printf("Failed to validate callback token for order %s: %v", orderID, err)
		c.Redirect(http.StatusFound, "/orders?error=validation_failed")
		return
	}
	if !valid {
		log.Printf("Invalid callback token for order %s", orderID)
		c.JSON(http.StatusForbidden, gin.H{"error": "無效的回調驗證"})
		return
	}

	// Idempotency: if order is already confirmed, redirect to success
	order, _, err := h.svc.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		c.Redirect(http.StatusFound, "/orders?error=order_not_found")
		return
	}
	if order.Status == "confirmed" {
		c.Redirect(http.StatusFound, "/orders/"+orderID+"/confirmation")
		return
	}

	// Check if seat locks have expired
	expired, err := h.svc.AreSeatsExpired(c.Request.Context(), orderID)
	if err != nil {
		log.Printf("Failed to check seat lock expiry for order %s: %v", orderID, err)
		c.Redirect(http.StatusFound, "/orders/"+orderID+"/confirmation?error=check_failed")
		return
	}
	if expired {
		log.Printf("Seat locks expired for order %s", orderID)
		if err := h.linePayCli.VoidPaymentWithRetry(transactionID); err != nil {
			log.Printf("LINE Pay void failed for expired order %s transaction %s: %v", orderID, transactionID, err)
			middleware.ErrorsTotal.WithLabelValues("line_pay_void_error").Inc()
		}
		_ = h.svc.CancelOrder(c.Request.Context(), orderID)
		h.restoreSelection(c, order.EventID, order.UserID)
		middleware.PaymentTotal.WithLabelValues("timeout").Inc()
		middleware.ErrorsTotal.WithLabelValues("payment_timeout").Inc()
		c.Redirect(http.StatusFound, "/events/"+order.EventID+"/select?error=expired")
		return
	}

	// Call LINE Pay Confirm API with retry and exponential backoff
	err = h.linePayCli.ConfirmPaymentWithRetry(linepay.ConfirmPaymentInput{
		TransactionID: transactionID,
		Amount:        order.Total,
	})
	if err != nil {
		log.Printf("LINE Pay confirm failed for order %s after retries: %v", orderID, err)
		middleware.PaymentTotal.WithLabelValues("failed").Inc()
		middleware.ErrorsTotal.WithLabelValues("line_pay_error").Inc()
		// Mark as payment_pending for manual review
		_ = h.svc.MarkPaymentPending(c.Request.Context(), orderID)
		c.Redirect(http.StatusFound, "/orders/"+orderID+"/confirmation")
		return
	}

	if err := h.svc.ConfirmOrder(c.Request.Context(), orderID, transactionID); err != nil {
		log.Printf("Failed to confirm order %s: %v", orderID, err)
		middleware.PaymentTotal.WithLabelValues("failed").Inc()
		middleware.ErrorsTotal.WithLabelValues("confirm_failed").Inc()
		c.Redirect(http.StatusFound, "/orders/"+orderID+"/confirmation?error=confirm_failed")
		return
	}

	middleware.PaymentTotal.WithLabelValues("success").Inc()
	middleware.TicketSalesRate.WithLabelValues(order.EventID).Inc()
	c.Redirect(http.StatusFound, "/orders/"+orderID+"/confirmation")
}

func (h *OrderHandler) CancelPayment(c *gin.Context) {
	orderID := c.Query("orderId")
	callbackToken := c.Query("token")
	if orderID == "" || callbackToken == "" {
		c.Redirect(http.StatusFound, "/events")
		return
	}

	// Validate callback token
	valid, err := h.svc.ValidateCallbackToken(c.Request.Context(), orderID, callbackToken)
	if err != nil || !valid {
		log.Printf("Invalid callback token on cancel for order %s", orderID)
		c.Redirect(http.StatusFound, "/events")
		return
	}

	if err := h.svc.CancelOrder(c.Request.Context(), orderID); err != nil {
		log.Printf("Failed to cancel order %s: %v", orderID, err)
	}
	middleware.PaymentTotal.WithLabelValues("cancelled").Inc()

	// Get event ID from order to redirect back to event
	order, _, err := h.svc.GetOrder(c.Request.Context(), orderID)
	if err == nil {
		h.restoreSelection(c, order.EventID, order.UserID)
		c.Redirect(http.StatusFound, "/events/"+order.EventID+"/select?error=payment_cancelled")
		return
	}
	c.Redirect(http.StatusFound, "/events")
}

func (h *OrderHandler) restoreSelection(c *gin.Context, eventID, userID string) {
	if h.queueSvc == nil {
		return
	}
	if err := h.queueSvc.RestoreSelection(c.Request.Context(), eventID, userID); err != nil {
		log.Printf("Failed to restore selection session for user %s event %s: %v", userID, eventID, err)
	}
}
