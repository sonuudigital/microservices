# Go Microservices E-commerce Platform

This project is a Go-based microservices application for an e-commerce platform, created for educational purposes. It utilizes a Go workspace, with services communicating via gRPC. The entire application is containerized using Docker and orchestrated with `docker-compose`.

## Architecture

The project follows a microservices architecture. Key components include:

*   **API Gateway ([`api-gateway`](api-gateway)):** The single entry point for all client requests. It handles routing and authentication, acting as a gRPC client to proxy requests to the appropriate downstream services.
*   **User Service ([`user-service`](user-service)):** Manages user creation, authentication, and retrieval via a gRPC server. It uses a PostgreSQL database.
*   **Product Service ([`product-service`](product-service)):** Manages products and product categories through a gRPC server, with its own PostgreSQL database.
*   **Cart Service ([`cart-service`](cart-service)):** Manages shopping cart operations. It communicates with the Product Service via gRPC to get product details and uses its own PostgreSQL database.
*   **Order Service ([`order-service`](order-service)):** Orchestrates the order creation process, communicating with the Cart, Product, and Payment services. It implements the Saga and Outbox patterns to ensure data consistency across services.
*   **Payment Service ([`payment-service`](payment-service)):** Simulates payment processing for orders. It's an internal service called by the Order Service.
*   **Shared ([`shared`](shared)):** A shared module containing common code for authentication, logging, and database connections.
*   **Protobufs ([`protos`](protos)):** Contains all gRPC service definitions for the project.

### Resiliency and Data Consistency

The project implements the **Saga and Outbox patterns** to maintain data consistency across microservices during the order creation process.

*   **Saga Pattern:** The `order-service` acts as a saga orchestrator. When a user creates an order, it initiates a distributed transaction that involves:
    1.  Creating the order in a `PENDING` state.
    2.  Requesting payment processing from the `payment-service`.
    3.  On successful payment, publishing an `OrderCreated` event.
    4.  The `product-service` consumes this event to update stock levels.
    5.  If stock updates fail, a `StockUpdateFailed` event is published, which triggers a compensating transaction in the `order-service` to cancel the order.

*   **Outbox Pattern:** To ensure reliable event publishing, the `order-service` and `product-service` use the outbox pattern. Instead of publishing events directly to the message broker, they are first saved to an `outbox_events` table in the local database within the same transaction as the business logic. A separate worker process (`MessageRelayer`) polls this table and publishes the events to RabbitMQ, guaranteeing that events are published if and only if the original transaction was successful.

## Building and Running

The project is designed to be run using Docker and `docker-compose`.

**To build and run the application for development:**

```bash
docker-compose up --build
```

This command builds and starts all services. For production, a `docker-compose.prod.yml` is available that uses Docker secrets for JWT keys.

**Services and Ports:**

*   API Gateway: `http://localhost:8080`
*   User Service (gRPC): Internal (port 9081)
*   Product Service (gRPC): Internal (port 9082)
*   Cart Service (gRPC): Internal (port 9083)
*   Order Service (gRPC): Internal (port 9084)
*   Payment Service (gRPC): Internal (port 9085)
*   PostgreSQL Databases: Separate instances for each service.
*   pgAdmin: `http://localhost:5050`
*   RabbitMQ: `http://localhost:15672`

## Development Conventions

*   **Go Workspace:** The project uses a Go workspace ([`go.work`](go.work)).
*   **gRPC Communication:** Services communicate using gRPC, with definitions stored in the [`protos`](protos) directory.
*   **SQLC:** [`sqlc`](https://sqlc.dev/) generates type-safe Go code from SQL queries in each service.
*   **Database Migrations:** Schema changes are managed in a `migrations` directory per service and applied on startup.
*   **Authentication:** Handled via JWT (ECDSA), with the API Gateway protecting routes.
*   **Containerization:** Multi-stage Docker builds using Go `1.25.0` and distroless images.
*   **Saga & Outbox Patterns:** Used for handling distributed transactions and ensuring reliable eventing.

## Testing

The project includes both unit and integration tests.

*   **Unit Tests:** Run with `go test ./...` inside each service directory.
*   **Integration Tests:** Located in the [`/tests`](tests) directory. To run them:

    1.  Start the services: `docker-compose up --build`
    2.  Execute the tests:
        ```bash
        cd tests
        GOWORK=off API_GATEWAY_URL=http://localhost:8080 go test -v ./integration
        ```

## Key Endpoints

- `POST /api/users` - User registration
- `POST /api/auth/login` - User login
- `GET /api/users/{id}` - Get user (protected)
- `GET /api/products` - List products (paginated)
- `GET /api/products/{id}` - Get product
- `POST /api/products` - Create product (protected)
- `PUT /api/products/{id}` - Update product (protected)
- `DELETE /api/products/{id}` - Delete product (protected)
- `GET /api/products/categories` - List all product categories
- `POST /api/products/categories` - Create a product category (protected)
- `PUT /api/products/categories` - Update a product category (protected)
- `DELETE /api/products/categories/{id}` - Delete a product category (protected)
- `GET /api/products/categories/{categoryId}` - Get products by category ID
- `GET /api/carts` - Get user's cart (protected)
- `POST /api/carts/products` - Add product to cart (protected)
- `DELETE /api/carts/products/{productId}` - Remove product from cart (protected)
- `DELETE /api/carts/products` - Clear all products from the cart (protected)
- `DELETE /api/carts` - Deletes the entire cart (protected)
- `POST /api/orders` - Create a new order from the user's cart (protected)

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.