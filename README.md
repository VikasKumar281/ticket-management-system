# Ticket Management System

A simple, secure RESTful backend API for managing support tickets, built
with Go.

## Author - **Vikas Kumar**

-   GitHub: https://github.com/VikasKumar281
-   Repository:
    https://github.com/VikasKumar281/ticket-management-system

## Live Deployment

The application is deployed as a Docker-based web service on Render.

**Application:**\
https://ticket-management-system-wc3q.onrender.com

**Health Check:**\
https://ticket-management-system-wc3q.onrender.com/health

Expected health response:

``` json
{
  "status": "ok"
}
```

------------------------------------------------------------------------

## 1. Project Overview

This project implements a small RESTful ticket management backend. A
user can:

1.  Register an account.
2.  Log in and receive a JWT.
3.  Create tickets.
4.  View only tickets owned by that user.
5.  View an individual owned ticket.
6.  Update the status of an owned ticket.
7.  Follow the fixed status workflow `open -> in_progress -> closed`.

The implementation deliberately avoids unnecessary complexity such as
admin roles, ticket assignment, comments, attachments, and a complex
database layer.

------------------------------------------------------------------------

## 2. Requirements Implemented

-   Go backend on port 8080 locally.
-   REST APIs with JSON request/response bodies.
-   JWT authentication.
-   Password hashing.
-   Protected ticket APIs.
-   Ownership-based authorization.
-   Status validation.
-   `open -> in_progress -> closed` workflow.
-   Closed tickets cannot be reopened.
-   Meaningful HTTP status codes.
-   Automated tests.
-   Dockerfile and Docker image.
-   Public deployment on Render.
-   Public `/health` endpoint.
-   Environment-based JWT secret configuration.

------------------------------------------------------------------------

## 3. Technology Stack

  Technology           Purpose
  -------------------- -------------------------------
  Go 1.22              Backend language
  `net/http`           HTTP server and REST API
  JSON                 API format
  HMAC-SHA256          JWT signing
  PBKDF2-HMAC-SHA256   Password hashing
  `sync.RWMutex`       Thread-safe in-memory storage
  Docker               Containerization
  Alpine Linux         Lightweight runtime
  Render               Cloud deployment
  Git/GitHub           Version control

The project uses the Go standard library and intentionally has no
unnecessary third-party dependencies.

------------------------------------------------------------------------

## 4. Project Structure

``` text
ticket-management-system/
├── main.go
├── go.mod
├── Dockerfile
├── .dockerignore
├── .env.example
├── README.md
├── internal/
│   ├── auth/
│   │   └── auth.go
│   ├── handlers/
│   │   └── auth.go
│   ├── middleware/
│   │   └── auth.go
│   ├── models/
│   │   └── models.go
│   ├── store/
│   │   └── store.go
│   └── utils/
│       └── validation.go
└── *_test.go
```

### Responsibilities

-   `main.go` --- application startup and HTTP server setup.
-   `internal/models` --- user, ticket, and status models.
-   `internal/auth` --- password hashing and JWT functionality.
-   `internal/middleware` --- authentication middleware.
-   `internal/handlers` --- HTTP request handling.
-   `internal/store` --- in-memory user/ticket storage.
-   `internal/utils` --- validation helpers.
-   `Dockerfile` --- multi-stage container build.
-   `.env.example` --- example environment configuration.

------------------------------------------------------------------------

## 5. Architecture

The application follows a small layered design:

``` text
Client
  |
  v
HTTP Router / Handlers
  |
  +---- Authentication
  |       +---- Password Hashing
  |       +---- JWT Creation / Verification
  |
  +---- Authentication Middleware
  |
  +---- Ticket Handlers
  |
  v
Thread-Safe In-Memory Store
  |
  +---- Users
  +---- Tickets
```

The separation keeps authentication, HTTP handling, models,
authorization, validation, and storage understandable without
introducing a large framework.

------------------------------------------------------------------------

## 6. Data Models

### User

A user contains:

``` text
ID
Email
Password Hash
Created At
```

The original password is never stored.

### Ticket

A ticket contains:

