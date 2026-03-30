package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// Business metrics
	QueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ticketing_queue_depth",
			Help: "Current number of users in queue per event",
		},
		[]string{"event_id"},
	)

	TicketSalesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ticketing_ticket_sales_total",
			Help: "Total tickets sold",
		},
		[]string{"event_id", "section_id"},
	)

	TicketSalesRate = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ticketing_ticket_sales_rate",
			Help: "Ticket sales rate (for rate() queries)",
		},
		[]string{"event_id"},
	)

	SeatAllocationAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ticketing_seat_allocation_attempts_total",
			Help: "Total seat allocation attempts",
		},
		[]string{"event_id", "result"}, // result: success, conflict, error
	)

	SeatAllocationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ticketing_seat_allocation_duration_seconds",
			Help:    "Seat allocation duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		},
		[]string{"event_id"},
	)

	PaymentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ticketing_payments_total",
			Help: "Total payment attempts",
		},
		[]string{"status"}, // success, failed, timeout
	)

	ActiveWebSocketConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ticketing_websocket_connections_active",
			Help: "Current number of active WebSocket connections",
		},
	)

	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ticketing_errors_total",
			Help: "Total application errors by type",
		},
		[]string{"type"}, // seat_lock_failed, payment_timeout, line_pay_error, etc.
	)
)

// PrometheusMetrics middleware records HTTP request metrics.
func PrometheusMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := normalizePath(c.FullPath())

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// normalizePath replaces path parameters with placeholders for cardinality control.
func normalizePath(path string) string {
	if path == "" {
		return "unknown"
	}
	return path
}
