-- Automatically pre-create event_seats when an event is published/on sale.
CREATE OR REPLACE FUNCTION create_event_seats_on_publish()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status IN ('on_sale', 'published')
        AND (TG_OP = 'INSERT' OR OLD.status IS DISTINCT FROM NEW.status)
    THEN
        INSERT INTO event_seats (event_id, seat_id, status)
        SELECT NEW.id, se.id, 'available'
        FROM seats se
        JOIN rows rw ON rw.id = se.row_id
        JOIN sections s ON s.id = rw.section_id
        WHERE s.venue_id = NEW.venue_id
        ON CONFLICT (event_id, seat_id) DO NOTHING;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_create_event_seats_on_publish
AFTER INSERT OR UPDATE OF status ON events
FOR EACH ROW
EXECUTE FUNCTION create_event_seats_on_publish();