``` text
ID
User ID
Title
Description
Status
Created At
Updated At
```

`User ID` establishes ownership and is used for authorization checks.

------------------------------------------------------------------------

## 7. Authentication

### Registration flow

``` text
POST /auth/register
        |
        v
Validate request
        |
        v
Hash password
        |
        v
Create user
        |
        v
Return user information
```

### Login flow

``` text
POST /auth/login
        |
        v
Find user
        |
        v
Verify password
        |
        v
Create JWT
        |
        v
Return token
```

Protected endpoints require:

``` http
Authorization: Bearer <JWT_TOKEN>
```

------------------------------------------------------------------------

## 8. Password Security

Passwords are hashed using **PBKDF2-HMAC-SHA256** with:

-   A random salt.
-   100,000 iterations.
-   A derived password hash.

Conceptually:

``` text
Password + Random Salt
          |
          v
PBKDF2-HMAC-SHA256
          |
          v
Stored Password Hash
```

During login, the submitted password is derived using the stored salt
and compared with the stored hash.

The plaintext password is therefore not stored in the application data.

------------------------------------------------------------------------

## 9. JWT Security

JWT authentication uses an HMAC-SHA256 signing mechanism.

The signing secret is read from:

``` text
JWT_SECRET
```

For deployment, the secret is configured in Render Environment
Variables.

The production secret is not committed to GitHub and should never be
placed in `README.md` or `.env.example`.

------------------------------------------------------------------------

## 10. Authorization and Ticket Ownership

Authentication identifies the current user. Authorization verifies
whether that user is allowed to access a ticket.

Example:

``` text
User A
 ├── Ticket 1
 └── Ticket 2

User B
 └── Ticket 3
```

User A can access Ticket 1 and Ticket 2, but cannot access or modify
Ticket 3.

Ownership checks are applied to:

-   `GET /tickets`
-   `GET /tickets/{id}`
-   `PATCH /tickets/{id}/status`

This prevents one authenticated user from modifying another user's
ticket.

------------------------------------------------------------------------

## 11. Ticket Status Workflow

Supported statuses:

``` text
open
in_progress
closed
```

Allowed transitions:

``` text
open
  |
  v
in_progress
  |
  v
closed
```

Valid:

``` text
open -> in_progress
in_progress -> closed
```

Invalid:

``` text
closed -> open
closed -> in_progress
in_progress -> open
```

A closed ticket cannot be reopened.

Example error:

``` json
{
  "error": "invalid status transition from closed to open"
}
```

------------------------------------------------------------------------

# 12. API Endpoints

  Method   Endpoint                 Authentication   Purpose
  -------- ------------------------ ---------------- --------------------------
  GET      `/health`                No               Health check
  POST     `/auth/register`         No               Register user
  POST     `/auth/login`            No               Login and receive JWT
  POST     `/tickets`               Yes              Create ticket
  GET      `/tickets`               Yes              List own tickets
  GET      `/tickets/{id}`          Yes              Get own ticket
  PATCH    `/tickets/{id}/status`   Yes              Update own ticket status

------------------------------------------------------------------------

## 13. Health Check

### Request

``` http
GET /health
```

Local:

``` text
http://localhost:8080/health
```

Production:

``` text
https://ticket-management-system-wc3q.onrender.com/health
```

Response:

``` json
{
  "status": "ok"
}
```

------------------------------------------------------------------------

## 14. Register User

### Request

``` http
POST /auth/register
Content-Type: application/json
```

Body:

``` json
{
  "email": "user@example.com",
  "password": "Test@12345"
}
```

Successful response:

``` text
201 Created
```

Example:

``` json
{
  "id": "user-id",
  "email": "user@example.com",
  "created_at": "2026-09-12T00:00:00Z"
}
```

------------------------------------------------------------------------

## 15. Login

### Request

``` http
POST /auth/login
Content-Type: application/json
```

Body:

``` json
{
  "email": "user@example.com",
  "password": "Test@12345"
}
```

The response contains a JWT token.

Use it on protected APIs:

``` http
Authorization: Bearer <JWT_TOKEN>
```

------------------------------------------------------------------------

## 16. Create Ticket

