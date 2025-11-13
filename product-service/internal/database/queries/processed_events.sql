-- name: GetProcessedEventByAggregateIDAndEventName :one
SELECT * FROM processed_events
WHERE aggregate_id = $1 AND event_name = $2;

-- name: CreateProcessedEvent :exec
INSERT INTO processed_events (aggregate_id, event_name)
VALUES ($1, $2);
