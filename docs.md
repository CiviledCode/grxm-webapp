# grxm-stack: Development Environment

This repository contains the Docker Compose environment and configuration files for running the full "grxm" tech stack locally. This stack is designed as a robust foundation for modern Go-based WebApps and SaaS projects.

## Architecture Overview

The stack consists of five core components communicating over an isolated internal Docker network (`grxm-net`):

1.  **Nginx (Reverse Proxy):** Acts as the single entry point (port 80), routing traffic and enforcing security.
2.  **grxm-webapp (Frontend/API):** The primary user-facing Go application. It features a modular internal architecture, multi-provider database support, and deep IAM integration.
3.  **grxm-iam (Identity & Access Management):** A decentralized, zero-trust authentication bearer service.
4.  **MongoDB:** The persistent database layer for both `grxm-iam` and the `grxm-webapp` (when configured).
5.  **Redis:** High-speed key-value store used by the webapp for real-time token denylisting (banning).

---

## Webapp Internal Architecture (Go)

The `grxm-webapp` is organized into a clean, modular structure within the `internal/` directory to facilitate rapid development and maintenance.

### Packages
*   **`internal/api`**: Contains HTTP handlers and routing logic. This is the primary area for building out new application features.
*   **`internal/config`**: Handles JSON-based configuration loading with environment variable overrides and safe defaults.
*   **`internal/db`**: A provider-based database abstraction layer.
    *   Supports multiple backends (currently **MongoDB**).
    *   Automatically establishes connections on boot based on `db_provider` in `config.json`.
    *   Exposes global `db.MongoDB` and `db.MongoClient` for easy access in handlers.
*   **`internal/iam`**: The core integration package for `grxm-iam`.
    *   **`AuthUser` Middleware**: An overarching authentication wrapper that validates JWTs and performs high-speed Redis denylist checks for banned users.
    *   **Authority Client**: Provides programmatic methods (`BanUser`, `UpdateRoles`, etc.) that communicate directly with the IAM service via its secure WebSocket API.
*   **`internal/utils`**: Reserved for shared utility functions and helper scripts.

---

## Directory Structure

```text
grxm-stack/
├── .env                # Defines PROJECT_ROOT for absolute paths to Go source code
├── docker-compose.yml  # The core orchestration file
├── build/              # Dockerfiles for building services
├── internal/           # Modular Go logic for grxm-webapp
│   ├── api/            # HTTP Handlers & API logic
│   ├── config/         # Multi-environment configuration loading
│   ├── db/             # Multi-provider DB Layer (MongoDB)
│   ├── iam/            # Zero-trust IAM Client & Auth Middleware
│   └── utils/          # Empty (reserved for shared utilities)
├── config/             # Bind-mounted configuration files
│   ├── iam-config.json
│   ├── webapp-config.json
│   └── nginx.conf
├── iam-keys/           # Auto-generated RSA keys persisted via volume mount (do not delete)
├── start.sh            # Builds and starts the environment in detached mode
├── stop.sh             # Safely stops the environment and removes networks
├── logs.sh             # Utility to tail logs for all or specific services
├── status.sh           # Prints a comprehensive health and security report
└── wipe_database.sh    # Destructive script to wipe MongoDB volumes and all user data
```

---

## Configuration

### `.env`
The `.env` file is crucial for mapping your local source code into the Docker environment. Ensure `PROJECT_ROOT` points to the base directory containing both the `grxm-iam` and `grxm-webapp` repositories.

### `config/` Directory
The files in the `config/` directory are bind-mounted directly into the running containers.
*   **`iam-config.json`**: Configures the IAM service (database connection, authority path, default roles, etc.).
*   **`webapp-config.json`**: Configures the webapp's behavior:
    *   `iam_host`: Points to the IAM service (e.g., `grxm-iam:8081`).
    *   `db_provider`: Specifies the active database engine (e.g., `"mongo"`).
    *   `mongo_uri`: The connection string for the MongoDB provider.
    *   `redis_host`: Points to the Redis instance for high-speed token denylisting.
*   **`nginx.conf`**: Defines the routing rules and blocks external access to the sensitive `/iam/api/v1/authority` WebSocket.

### Live Code Reloading (Assets)
The `grxm-webapp` service has its `static/` and `dynamic/` folders bind-mounted from the host (`$PROJECT_ROOT/grxm-webapp/...`). Any changes made to HTML, CSS, or JS files on your host machine will immediately reflect in the running container without requiring a rebuild. (Note: Changes to Go code still require a rebuild).

---

## Operational Scripts

We use shell scripts to manage the lifecycle of the stack. All scripts utilize `sudo docker-compose` to prevent permission issues with the Docker daemon.

### `start.sh`
Builds any changed Dockerfiles and starts the entire stack in the background (`-d`). It respects boot order dependencies, ensuring the webapp does not start until the IAM service passes its health check.
```bash
./start.sh
```

### `stop.sh`
Safely stops all running containers and removes the default network. It does **not** remove persistent volumes (database data or RSA keys).
```bash
./stop.sh
```

### `status.sh`
Provides a detailed snapshot of the environment:
*   Checks if containers are running.
*   Displays real-time CPU/Memory usage.
*   Pings the native `/health` endpoints of both the webapp and IAM services.
*   Verifies that the Nginx security block on the Authority WebSocket is active.
```bash
./status.sh
```

### `logs.sh`
Streams the logs from the containers.
```bash
./logs.sh          # Tail logs for all services
./logs.sh webapp   # Tail logs for a specific service (e.g., webapp, grxm-iam, mongo, nginx)
```

### `wipe_database.sh`
**WARNING: Destructive.** This script stops the environment and permanently deletes the Docker named volumes associated with the stack (specifically `mongo-data`). This completely resets the database and erases all IAM users.
```bash
./wipe_database.sh
```

---

## Boot Sequence & Dependencies

The `docker-compose.yml` is configured with strict startup dependencies:
1.  **MongoDB** starts first.
2.  **grxm-iam** starts, connects to MongoDB, and begins listening. A native Docker `healthcheck` verifies the `/health` endpoint is responding.
3.  **grxm-webapp** waits in a pending state until `grxm-iam` reports as `healthy`. Once healthy, the webapp boots and instantly establishes its WebSocket connection to the IAM Authority API to retrieve the RSA public key.
4.  **Nginx** starts once both Go services are up and routes traffic.