### Request

``` http
POST /tickets
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json
```

Body:

``` json
{
  "title": "Login issue",
  "description": "I am unable to login to my account"
}
```

The authenticated user's ID is automatically associated with the ticket.

------------------------------------------------------------------------

## 17. List Own Tickets

### Request

``` http
GET /tickets
Authorization: Bearer <JWT_TOKEN>
```

Only tickets belonging to the authenticated user are returned.

------------------------------------------------------------------------

## 18. Get Ticket By ID

### Request

``` http
GET /tickets/{id}
Authorization: Bearer <JWT_TOKEN>
```

The API verifies ownership before returning the ticket.

------------------------------------------------------------------------

## 19. Update Ticket Status

### Request

``` http
PATCH /tickets/{id}/status
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json
```

Body:

``` json
{
  "status": "in_progress"
}
```

Then:

``` json
{
  "status": "closed"
}
```

Invalid transitions are rejected.

------------------------------------------------------------------------

## 20. HTTP Status Codes

  Status                        Meaning
  ----------------------------- --------------------------------------------
  `200 OK`                      Successful request
  `201 Created`                 Resource created
  `400 Bad Request`             Invalid input or status transition
  `401 Unauthorized`            Missing/invalid authentication
  `403 Forbidden`               Authenticated user does not own resource
  `404 Not Found`               Resource not found
  `405 Method Not Allowed`      HTTP method is unsupported
  `409 Conflict`                Resource/request conflict where applicable
  `500 Internal Server Error`   Unexpected server error

------------------------------------------------------------------------

# 21. Validation

The application validates incoming requests before processing them.

Examples:

-   Email is required and validated.
-   Password is required.
-   Password must satisfy the implemented minimum length.
-   Ticket title is required.
-   Ticket description is required.
-   Ticket status must be one of the supported statuses.
-   Status transitions must follow the allowed workflow.

Invalid requests are rejected rather than stored.

------------------------------------------------------------------------

# 22. In-Memory Storage

The project uses an in-memory store because the assignment explicitly
allows it and does not require a database.

The store maintains users and tickets in memory and protects shared
state with `sync.RWMutex`.

### Why this approach?

It keeps the implementation:

-   Small.
-   Easy to review.
-   Easy to test.
-   Free of database setup.
-   Focused on authentication, authorization, and API design.

### Limitation

Data is lost when the application restarts or is redeployed:

``` text
Application restart
       |
       v
In-memory data cleared
```

This is an intentional trade-off for this assignment.

------------------------------------------------------------------------

# 23. Thread Safety

Because the server can process concurrent HTTP requests, shared
in-memory data must be protected.

`sync.RWMutex` is used to coordinate access:

``` text
Concurrent Requests
        |
        v
     RWMutex
     /       Reads   Writes
```

Read operations can use read locking, while modifications use write
locking.

------------------------------------------------------------------------

# 24. Local Setup

## Prerequisites

Install:

-   Go
-   Git
-   Docker (optional, for container testing)

Verify Go:

``` bash
go version
```

------------------------------------------------------------------------

## Clone Repository

``` bash
git clone https://github.com/VikasKumar281/ticket-management-system.git
cd ticket-management-system
```

------------------------------------------------------------------------

## Install/Download Modules

The project does not require external Go dependencies.

You can still run:

``` bash
go mod download
```

------------------------------------------------------------------------

## Run Tests

``` bash
go test ./...
```

------------------------------------------------------------------------

## Run Application

``` bash
go run .
```

The local server runs on:

``` text
http://localhost:8080
```

------------------------------------------------------------------------

# 25. Local Health Test

Windows PowerShell:

``` powershell
curl.exe http://localhost:8080/health
```

Expected:

``` json
{
  "status": "ok"
}
```

------------------------------------------------------------------------

# 26. Complete API Testing Flow

The complete workflow is:

``` text
1. Register
      |
      v
2. Login
      |
      v
3. Copy JWT
      |
      v
4. Create Ticket
      |
      v
5. GET /tickets
      |
      v
6. GET /tickets/{id}
      |
      v
7. open -> in_progress
      |
      v
8. in_progress -> closed
      |
      v
9. Verify closed -> open is rejected
```

