-- Events and event-seat mapping
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    venue_id UUID NOT NULL REFERENCES venues(id),
    title VARCHAR(255) NOT NULL,
    event_date TIMESTAMP WITH TIME ZONE NOT NULL,
    sale_start TIMESTAMP WITH TIME ZONE NOT NULL,
    sale_end TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    image_url VARCHAR(500) DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE event_sections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id),
    section_id UUID NOT NULL REFERENCES sections(id),
    price INT NOT NULL,
    quota INT NOT NULL,
    UNIQUE(event_id, section_id)
);

CREATE TABLE event_seats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id),
    seat_id UUID NOT NULL REFERENCES seats(id),
    status VARCHAR(20) NOT NULL DEFAULT 'available',
    locked_by UUID,
    locked_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(event_id, seat_id)
);

CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_sale_start ON events(sale_start);
CREATE INDEX idx_event_sections_event ON event_sections(event_id);
CREATE INDEX idx_event_seats_event_status ON event_seats(event_id, status);
