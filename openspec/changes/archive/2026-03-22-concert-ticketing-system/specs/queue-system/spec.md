## ADDED Requirements

### Requirement: Enter waiting room
The system SHALL place users into a virtual waiting room when they attempt to purchase tickets for an event that is currently on sale.

#### Scenario: User joins queue at sale time
- **WHEN** user clicks "購票" after sale_start time
- **THEN** the system adds the user to the queue with a timestamp-based priority and displays the waiting room page

#### Scenario: User joins queue before sale time
- **WHEN** user is on the event page and sale_start time arrives
- **THEN** the system automatically transitions to the waiting room and adds the user to the queue

### Requirement: Display queue position
The system SHALL display the user's current queue position and estimated wait time via WebSocket, updating every 3 seconds.

#### Scenario: Queue position updates in real time
- **WHEN** user is in the waiting room
- **THEN** the system pushes queue position updates via WebSocket every 3 seconds, showing number of people ahead and estimated wait time

#### Scenario: WebSocket disconnection recovery
- **WHEN** user's WebSocket connection drops and reconnects within 30 seconds
- **THEN** the system restores the user's original queue position

#### Scenario: WebSocket disconnection timeout
- **WHEN** user's WebSocket connection drops and does not reconnect within 30 seconds
- **THEN** the system removes the user from the queue, requiring them to rejoin

### Requirement: Batch admission to selection page
The system SHALL admit users from the queue to the seat selection page in controlled batches, maintaining a maximum concurrent user limit.

#### Scenario: Users admitted in batches
- **WHEN** the number of active users on the selection page is below max_concurrent (500)
- **THEN** the system admits the next batch of users (up to 50) from the queue and notifies them via WebSocket

#### Scenario: User notified when it's their turn
- **WHEN** user reaches the front of the queue and is admitted
- **THEN** the system sends a WebSocket message with status "your_turn" and the user has 60 seconds to enter the selection page

#### Scenario: User misses their turn
- **WHEN** user is notified but does not enter the selection page within 60 seconds
- **THEN** the system moves the user to the back of the queue and admits the next user

### Requirement: Single session per user
The system SHALL enforce that each authenticated user can only have one active queue session per event at a time.

#### Scenario: User opens second browser tab
- **WHEN** user attempts to join the queue for the same event in a second session
- **THEN** the system rejects the second session and displays "您已在排隊中，請回到原視窗"