A second user can be used to verify ownership protection.

------------------------------------------------------------------------

# 27. Dockerization

The project uses a multi-stage Docker build.

### Builder stage

``` text
golang:1.22-alpine
```

The Go application is compiled for Linux with:

``` text
CGO_ENABLED=0
GOOS=linux
```

### Runtime stage

``` text
alpine:3.20
```

Only the compiled binary is copied into the runtime image.

This avoids shipping the full Go build environment in the final image.

The runtime container also creates and uses a non-root application user.

------------------------------------------------------------------------

# 28. Build Docker Image

``` bash
docker build -t ticket-management-system .
```

This creates:

``` text
ticket-management-system:latest
```

------------------------------------------------------------------------

# 29. Run Docker Container

``` bash
docker run --rm -p 8080:8080 --name ticket-management-system ticket-management-system
```

With an explicit JWT secret:

``` bash
docker run --rm -p 8080:8080 -e JWT_SECRET=your-secret --name ticket-management-system ticket-management-system
```

PowerShell:

``` powershell
docker run --rm -p 8080:8080 -e JWT_SECRET=your-secret --name ticket-management-system ticket-management-system
```

------------------------------------------------------------------------

# 30. Test Docker Container

After starting the container:

``` bash
curl http://localhost:8080/health
```

Expected:

``` json
{
  "status": "ok"
}
```

The Docker image and container were tested locally before deployment.

------------------------------------------------------------------------

# 31. Environment Variables

Supported configuration includes:

``` text
JWT_SECRET
PORT
```

Example `.env.example`:

``` text
JWT_SECRET=your-secret-here
PORT=8080
```

### Production secret

Never commit the real `JWT_SECRET`.

For Render, configure it through:

``` text
Render Dashboard
  -> Service
  -> Environment
  -> Environment Variables
```

Only the variable name and placeholder belong in the repository
documentation.

------------------------------------------------------------------------

# 32. Deployment on Render

The application is deployed from GitHub using the repository's
Dockerfile.

Deployment process:

``` text
GitHub Repository
       |
       v
Render
       |
       v
Dockerfile detected
       |
       v
Docker image built
       |
       v
Container started
       |
       v
Public HTTPS API
```

The application listens on the port provided by the deployment
environment.

The production JWT secret is configured through Render Environment
Variables.

------------------------------------------------------------------------

# 33. Production URLs

### Application

https://ticket-management-system-wc3q.onrender.com

### Health Check

https://ticket-management-system-wc3q.onrender.com/health

### GitHub Repository

https://github.com/VikasKumar281/ticket-management-system

------------------------------------------------------------------------

# 34. Deployment Verification

The public health endpoint was verified successfully:

``` http
GET /health
```

Response:

``` json
{
  "status": "ok"
}
```

The deployed registration API was also verified using:

``` http
POST /auth/register
```

with a valid JSON request and received:

``` text
201 Created
```

The deployed authentication and ticket API flow was subsequently tested
using the public Render URL.

------------------------------------------------------------------------

# 35. Security Measures

The project implements:

### Password hashing

Plaintext passwords are not stored.

### JWT authentication

Protected ticket APIs require a valid JWT.

### Ownership authorization

Users can access and modify only their own tickets.

### Status restrictions

Closed tickets cannot be reopened.

### Secret management

Production JWT signing secrets are supplied through environment
variables.

### Non-root container

The Docker runtime uses a dedicated non-root application user.

------------------------------------------------------------------------

# 36. Testing Performed

## Authentication

-   Registration with valid credentials.
-   Login with valid credentials.
-   JWT generation.
-   Protected endpoint authentication.

## Ticket operations

-   Ticket creation.
-   Ticket listing.
-   Ticket retrieval by ID.
-   Ticket status update.

## Authorization

-   Accessing own tickets.
-   Preventing access to another user's tickets.
-   Preventing modification of another user's tickets.

## Status workflow

``` text
open -> in_progress       PASS
in_progress -> closed     PASS
closed -> open            REJECTED
```

## Infrastructure

