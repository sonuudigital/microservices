# Go Microservices E-commerce Platform

This project is a Go-based microservices application for an e-commerce platform, created for educational purposes. It utilizes a Go workspace to manage multiple services, which are containerized using Docker and orchestrated with `docker-compose`.

## Architecture

The project follows a microservices architecture. Key components include:

*   **API Gateway (`api-gateway`):** The single entry point for all client requests. It handles routing, authentication, and logging, proxying requests to the appropriate downstream services.
*   **User Service (`user-service`):** Manages user-related operations, including creation, authentication, and retrieval, using a PostgreSQL database.
*   **Product Service (`product-service`):** Manages product-related operations, such as creation, retrieval, and updates, with its own PostgreSQL database.
*   **Shared (`shared`):** A shared module containing common code for authentication (JWT), logging (slog), web utilities, and more.
*   **Other Services:** The project includes placeholders for `cart-service`, `notification-service`, `order-service`, and `payment-service`, which are not yet fully implemented.

## Building and Running

The project is designed to be run using Docker and `docker-compose`.

**To build and run the application for development:**

```bash
docker-compose up --build
```

This command will build the Docker images for each service and start the containers as defined in `docker-compose.yml`.

For production environments, a `docker-compose.prod.yml` is available.

## Development Conventions

*   **Go Workspace:** The project uses a Go workspace (`go.work`).
*   **Dependency Management:** Go modules are used for managing dependencies.
*   **Database Migrations:** Services with databases (`user-service`, `product-service`) use a `migrations` directory to manage schema changes.
*   **SQLC:** `sqlc` is used to generate type-safe Go code from SQL queries.
*   **Authentication:** Handled via JSON Web Tokens (JWT).
*   **Logging:** Uses the `slog` library for structured logging.
*   **Configuration:** Configured using environment variables (a `.env` file can be used for local development).
*   **Routing:** Services use the standard `net/http` library.
*   **Password Hashing:** The `user-service` uses `argon2id` for secure password hashing.

## Testing

A comprehensive testing strategy is not yet defined. Adding unit and integration tests for each service is a future goal for this project.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
