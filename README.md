# grxm-webapp (Template Web Application)

A robust, modern Go-based web application template designed to seamlessly integrate with the `grxm-iam` zero-trust Identity and Access Management service. This template provides a highly modular backend architecture, ready for rapid SaaS and API development.

## Features

*   **Zero-Trust Authentication:** Integrates out-of-the-box with `grxm-iam` via RSA-signed JWT bearer tokens.
*   **High-Speed Token Denylisting:** Utilizes a Redis cache layer to instantly reject revoked or banned sessions, overriding the mathematical validity of JWTs.
*   **Ad-Hoc Authority Connection:** Establishes a secure internal WebSocket connection to the IAM service on boot to automatically fetch cryptographic keys and dispatch administrative commands.
*   **Multi-Provider Database Layer:** Features a modular database layer (`internal/db`) currently supporting MongoDB, with the ability to easily swap or add providers (e.g., PostgreSQL, SQLite) via configuration.
*   **Clean Architecture:** Code is strictly segregated into `api`, `config`, `db`, `iam`, and `utils` packages, ensuring maintainable, scalable growth.
*   **Static Asset Serving:** Includes native HTTP handlers for serving static frontend templates, HTML, CSS, and JS files alongside protected API routes.

## Configuration

The application is driven by a simple `config.json` file. The backend looks for this file in the current directory, or you can specify its absolute path using the `API_CONFIG_LOCATION` environment variable.

### Example `config.json`
```json
{
    "port": "8080",
    "iam_host": "localhost:8081",
    "iam_authority_password": "change-this-in-production-123",
    "redis_host": "localhost:6379",
    "redis_password": "",
    "redis_db": 0,
    "cookie_name": "grxm-token",
    "db_provider": "mongo",
    "mongo_uri": "mongodb://localhost:27017",
    "mongo_db": "grxm_webapp"
}
```

### Configuration Fields
*   `port`: The HTTP port the web application listens on.
*   `iam_host`: The host address of the `grxm-iam` service.
*   `iam_authority_password`: The master password required to access the secure IAM Authority WebSocket API.
*   `redis_host`: The host address of the Redis instance used for high-speed token denylist checks.
*   `redis_password`: Optional password for the Redis instance.
*   `redis_db`: The Redis database index to use (default: `0`).
*   `cookie_name`: The name of the authentication cookie to read/set (default: `"grxm-token"`).
*   `db_provider`: The active database engine. Currently supports `"mongo"`.
*   `mongo_uri`: The connection string for the MongoDB instance.
*   `mongo_db`: The specific MongoDB database name to use for this application.

## Development & Usage

### Prerequisites
*   Go 1.25+
*   Running instances of Redis, MongoDB, and `grxm-iam`.

### Running Locally
1.  Ensure your `config.json` is properly configured.
2.  Install dependencies:
    ```bash
    go mod tidy
    ```
3.  Run the application:
    ```bash
    go run .
    ```

### Building Handlers
When creating new protected routes in `internal/api`, wrap them in the IAM authorization middleware provided by the initialized IAM client:

```go
// Example in main.go
mux.HandleFunc("/api/secure-data", iamClient.AuthRequired(api.SecureDataHandler))
```
The middleware guarantees the token is mathematically valid, not expired, and not present on the Redis denylist. It securely injects the validated user's ID into the HTTP request headers (`X-User-ID`), allowing your handlers to trust the caller's identity.

### IAM Administrative Commands
The initialized `iamClient` provides ad-hoc methods to control user states directly against the `grxm-iam` service:
*   `iamClient.BanUser(uid string, reason string) error`
*   `iamClient.UnbanUser(uid string) error`
*   `iamClient.UpdateRoles(uid string, roles []string) error`
*   `iamClient.AddRole(uid string, role string) error`
*   `iamClient.RemoveRole(uid string, role string) error`
