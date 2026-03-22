## ADDED Requirements

### Requirement: Initiate LINE Pay payment
The system SHALL create a LINE Pay payment request via the Reserve API when the user confirms their order, and redirect the user to the LINE Pay payment page.

#### Scenario: Successful payment initiation
- **WHEN** user confirms their order on the checkout page
- **THEN** the system calls LINE Pay Request API with order details (amount, currency TWD, order ID, product info) and redirects the user to the returned payment URL

#### Scenario: LINE Pay API unavailable
- **WHEN** the LINE Pay Request API returns an error or times out
- **THEN** the system displays "付款系統暫時無法使用，請稍後重試" and retains the seat lock

### Requirement: Confirm LINE Pay payment
The system SHALL confirm the payment by calling the LINE Pay Confirm API when the user is redirected back to the confirm URL.

#### Scenario: Successful payment confirmation
- **WHEN** LINE Pay redirects the user to confirmUrl with a valid transactionId
- **THEN** the system calls the Confirm API, updates the order status to confirmed, transitions seat status from locked to sold, and displays the success page

#### Scenario: Confirm API fails
- **WHEN** the Confirm API call fails
- **THEN** the system retries up to 3 times with exponential backoff. If all retries fail, the order is marked as "payment_pending" for manual review, and the seat lock is maintained

### Requirement: Handle payment cancellation
The system SHALL handle user-initiated payment cancellation from the LINE Pay page.

#### Scenario: User cancels on LINE Pay page
- **WHEN** user clicks cancel on the LINE Pay payment page and is redirected to cancelUrl
- **THEN** the system releases the locked seats, cancels the order, and redirects the user back to the selection page with their queue session still active

### Requirement: Handle payment timeout
The system SHALL handle the case where the user does not complete payment before the seat lock expires.

#### Scenario: Seat lock expires during payment
- **WHEN** the 10-minute seat lock TTL expires while the user is on the LINE Pay page
- **THEN** upon returning to the confirm URL, the system detects the expired lock, voids the payment if needed, displays "付款逾時，座位已釋出", and redirects to the selection page
