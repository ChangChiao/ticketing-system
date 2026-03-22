## ADDED Requirements

### Requirement: Auto-assign consecutive seats
The system SHALL automatically assign consecutive seats in the same row when a user submits a section and quantity selection. The algorithm prioritizes middle rows for better viewing experience.

#### Scenario: Successful consecutive seat assignment
- **WHEN** user requests 2 tickets in Section A
- **THEN** the system assigns 2 consecutive seats in the same row, starting from middle rows and searching outward

#### Scenario: No consecutive seats available in any row
- **WHEN** no single row in the selected section has enough consecutive available seats
- **THEN** the system displays "此區域已無連續 N 張座位，請減少張數或選擇其他區域"

### Requirement: Atomic seat locking
The system SHALL lock all requested seats atomically using a single Redis Lua Script operation — either all seats are locked or none are.

#### Scenario: Concurrent users request overlapping seats
- **WHEN** User A and User B simultaneously request seats that overlap
- **THEN** exactly one user succeeds (all seats locked), the other fails and is prompted to retry with a new assignment

#### Scenario: Partial lock prevention
- **WHEN** 3 seats are requested but only 2 are available in the target row
- **THEN** the system does NOT lock the 2 available seats; instead it searches for another row with 3 consecutive seats

### Requirement: 10-minute seat lock TTL
The system SHALL lock assigned seats for exactly 10 minutes. If payment is not completed within this window, the seats are automatically released.

#### Scenario: Seats released after timeout
- **WHEN** 10 minutes pass after seat locking without payment completion
- **THEN** the Redis TTL expires, seats return to available status, and the user's session is invalidated

#### Scenario: Payment countdown warning
- **WHEN** 8 minutes have passed since seat locking (2 minutes remaining)
- **THEN** the system displays a prominent countdown warning to the user

#### Scenario: Successful payment within time limit
- **WHEN** user completes payment within 10 minutes
- **THEN** the seat status transitions from locked to sold in both Redis and PostgreSQL

### Requirement: Retry seat allocation on failure
The system SHALL automatically attempt to find alternative seats when the first allocation attempt fails due to concurrency.

#### Scenario: Automatic retry after lock failure
- **WHEN** seat locking fails because another user locked the target seats first
- **THEN** the system immediately runs the allocation algorithm again to find different consecutive seats, up to 3 retries

#### Scenario: All retries exhausted
- **WHEN** 3 consecutive allocation attempts fail
- **THEN** the system displays "此區域座位搶購中，請稍後重試或選擇其他區域"
