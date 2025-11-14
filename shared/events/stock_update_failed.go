package events

type StockUpdateFailedEvent struct {
	OrderID  string      `json:"orderId"`
	Products []OrderItem `json:"products"`
}
