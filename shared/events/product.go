package events

const (
	ProductExchangeName       = "products.events"
	ProductCreatedRoutingKey  = "product.created"
	ProductUpdatedRoutingKey  = "product.updated"
	ProductDeletedRoutingKey  = "product.deleted"
	ProductWildcardRoutingKey = "product.#"
	ProductCreatedEventName   = ProductExchangeName + ":" + ProductCreatedRoutingKey
	ProductUpdatedEventName   = ProductExchangeName + ":" + ProductUpdatedRoutingKey
	ProductDeletedEventName   = ProductExchangeName + ":" + ProductDeletedRoutingKey
)

type Product struct {
	ID            string `json:"id"`
	CategoryID    string `json:"categoryId"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Price         string `json:"price"`
	StockQuantity int32  `json:"stockQuantity"`
}
