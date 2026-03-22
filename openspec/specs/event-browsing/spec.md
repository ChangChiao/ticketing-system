## ADDED Requirements

### Requirement: Display event list
The system SHALL display a list of available concert events, showing event title, date, venue name, price range, and sale status (upcoming/on-sale/sold-out).

#### Scenario: User views event list
- **WHEN** user navigates to the home page
- **THEN** the system displays all published events sorted by event date ascending

#### Scenario: Event shows correct sale status
- **WHEN** current time is before the event's sale_start
- **THEN** the event displays "即將開賣" with the sale start datetime

#### Scenario: Sold out event
- **WHEN** all sections of an event have zero remaining tickets
- **THEN** the event displays "已售完" and the purchase button is disabled

### Requirement: Display event detail
The system SHALL display event detail page with full event information including title, date, venue, description, section-level pricing, and a venue map preview.

#### Scenario: User views event detail
- **WHEN** user clicks on an event from the list
- **THEN** the system displays the event detail page with all sections, their prices, and remaining ticket availability per section

#### Scenario: Event detail shows countdown before sale
- **WHEN** the event's sale_start is in the future
- **THEN** the detail page displays a countdown timer to sale start time
