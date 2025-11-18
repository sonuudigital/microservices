package events

type OrderCreatedEvent struct {
	OrderID   string      `json:"orderId"`
	UserID    string      `json:"userId"`
	UserEmail string      `json:"userEmail"`
	Products  []OrderItem `json:"products"`
}

type OrderItem struct {
	ProductID string `json:"productId"`
	Quantity  int32  `json:"quantity"`
}
