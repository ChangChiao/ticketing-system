-- Venues and seating structure
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE venues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    address VARCHAR(500),
    layout_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE sections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    venue_id UUID NOT NULL REFERENCES venues(id),
    name VARCHAR(100) NOT NULL,
    capacity INT NOT NULL,
    polygon JSONB NOT NULL,
    sort_order INT NOT NULL DEFAULT 0
);

CREATE TABLE rows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    section_id UUID NOT NULL REFERENCES sections(id),
    label VARCHAR(20) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0
);

CREATE TABLE seats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    row_id UUID NOT NULL REFERENCES rows(id),
    number INT NOT NULL
);

CREATE INDEX idx_sections_venue ON sections(venue_id);
CREATE INDEX idx_rows_section ON rows(section_id);
CREATE INDEX idx_seats_row ON seats(row_id);
