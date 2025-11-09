package events

type OrderCreatedEvent struct {
	OrderID  string      `json:"orderId"`
	UserID   string      `json:"userId"`
	Products []OrderItem `json:"products"`
}

type OrderItem struct {
	ProductID string `json:"productId"`
	Quantity  int32  `json:"quantity"`
}
