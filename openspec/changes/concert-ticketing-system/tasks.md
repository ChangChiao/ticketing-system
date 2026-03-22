## 1. Project Setup

- [ ] 1.1 Initialize Go backend project with Gin framework, configure project structure (cmd/, internal/, pkg/)
- [ ] 1.2 Initialize Next.js 14 frontend project with App Router, Tailwind CSS, and Zustand
- [ ] 1.3 Set up PostgreSQL database with Docker Compose (dev environment)
- [ ] 1.4 Set up Redis with Docker Compose (dev environment)
- [ ] 1.5 Create docker-compose.yml combining all services (Go API, Next.js, PostgreSQL, Redis)

## 2. Database Schema & Models

- [ ] 2.1 Create database migration: Venue, Section (with polygon JSONB), Row, Seat tables
- [ ] 2.2 Create database migration: Event, EventSection (with price/quota), EventSeat tables
- [ ] 2.3 Create database migration: User table with authentication fields
- [ ] 2.4 Create database migration: Order, OrderItem, Payment tables
- [ ] 2.5 Implement Go models and repository layer for all entities
- [ ] 2.6 Create seed data script for a sample venue (台北大巨蛋 layout with sections/rows/seats)

## 3. Event Browsing API & Frontend

- [ ] 3.1 Implement GET /api/events endpoint (list events with sale status)
- [ ] 3.2 Implement GET /api/events/:id endpoint (event detail with sections and pricing)
- [ ] 3.3 Build event list page (/) with event cards showing title, date, venue, price range, sale status
- [ ] 3.4 Build event detail page (/events/[id]) with section pricing table and countdown timer

## 4. User Authentication

- [ ] 4.1 Implement user registration and login API endpoints (JWT-based)
- [ ] 4.2 Implement JWT middleware for protected routes
- [ ] 4.3 Build login/register pages in Next.js
- [ ] 4.4 Set up Zustand auth store and token management

## 5. Venue Map (Canvas)

- [ ] 5.1 Build VenueMap Canvas component: render section polygons from JSONB coordinate data
- [ ] 5.2 Implement color-coded section fill based on availability (green/yellow/red/grey thresholds)
- [ ] 5.3 Implement hover tooltip showing section name, price, and remaining tickets
- [ ] 5.4 Implement section click selection and quantity picker (1-4 tickets)
- [ ] 5.5 Implement GET /api/events/:id/availability endpoint returning per-section remaining counts

## 6. Seat Allocation Engine

- [ ] 6.1 Implement seat allocation algorithm: middle-row-first search with sliding window for consecutive seats
- [ ] 6.2 Write Redis Lua Script for atomic multi-seat locking with 10-minute TTL
- [ ] 6.3 Implement POST /api/events/:id/allocate endpoint (accepts section_id + quantity, returns assigned seats)
- [ ] 6.4 Implement retry logic: up to 3 allocation attempts on lock failure
- [ ] 6.5 Implement EventSeat batch pre-creation when an event is published
- [ ] 6.6 Build Redis ↔ PostgreSQL sync: write seat status to DB on payment success

## 7. Queue System

- [ ] 7.1 Implement POST /api/events/:id/queue/join endpoint (ZADD to Redis Sorted Set with CAPTCHA verification)
- [ ] 7.2 Implement queue position query (ZRANK) and estimated wait time calculation
- [ ] 7.3 Build WebSocket server in Go for queue status push (position updates every 3 seconds)
- [ ] 7.4 Implement queue controller worker: batch admission logic (ZPOPMIN, max_concurrent check)
- [ ] 7.5 Implement single-session enforcement per user per event
- [ ] 7.6 Implement WebSocket reconnection: restore queue position within 30-second window
- [ ] 7.7 Build waiting room page (/events/[id]/queue) with position display, estimated wait, and animations
- [ ] 7.8 Build "your turn" notification and 60-second entry window logic

## 8. Checkout & LINE Pay Integration

- [ ] 8.1 Implement POST /api/orders endpoint (create order with pending status after seat allocation)
- [ ] 8.2 Integrate LINE Pay Request API: generate payment URL and redirect user
- [ ] 8.3 Implement GET /api/payments/confirm callback handler (Confirm API call, order status update)
- [ ] 8.4 Implement GET /api/payments/cancel callback handler (release seats, cancel order)
- [ ] 8.5 Implement payment timeout handling: detect expired seat locks on confirm callback
- [ ] 8.6 Implement Confirm API retry with exponential backoff (up to 3 retries)
- [ ] 8.7 Build checkout page (/events/[id]/checkout) showing order summary and 10-minute countdown
- [ ] 8.8 Build payment processing page (/events/[id]/payment) with redirect handling

## 9. Order Management

- [ ] 9.1 Implement GET /api/orders endpoint (list user orders)
- [ ] 9.2 Implement GET /api/orders/:id endpoint (order detail with seat info)
- [ ] 9.3 Implement QR code generation for confirmed orders (electronic ticket)
- [ ] 9.4 Build order confirmation page (/orders/[id]/confirmation) with event details and QR code
- [ ] 9.5 Build "My Orders" page (/orders) with order list and status display

## 10. Real-time Updates (WebSocket)

- [ ] 10.1 Implement Redis Pub/Sub channel for seat availability changes
- [ ] 10.2 Implement WebSocket broadcast: push section availability updates to all users on selection page
- [ ] 10.3 Implement payment countdown WebSocket push (2-minute warning at 8 minutes)
- [ ] 10.4 Connect frontend VenueMap component to WebSocket for real-time color updates

## 11. Anti-Fraud & Security

- [ ] 11.1 Integrate CAPTCHA (hCaptcha or Cloudflare Turnstile) on queue entry
- [ ] 11.2 Implement IP-based rate limiting (5 queue entries/min per IP, 30 req/min unauthenticated)
- [ ] 11.3 Implement device fingerprint collection and rate limiting (3 queue entries per device)
- [ ] 11.4 Implement API rate limiting middleware (100 req/min per authenticated user)
- [ ] 11.5 Add request signature validation to prevent direct API abuse

## 12. Testing & Quality

- [ ] 12.1 Write unit tests for seat allocation algorithm (consecutive seat finding, edge cases)
- [ ] 12.2 Write unit tests for Redis Lua Script (atomic locking, TTL, concurrent access)
- [ ] 12.3 Write integration tests for queue → selection → allocation → payment flow
- [ ] 12.4 Write integration tests for LINE Pay Request/Confirm/Cancel flows
- [ ] 12.5 Set up load testing with k6: simulate 10,000 concurrent queue joins and seat allocations

## 13. Deployment & Infrastructure

- [ ] 13.1 Create Dockerfiles for Go API, WebSocket server, and Next.js frontend
- [ ] 13.2 Create Kubernetes manifests: Deployments, Services, Ingress for all components
- [ ] 13.3 Configure HPA (Horizontal Pod Autoscaler) for API and WebSocket pods
- [ ] 13.4 Set up Cloudflare CDN for static assets and DDoS protection
- [ ] 13.5 Create pre-scaling CronJob for scheduled event sale starts
- [ ] 13.6 Set up monitoring: Prometheus metrics, Grafana dashboards (queue depth, ticket sales rate, error rate)
