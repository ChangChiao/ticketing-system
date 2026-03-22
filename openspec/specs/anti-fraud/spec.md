## ADDED Requirements

### Requirement: CAPTCHA verification before queue entry
The system SHALL require users to pass a CAPTCHA challenge (hCaptcha or Cloudflare Turnstile) before joining the ticket queue.

#### Scenario: User passes CAPTCHA
- **WHEN** user completes the CAPTCHA challenge successfully
- **THEN** the system allows the user to join the queue

#### Scenario: User fails CAPTCHA
- **WHEN** user fails the CAPTCHA challenge
- **THEN** the system blocks queue entry and allows the user to retry

### Requirement: Single account single session
The system SHALL enforce that each user account can only maintain one active queue/purchase session per event.

#### Scenario: Duplicate session detected
- **WHEN** a user with an active session attempts to start a new session for the same event
- **THEN** the system rejects the new session and returns "您已有進行中的購票程序"

### Requirement: IP and device rate limiting
The system SHALL limit the number of queue entries per IP address and device fingerprint to prevent bulk bot attacks.

#### Scenario: IP rate limit exceeded
- **WHEN** more than 5 queue entries originate from the same IP address within 1 minute
- **THEN** the system blocks further queue entries from that IP and displays "請求過於頻繁，請稍後再試"

#### Scenario: Device fingerprint limit exceeded
- **WHEN** more than 3 queue entries originate from the same device fingerprint
- **THEN** the system blocks further queue entries from that device

### Requirement: API rate limiting
The system SHALL enforce rate limits on all API endpoints to prevent abuse.

#### Scenario: API rate limit triggered
- **WHEN** a client exceeds the allowed request rate (100 requests per minute per user)
- **THEN** the system returns HTTP 429 with a Retry-After header

#### Scenario: Unauthenticated API rate limit
- **WHEN** unauthenticated requests from a single IP exceed 30 requests per minute
- **THEN** the system returns HTTP 429
