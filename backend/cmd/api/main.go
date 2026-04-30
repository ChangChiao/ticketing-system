package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ticketing-system/backend/internal/config"
	"github.com/ticketing-system/backend/internal/handler"
	"github.com/ticketing-system/backend/internal/middleware"
	"github.com/ticketing-system/backend/internal/repository"
	"github.com/ticketing-system/backend/internal/service"
	"github.com/ticketing-system/backend/internal/ws"
	"github.com/ticketing-system/backend/pkg/linepay"
	pkgredis "github.com/ticketing-system/backend/pkg/redis"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	goredis "github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})

	redisClient := pkgredis.NewClient(rdb)

	// Repositories
	eventRepo := repository.NewEventRepository(db)
	seatRepo := repository.NewSeatRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Services
	eventSvc := service.NewEventService(eventRepo)
	seatSvc := service.NewSeatService(seatRepo, redisClient)
	orderSvc := service.NewOrderService(orderRepo, seatSvc, redisClient)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	queueSvc := service.NewQueueService(redisClient)

	// LINE Pay Client
	linePayCli := linepay.NewClient(cfg.LinePayChannelID, cfg.LinePayChannelSecret, cfg.LinePayBaseURL, cfg.AppBaseURL)

	// WebSocket Hub
	wsHub := ws.NewHub(cfg.AppBaseURL, cfg.JWTSecret)
	go wsHub.Run()
	wsHub.SubscribeRedis(redisClient)

	if cfg.ServiceRole != "ws" {
		// Start payment warning worker (10.3)
		go orderSvc.StartPaymentWarningWorker(context.Background())
		go orderSvc.StartPaymentTimeoutWorker(context.Background())
		go seatSvc.StartExpiredLockCleanupWorker(context.Background())
		go queueSvc.StartAdmissionWorker(context.Background(), func(eventID, userID string) {
			wsHub.SendToUser(userID, ws.Message{
				Type: "queue_update",
				Data: gin.H{
					"event_id":       eventID,
					"position":       0,
					"estimated_wait": "即將輪到您",
					"status":         "your_turn",
					"entry_window":   service.EntryWindowSeconds,
				},
			})
		})
		go queueSvc.StartPositionUpdateWorker(context.Background(), func(eventID, userID string, position, total int64, estimatedWait string) {
			wsHub.SendToUser(userID, ws.Message{
				Type: "queue_update",
				Data: gin.H{
					"event_id":       eventID,
					"position":       position,
					"total_in_queue": total,
					"estimated_wait": estimatedWait,
					"status":         "waiting",
				},
			})
		})
	}

	// Handlers
	eventHandler := handler.NewEventHandler(eventSvc)
	seatHandler := handler.NewSeatHandler(seatSvc, queueSvc)
	orderHandler := handler.NewOrderHandler(orderSvc, linePayCli)
	authHandler := handler.NewAuthHandler(authSvc)
	queueHandler := handler.NewQueueHandler(queueSvc)

	// Router
	r := gin.Default()
	r.Use(middleware.PrometheusMetrics())
	r.Use(middleware.CORS(cfg.AppBaseURL))
	r.Use(middleware.IPRateLimit(redisClient, 30, time.Minute)) // 30 req/min for unauthenticated

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := r.Group("/api")
	{
		// Public routes
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
		api.GET("/events", eventHandler.ListEvents)
		api.GET("/events/:id", eventHandler.GetEvent)
		api.GET("/events/:id/availability", seatHandler.GetAvailability)

		// Payment callbacks (no auth, verified by transaction)
		api.GET("/payments/confirm", orderHandler.ConfirmPayment)
		api.GET("/payments/cancel", orderHandler.CancelPayment)

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.Auth(cfg.JWTSecret))
		protected.Use(middleware.RateLimit(redisClient, 100, time.Minute)) // 100 req/min per user
		protected.Use(middleware.RequestSignature(cfg.RequestSignSecret))  // Request signature validation
		{
			// Queue join: CAPTCHA + device fingerprint + IP queue rate limit
			protected.POST("/events/:id/queue/join",
				middleware.CaptchaVerify(cfg.TurnstileSecretKey),
				middleware.DeviceFingerprintLimit(redisClient, 3, time.Minute), // 3 queue entries per device
				middleware.IPRateLimit(redisClient, 5, time.Minute),            // 5 queue entries/min per IP
				queueHandler.JoinQueue,
			)
			protected.GET("/events/:id/queue/position", queueHandler.GetPosition)
			protected.POST("/events/:id/queue/enter", queueHandler.EnterSelection)
			protected.POST("/events/:id/allocate", seatHandler.AllocateSeats)
			protected.POST("/orders", orderHandler.CreateOrder)
			protected.GET("/orders", orderHandler.ListOrders)
			protected.GET("/orders/:id", orderHandler.GetOrder)
		}
	}

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		wsHub.HandleWebSocket(c.Writer, c.Request)
	})

	port := cfg.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
		os.Exit(1)
	}
}
