-- name: CreateOutboxEvent :exec
INSERT INTO outbox_events (aggregate_id, event_name, payload)
VALUES ($1, $2, $3);

-- name: UpdateOutboxEventStatus :exec
UPDATE outbox_events
SET
    status = $2,
    published_at = NOW()
WHERE
    id = $1;

-- name: CancelOutboxEventStatusByAggregateID :exec
UPDATE outbox_events
SET
    status = 'CANCELLED'
WHERE
    aggregate_id = $1;

-- name: GetUnpublishedOutboxEvents :many
SELECT *
FROM outbox_events
WHERE status = 'UNPUBLISHED'
ORDER BY created_at ASC
LIMIT $1;