-   `go test ./...`
-   Docker image build.
-   Docker container startup.
-   Local `/health`.
-   Public Render `/health`.
-   Public registration endpoint.
-   Public authentication/ticket flow.

------------------------------------------------------------------------

# 37. Design Decisions

## Why Go standard library?

The assignment does not require a web framework. `net/http` is
sufficient for the required REST APIs and keeps the implementation
lightweight.

## Why in-memory storage?

The assignment allows in-memory storage. It avoids unnecessary database
infrastructure and keeps the focus on the required backend behavior.

## Why JWT?

JWT provides a straightforward stateless authentication mechanism for
protected REST endpoints.

## Why PBKDF2?

PBKDF2 provides salted password derivation with a deliberately expensive
iteration count and can be implemented without adding an external
dependency.

## Why no admin?

The assignment does not require administrative functionality.

## Why no assignment/comments?

Ticket assignment, comments, and other advanced features are outside the
required scope.

------------------------------------------------------------------------

# 38. Assumptions

1.  Email addresses are handled case-insensitively.
2.  Passwords must satisfy the application's minimum validation rule.
3.  JWT tokens have a fixed expiration period implemented by the
    service.
4.  There is no admin role.
5.  There is no ticket assignment.
6.  There are no comments or attachments.
7.  Ticket data is stored in memory.
8.  In-memory data is lost after application restart/redeployment.
9.  Closed tickets cannot be reopened.
10. Production secrets are provided through environment variables.

------------------------------------------------------------------------

# 39. Limitations

This assignment implementation intentionally does not provide:

-   Database persistence.
-   Admin dashboard.
-   Ticket assignment.
-   Comments.
-   File attachments.
-   Email verification.
-   Password reset.
-   Advanced search.
-   Production-grade rate limiting.
-   Refresh-token management.
-   Full observability stack.

These are possible future enhancements rather than current requirements.

------------------------------------------------------------------------

# 40. Future Improvements

If the project were extended beyond the assignment, it could include:

-   PostgreSQL persistence.
-   Database migrations.
-   Refresh tokens.
-   Token revocation.
-   Rate limiting.
-   Structured logging.
-   OpenAPI/Swagger documentation.
-   Pagination and filtering.
-   Search.
-   Admin/support-agent roles.
-   Ticket assignment.
-   Comments and attachments.
-   Email notifications.
-   CI/CD.
-   Monitoring and alerting.

------------------------------------------------------------------------

# 41. Quick Start

### Run directly with Go

``` bash
git clone https://github.com/VikasKumar281/ticket-management-system.git
cd ticket-management-system
go test ./...
go run .
```

Then open:

``` text
http://localhost:8080/health
```

### Run with Docker

``` bash
docker build -t ticket-management-system .
docker run --rm -p 8080:8080 -e JWT_SECRET=your-secret ticket-management-system
```

Then:

``` text
http://localhost:8080/health
```

------------------------------------------------------------------------

# 42. Submission Information

**GitHub Repository**

https://github.com/VikasKumar281/ticket-management-system

**Deployed Application**

https://ticket-management-system-wc3q.onrender.com

**Public Health Check**

https://ticket-management-system-wc3q.onrender.com/health

------------------------------------------------------------------------

# 43. Final Implementation Checklist

``` text
User Registration              ✓
User Login                     ✓
JWT Authentication             ✓
Password Hashing               ✓
Create Ticket                  ✓
List Own Tickets               ✓
Get Own Ticket                 ✓
Ownership Authorization        ✓
Ticket Status Updates          ✓
Status Transition Validation   ✓
Closed Ticket Reopen Block     ✓
Health Endpoint                ✓
Input Validation               ✓
Meaningful HTTP Status Codes   ✓
Automated Tests                ✓
Dockerfile                     ✓
Docker Build                   ✓
Docker Run                     ✓
Render Deployment              ✓
Public Health Check            ✓
Production JWT Secret          ✓
```

The project is intentionally simple and focused on the required backend
functionality, while keeping authentication, authorization, ticket
lifecycle management, testing, Dockerization, and deployment clearly
separated and reviewable.

