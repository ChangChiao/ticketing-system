-- Seed: 台北大巨蛋 venue with sections, rows, seats, and a sample event

-- Venue
INSERT INTO venues (id, name, address, layout_data) VALUES (
    'a0000000-0000-0000-0000-000000000001',
    '台北大巨蛋',
    '台北市信義區忠孝東路五段',
    '{
        "width": 800,
        "height": 600,
        "stage": {"x": 400, "y": 50, "width": 200, "height": 60}
    }'
);

-- Sections (simplified 台北大巨蛋 layout with 6 zones)
INSERT INTO sections (id, venue_id, name, capacity, polygon, sort_order) VALUES
('b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'VIP區',  500, '[[300,130],[500,130],[520,230],[280,230]]', 1),
('b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'A區左',  600, '[[100,130],[280,130],[260,280],[80,280]]', 2),
('b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'A區右',  600, '[[520,130],[700,130],[720,280],[540,280]]', 3),
('b0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'B區',   1000, '[[150,250],[650,250],[680,380],[120,380]]', 4),
('b0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001', 'C區',   1200, '[[100,400],[700,400],[730,500],[70,500]]', 5),
('b0000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000001', 'D區',   800, '[[70,520],[730,520],[750,580],[50,580]]', 6);

-- Generate rows and seats for each section using a DO block
DO $$
DECLARE
    sec RECORD;
    row_id UUID;
    seats_per_row INT;
    num_rows INT;
    r INT;
    s INT;
BEGIN
    FOR sec IN SELECT id, name, capacity FROM sections WHERE venue_id = 'a0000000-0000-0000-0000-000000000001' LOOP
        -- Calculate rows and seats per row
        CASE sec.name
            WHEN 'VIP區' THEN num_rows := 10; seats_per_row := 50;
            WHEN 'A區左' THEN num_rows := 12; seats_per_row := 50;
            WHEN 'A區右' THEN num_rows := 12; seats_per_row := 50;
            WHEN 'B區' THEN num_rows := 20; seats_per_row := 50;
            WHEN 'C區' THEN num_rows := 24; seats_per_row := 50;
            WHEN 'D區' THEN num_rows := 16; seats_per_row := 50;
        END CASE;

        FOR r IN 1..num_rows LOOP
            row_id := uuid_generate_v4();
            INSERT INTO rows (id, section_id, label, sort_order)
            VALUES (row_id, sec.id, r || '排', r);

            FOR s IN 1..seats_per_row LOOP
                INSERT INTO seats (id, row_id, number)
                VALUES (uuid_generate_v4(), row_id, s);
            END LOOP;
        END LOOP;
    END LOOP;
END $$;

-- Sample Event: 五月天 2026 演唱會
INSERT INTO events (id, venue_id, title, description, event_date, sale_start, sale_end, status, image_url) VALUES (
    'e0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    '五月天 2026 NO WHERE 巡迴演唱會',
    '五月天年度大型巡迴演唱會台北場，於台北大巨蛋登場。請依票面時間入場，並於付款期限內完成交易。',
    '2026-06-15 19:00:00+08',
    '2026-04-01 12:00:00+08',
    '2026-06-15 18:00:00+08',
    'on_sale',
    ''
);

-- Event sections with pricing
INSERT INTO event_sections (id, event_id, section_id, price, quota) VALUES
(uuid_generate_v4(), 'e0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001', 6800, 500),
(uuid_generate_v4(), 'e0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000002', 4800, 600),
(uuid_generate_v4(), 'e0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000003', 4800, 600),
(uuid_generate_v4(), 'e0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000004', 3200, 1000),
(uuid_generate_v4(), 'e0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000005', 1800, 1200),
(uuid_generate_v4(), 'e0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000006', 1200, 800);

-- Pre-create event_seats for all seats
INSERT INTO event_seats (id, event_id, seat_id, status)
SELECT uuid_generate_v4(), 'e0000000-0000-0000-0000-000000000001', s.id, 'available'
FROM seats s
JOIN rows r ON r.id = s.row_id
JOIN sections sec ON sec.id = r.section_id
WHERE sec.venue_id = 'a0000000-0000-0000-0000-000000000001';
