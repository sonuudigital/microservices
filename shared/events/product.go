package events

type Product struct {
	ID            string `json:"id"`
	CategoryID    string `json:"categoryId"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Price         string `json:"price"`
	StockQuantity int32  `json:"stockQuantity"`
}
