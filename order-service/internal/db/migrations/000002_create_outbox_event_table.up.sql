CREATE TYPE outbox_event_status AS ENUM ('UNPUBLISHED', 'PUBLISHED');
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate_id UUID NOT NULL,
    event_name VARCHAR(150) NOT NULL,
    payload JSONB NOT NULL,
    status outbox_event_status DEFAULT 'UNPUBLISHED' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    published_at TIMESTAMP WITH TIME ZONE NULL
);