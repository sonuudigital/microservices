CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS order_statuses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

CREATE UNIQUE INDEX idx_order_status_name ON order_statuses(name);

INSERT INTO order_statuses (name)
VALUES
    ('CREATED'),
    ('DELIVERED'),
    ('CANCELLED');

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    total_amount NUMERIC(10, 2) NOT NULL CHECK (total_amount > 0),
    status UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    FOREIGN KEY (status) REFERENCES order_statuses(id)
);

CREATE OR REPLACE FUNCTION set_default_order_status()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status IS NULL THEN
        NEW.status := (SELECT id FROM order_statuses WHERE name = 'CREATED');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_default_order_status
BEFORE INSERT ON orders
FOR EACH ROW
EXECUTE FUNCTION set_default_order_status();
