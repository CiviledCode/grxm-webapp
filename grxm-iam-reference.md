# grxm-iam: API & Integration Reference

This document serves as the comprehensive technical reference for integrating external applications and administrative backend services with the `grxm-iam` (Identity and Access Management) service. `grxm-iam` is designed as a highly modular, zero-trust authentication bearer service.

---

## 1. Core Architecture & Concepts

### Zero-Trust Bearer Model
Consumer applications (like your API servers or frontends) **must never** store sensitive Personally Identifiable Information (PII) such as passwords, emails, or phone numbers in their own databases.
Instead, when a user successfully authenticates through `grxm-iam`, the service returns a signed JSON Web Token (JWT) containing a unique `user_id`. Your consumer applications should only store this `user_id` as the primary identifier to link resources.

### Decentralized Token Verification
Consumer applications validate the authenticity and integrity of the JWT cryptographically using the IAM service's **public key**. Because the token signature guarantees it was minted by the IAM service, your APIs can instantly verify a user's identity and roles without needing to make network requests or query the central IAM database for every single API call.

### Object-Oriented Auth Methods
Authentication and registration are handled via dynamic "methods" requested in the JSON payload. Methods are built like blocks (e.g., combining an `EmailField` and `PasswordField` creates the `"email-password"` method).

---

## 2. Configuration (`config.json`)

The IAM service is entirely driven by `config.json`. This documentation is crucial for orchestration (e.g., creating `docker-compose.yml` files).

```json
{
    "server": {
        "host": "0.0.0.0",          // The interface to bind the HTTP server to. Use "0.0.0.0" for Docker.
        "port": 8080                // The port the API server listens on.
    },
    "database": {
        "uri": "mongodb://localhost:27017", // The MongoDB connection string. In Docker Compose, this would be e.g., "mongodb://mongo:27017"
        "database": "grxm_iam"      // The specific MongoDB database to use.
    },
    "authority": {
        "password": "change-this-in-production-123", // The critical master password required to access the WebSocket API.
        "path": "/api/v1/authority" // The path the WebSocket server is mounted on.
    },
    "id": {
        "type": "uid",              // Strategy for ID generation.
        "length": 32,               // Length of the generated User IDs.
        "charset": "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890" // Allowed characters in the User ID.
    },
    "token": {
        "type": "jwt",              // Token format (currently JWT).
        "bits": 2048,               // RSA key bit size.
        "algorithm": "RS256",       // Signing algorithm.
        "key_path": "./keys/newest" // Path to save/load the private (.pem) and public (.pub) keys. Mount this as a volume in Docker to persist keys across restarts.
    },
    // Validation rules for specific fields when used in auth methods:
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
    // Toggles and settings for the modular authentication methods:
    "auth_methods": {
        "email-password": { "verify": true },
        "sms-password": { "verify": true },
        "username-password": {
            "verify": true,
            "verify_sources": ["email", "sms"]
        }
    }
}
```

---

## 3. Public REST API

The IAM service exposes standard HTTP endpoints for client-facing user onboarding and authentication.

### `POST /api/v1/register`
Creates a new user account in the MongoDB database. The submitted password will be automatically hashed using `bcrypt`.

**Request Body:**
```json
{
  "type": "string", // The ID of the registration method (e.g., "email-password", "sms-password", "username-password")
  "fields": {
    // Dynamic fields required by the chosen method.
    // Example for "email-password":
    "email": "user@example.com",
    "password": "securepassword123!"
  }
}
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2...etc" // The signed JWT bearer token.
}
```

**Error Response (400 Bad Request):**
```json
{
  "success": false,
  "message": "email already in use" // Indicates validation failures, missing fields, or duplicate users.
}
```

### `POST /api/v1/login`
Authenticates an existing user and issues a bearer token. This endpoint queries MongoDB to verify the password hash and checks the user's `is_banned` status before issuing a token.

**Request Body:**
```json
{
  "type": "string", // The ID of the login method (e.g., "email-password")
  "fields": {
    "email": "user@example.com",
    "password": "securepassword123!"
  }
}
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2...etc" // The signed JWT bearer token.
}
```

**Error Response (401 Unauthorized):**
```json
{
  "success": false,
  "message": "invalid email or password" // Or "user is banned: <reason>"
}
```

---

## 4. JWT Token Structure

The tokens issued by `grxm-iam` are standard JWTs (signed with RSA/RS256). Consumer applications should decode the payload (using the IAM service's Public Key) to identify the user making the request.

**Standard Claims Included:**
*   `uid` (String): The cryptographically secure User ID. This is the primary key you should use to link resources in your consumer APIs.
*   `exp` (NumericDate): The expiration time of the token (Unix timestamp).

*(Note: In upcoming iterations, user roles and verification statuses will also be embedded into these claims).*

---

## 5. Authority WebSocket API

The Authority API is a persistent, secure WebSocket connection intended **only** for trusted backend administrative services to mutate user state (banning, role management) in real-time and retrieve cryptographic materials.

**Endpoint:** `ws://<iam-host>:<port>/api/v1/authority` (The path is configurable in `config.json`)

### Authentication
Connections are immediately terminated if not authenticated. Provide the authority password (from `config.json`) via one of two methods when establishing the connection:
1.  **HTTP Header:** `Authorization: Bearer <authority_password>`
2.  **Query Parameter:** `ws://.../authority?auth=<authority_password>`

### Real-Time Commands

Commands are sent as JSON messages over the open WebSocket connection. The IAM service executes these directly against the MongoDB layer.

**1. Fetch Public Key**
Retrieves the RSA Public Key needed by your API servers to validate incoming user tokens.
```json
// Request
{ "action": "public_key" }

// Response
{
  "success": true,
  "message": "-----BEGIN PUBLIC KEY-----\nMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAvp2E... \n-----END PUBLIC KEY-----"
}
```

**2. Ban User**
Instantly sets the user's `is_banned` flag in MongoDB to `true`, blocking all future login attempts.
```json
// Request
{
  "action": "ban",
  "payload": {
    "user_id": "UID_STRING_HERE",
    "reason": "Violated terms of service."
  }
}

// Response
{ "success": true, "message": "User banned successfully" }
```

**3. Unban User**
Restores access to a banned user.
```json
// Request
{
  "action": "unban",
  "payload": {
    "user_id": "UID_STRING_HERE"
  }
}

// Response
{ "success": true, "message": "User unbanned successfully" }
```

**4. Update Roles**
Modifies the roles assigned to a user in the database.
```json
// Request
{
  "action": "role",
  "payload": {
    "user_id": "UID_STRING_HERE",
    "roles": ["user", "admin", "moderator"]
  }
}

// Response
{ "success": true, "message": "Roles updated successfully" }
```
