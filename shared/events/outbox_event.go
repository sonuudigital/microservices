package events

type OutboxEvent struct {
	ID          string `json:"id"`
	AggregateID string `json:"aggregateId"`
	EventName   string `json:"eventName"`
	Payload     []byte `json:"payload"`
	Status      any    `json:"status"`
}
