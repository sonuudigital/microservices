package events

type OrderCreatedEvent struct {
	OrderID  string      `json:"order_id"`
	UserID   string      `json:"user_id"`
	Products []OrderItem `json:"products"`
}

type OrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}
