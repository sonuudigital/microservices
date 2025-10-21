# Go Microservices E-commerce Platform

This project is a Go-based microservices application for an e-commerce platform, created for educational purposes. It utilizes a Go workspace to manage multiple services, which are containerized using Docker and orchestrated with `docker-compose`.

## Architecture

The project follows a microservices architecture. Key components include:

*   **API Gateway ([`api-gateway`](api-gateway)):** The single entry point for all client requests. It handles routing, authentication, and logging, proxying requests to the appropriate downstream services.
*   **User Service ([`user-service`](user-service)):** Manages user-related operations, including creation, authentication, and retrieval, using a PostgreSQL database.
*   **Product Service ([`product-service`](product-service)):** Manages product-related operations, such as creation, retrieval, and updates, with its own PostgreSQL database.
*   **Cart Service ([`cart-service`](cart-service)):** Manages shopping cart operations, including adding/removing products, clearing cart, and retrieving cart contents. It uses its own PostgreSQL database and communicates with the product service to validate products.
*   **Shared ([`shared`](shared)):** A shared module containing common code for authentication (JWT), logging (slog), web utilities, database connections, and more.
*   **Other Services:** The project includes placeholders for [`notification-service`](notification-service), [`order-service`](order-service), and [`payment-service`](payment-service), which are not yet fully implemented.

## Building and Running

The project is designed to be run using Docker and `docker-compose`.

**To build and run the application for development:**

```bash
docker-compose up --build
```

This command will build the Docker images for each service and start the containers as defined in [`docker-compose.yml`](docker-compose.yml).

For production environments, a [`docker-compose.prod.yml`](docker-compose.prod.yml) is available that uses Docker secrets for JWT key management.

**Services and Ports:**

*   API Gateway: `http://localhost:8080`
*   User Service: Internal (port 8081)
*   Product Service: Internal (port 8082)
*   Cart Service: Internal (port 8083)
*   PostgreSQL Databases: Separate instances for each service (user-db:5432, product-db:5433, cart-db:5434)
*   pgAdmin: `http://localhost:5050`

## Development Conventions

*   **Go Workspace:** The project uses a Go workspace ([`go.work`](go.work)).
*   **Dependency Management:** Go modules are used for managing dependencies.
*   **Database Migrations:** Services with databases ([`user-service`](user-service), [`product-service`](product-service), [`cart-service`](cart-service)) use a `migrations` directory to manage schema changes. Migrations are automatically executed on service startup using [`golang-migrate`](https://github.com/golang-migrate/migrate).
*   **SQLC:** [`sqlc`](https://sqlc.dev/) is used to generate type-safe Go code from SQL queries. Configuration files ([`sqlc.yml`](user-service/sqlc.yml)) are located in each service directory.
*   **Authentication:** Handled via JSON Web Tokens (JWT) using ECDSA (ES256) signing. The [`shared/auth`](shared/auth) package provides a [`JWTManager`](shared/auth/jwt.go) for generating and validating tokens.
*   **Logging:** Uses the `slog` library for structured logging via [`shared/logs`](shared/logs).
*   **Configuration:** Configured using environment variables. A `.env` file can be used for local development (see [`.env`](.env)).
*   **Routing:** Services use the standard `net/http` library.
*   **Password Hashing:** The [`user-service`](user-service) uses `argon2id` for secure password hashing.
*   **Error Handling:** API responses follow the [RFC 7807 Problem Details](https://tools.ietf.org/html/rfc7807) specification via [`shared/web/response.go`](shared/web/response.go).
*   **Database:** PostgreSQL 18 (Alpine) with `pgx/v5` driver for database connectivity.
*   **Containerization:** Multi-stage Docker builds using Go 1.25.0 and distroless base images for minimal attack surface.

## Testing

The project includes both unit and integration tests to ensure the reliability of the services.

*   **Unit Tests:** Located within each service's directory (e.g., [`api-gateway`](api-gateway), [`user-service`](user-service)), these tests verify individual components in isolation. They can be run using the standard `go test` command within each service's folder. For example:
    ```bash
    cd api-gateway
    go test ./...
    ```

*   **Integration Tests:** Found in the [`/tests`](tests) directory, these tests validate the interactions between the microservices. They are designed to be run against a live environment managed by `docker-compose`. The tests cover:
    *   User registration and login
    *   Protected route access with JWT authentication
    *   Product CRUD operations
    *   Cart operations (add, update, remove products)

**To run the integration tests:**

1.  Ensure the services are running:
    ```bash
    docker-compose up --build
    ```
2.  Execute the tests from the `tests` directory:
    ```bash
    cd tests
    GOWORK=off API_GATEWAY_URL=http://localhost:8080 go test -v ./integration
    ```

## Project Structure

```
.
├── api-gateway/            # API Gateway service
├── cart-service/           # Cart management service
├── certs/                  # JWT public/private keys
├── notification-service/   # Notification service (placeholder)
├── order-service/          # Order service (placeholder)
├── payment-service/        # Payment service (placeholder)
├── product-service/        # Product management service
├── shared/                 # Shared utilities and packages
├── tests/                  # Integration tests
├── user-service/           # User management and authentication service
├── docker-compose.yml      # Development compose file
├── docker-compose.prod.yml # Production compose file
└── go.work                 # Go workspace file
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.