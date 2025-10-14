# GEMINI Project Overview

This project is a Go-based microservices application for an e-commerce platform for educational purpose. It utilizes a Go workspace to manage multiple services, including an API gateway and a user service. The services are containerized using Docker and orchestrated with `docker-compose`.

## Architecture

The project follows a microservices architecture. Key components include:

*   **API Gateway (`api-gateway`):** The single entry point for all client requests. It handles routing, authentication, and logging. It proxies requests to the appropriate downstream services.
*   **User Service (`user-service`):** Manages user-related operations, including user creation, authentication, and retrieval. It uses a PostgreSQL database for data persistence.
*   **Shared (`shared`):** A shared module containing common code for authentication (JWT), logging (slog), and other utilities.
*   **Other Services:** The project also includes placeholders for `cart-service`, `notification-service`, `order-service`, and `payment-service`, which are not yet fully implemented.

## Building and Running

The project is designed to be run using Docker and `docker-compose`.

**To build and run the application:**

```bash
docker-compose up --build
```

This command will build the Docker images for each service and start the containers as defined in the `docker-compose.yml` file.

**To run the tests (TODO):**

A testing strategy is not yet defined in the project. It is recommended to add unit and integration tests for each service.

## Development Conventions

*   **Go Workspace:** The project uses a Go workspace (`go.work`) to manage the multiple Go modules.
*   **Dependency Management:** Go modules are used for dependency management.
*   **Database Migrations:** The `user-service` uses a `migrations` directory to manage database schema changes.
*   **SQLC:** The project uses `sqlc` to generate type-safe Go code from SQL queries. The generated code is in `user-service/internal/repository/users.sql.go`.
*   **Authentication:** Authentication is handled using JSON Web Tokens (JWT). The `shared/auth` package provides a `JWTManager` for generating and validating tokens. The `api-gateway` uses an `AuthMiddleware` to protect routes.
*   **Logging:** The project uses the `slog` library for structured logging. A logger is initialized in `shared/logs/slog.go`.
*   **Configuration:** The application is configured using environment variables. A `.env` file can be used for local development.
*   **Routing:** The `api-gateway` uses the standard `net/http` library for routing.
*   **Password Hashing:** The `user-service` uses the `argon2id` library to securely hash and verify passwords.
