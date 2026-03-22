## ADDED Requirements

### Requirement: Create order upon seat allocation
The system SHALL create an order record with status "pending" immediately after successful seat allocation, before initiating payment.

#### Scenario: Order created after seat lock
- **WHEN** seats are successfully locked for the user
- **THEN** the system creates an Order with status "pending", associated OrderItems linking to the locked seats, and displays the checkout confirmation page

### Requirement: Display order confirmation
The system SHALL display a confirmation page after successful payment showing the order details including event name, date, section, assigned seat numbers, and total amount.

#### Scenario: Payment success page
- **WHEN** payment is confirmed successfully
- **THEN** the system displays order confirmation with event title, event date, section name, row and seat numbers, total paid amount, and order ID

### Requirement: List user orders
The system SHALL provide a "My Orders" page listing all orders for the authenticated user, sorted by creation date descending.

#### Scenario: User views order history
- **WHEN** user navigates to the "我的訂單" page
- **THEN** the system displays all orders with event name, date, section, number of tickets, total amount, and order status

#### Scenario: User has no orders
- **WHEN** user navigates to "我的訂單" with no purchase history
- **THEN** the system displays "尚無訂單紀錄"

### Requirement: Display electronic ticket
The system SHALL generate and display an electronic ticket (QR code) for each confirmed order that can be used for event entry.

#### Scenario: User views electronic ticket
- **WHEN** user clicks on a confirmed order
- **THEN** the system displays a QR code containing the order ID and seat information, along with event entry instructions

#### Scenario: Order not yet confirmed
- **WHEN** user views an order with status "pending" or "payment_pending"
- **THEN** the system displays the order status without a QR code and shows appropriate messaging
