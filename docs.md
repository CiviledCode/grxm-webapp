# Environment Architecture & Setup Reference

This document serves as the technical reference for setting up the full Docker Compose environment for the Template Webapp and its dependencies.

## Architecture Overview

The environment consists of three primary components:
1.  **MongoDB:** The database layer for the Identity and Access Management (IAM) service.
2.  **grxm-iam:** A zero-trust Authentication Bearer Service.
3.  **Template Webapp:** The main backend/frontend application written in Go.

---

## 1. MongoDB Service

*   **Image:** Standard `mongo` image (latest or a specific version like `mongo:6.0`).
*   **Port:** `27017` (Internal Docker network only; no need to expose to the host).
*   **Volumes:** Requires a persistent volume mounted to `/data/db`.

---

## 2. grxm-iam Service

This is the central authentication service.

*   **Port:** Exposes port `8081` (Internal and Host).
*   **Dependencies:** Must start *after* the MongoDB service is healthy.
*   **Volumes:**
    *   Requires a volume mounted for its RSA key pairs (e.g., `./iam-keys:/app/keys`).
    *   Requires its `config.json` mounted into the container.

### `grxm-iam` Configuration (`config.json`)
The IAM service requires a `config.json` file. Here is the required template for the Docker environment:

```json
{
    "server": {
        "host": "0.0.0.0",
        "port": 8081
    },
    "database": {
        "uri": "mongodb://mongo:27017",
        "database": "grxm_iam"
    },
    "authority": {
        "password": "docker-compose-secure-password-123",
        "path": "/iam/api/v1/authority"
    },
    "id": {
        "type": "uid",
        "length": 32,
        "charset": "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
    },
    "token": {
        "type": "jwt",
        "bits": 2048,
        "algorithm": "RS256",
        "key_path": "/app/keys/newest"
    },
    "email": {
        "username_min_length": 3,
        "username_max_length": 50,
        "username_charlist": "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890.-_",
        "domain_min_length": 4,
        "domain_max_length": 50,
        "domain_charlist": "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890.-_",
        "domain_whitelist": [],
        "domain_blacklist": []
    },
    "password": {},
    "sms": {},
    "auth_methods": {
        "email-password": { "verify": false }
    }
}
```

---

## 3. Template Webapp Service (Go)

This is the main application that serves the protected frontend and API. It communicates with `grxm-iam` to fetch cryptographic keys and validate user sessions.

*   **Build Context:** The root directory of the Go project (`.`).
*   **Port:** Exposes port `8080` (Internal and Host).
*   **Health Check:** A public `GET /health` endpoint is available and returns `{"status": "healthy"}` (HTTP 200). Use this for Docker or load balancer health checks.
*   **Dependencies:** Must start *after* the `grxm-iam` service is fully operational, as it immediately attempts to establish a WebSocket connection on boot to fetch the RSA public key.
*   **Environment Variables:**
    *   `API_CONFIG_LOCATION`: Should point to the location of its configuration file within the container (e.g., `/app/config.json`).
*   **Volumes:**
    *   Requires its `config.json` mounted into the container at the path specified by `API_CONFIG_LOCATION`.

### Webapp Configuration (`config.json`)
The Webapp requires a `config.json` file to know how to connect to the IAM service. Here is the required template for the Docker environment:

```json
{
    "port": "8080",
    "iam_host": "grxm-iam:8081",
    "iam_authority_password": "docker-compose-secure-password-123"
}
```
*Note: The `iam_host` must use the Docker Compose service name (e.g., `grxm-iam:8081`). The application assumes the IAM service is reachable via `/iam/` routing.*

---

## Boot Sequence & Critical Constraints

When orchestrating this environment, ensure the following startup order and health checks:

1.  **MongoDB:** Starts first.
2.  **grxm-iam:** Waits for MongoDB to be healthy before starting.
3.  **Template Webapp:** **CRITICAL:** The Go webapp will crash on boot if it cannot connect to the `grxm-iam` Authority WebSocket. The Docker Compose file must ensure `grxm-iam` is fully ready (not just started) before the Webapp container is launched.
