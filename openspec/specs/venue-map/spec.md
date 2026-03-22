## ADDED Requirements

### Requirement: Render venue map with Canvas
The system SHALL render an interactive venue map using HTML5 Canvas, displaying all sections as colored polygons with the stage area clearly marked.

#### Scenario: Venue map renders on selection page
- **WHEN** user enters the seat selection page
- **THEN** the system renders the venue map via Canvas showing all sections as distinct polygons with the stage at the top

### Requirement: Color-coded availability
The system SHALL color each section polygon based on real-time ticket availability: green (>50% remaining), yellow (10-50% remaining), red (<10% remaining), grey (sold out).

#### Scenario: Section shows green when mostly available
- **WHEN** a section has more than 50% of tickets remaining
- **THEN** the section polygon is rendered in green (#22c55e)

#### Scenario: Section shows yellow when limited
- **WHEN** a section has between 10% and 50% of tickets remaining
- **THEN** the section polygon is rendered in yellow (#eab308)

#### Scenario: Section shows red when scarce
- **WHEN** a section has less than 10% of tickets remaining
- **THEN** the section polygon is rendered in red (#ef4444)

#### Scenario: Section shows grey when sold out
- **WHEN** a section has zero tickets remaining
- **THEN** the section polygon is rendered in grey (#9ca3af) and is not clickable

### Requirement: Section hover details
The system SHALL display section details (name, price, remaining tickets) when the user hovers over a section polygon.

#### Scenario: User hovers over a section
- **WHEN** user moves the cursor over a section polygon on the Canvas
- **THEN** a tooltip displays the section name, ticket price, and number of remaining tickets

### Requirement: Real-time availability updates
The system SHALL update section availability colors in real time via WebSocket as tickets are sold or released.

#### Scenario: Availability changes while user is viewing
- **WHEN** tickets are sold or released in a section while user is on the selection page
- **THEN** the section color updates within 3 seconds without page refresh

### Requirement: Section selection and quantity input
The system SHALL allow users to click a section to select it and specify the number of tickets (1-4 per transaction).

#### Scenario: User selects a section and quantity
- **WHEN** user clicks on an available section polygon
- **THEN** the system highlights the section and displays a quantity selector (1-4 tickets)

#### Scenario: User requests more tickets than available
- **WHEN** user selects a quantity greater than the section's remaining tickets
- **THEN** the system displays "此區域剩餘票數不足" and prevents submission
