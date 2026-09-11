# Ticket System (Golang Backend Intern Assignment)

A small REST API where a user can register, log in, create tickets, view
only their own tickets, and move a ticket through its status lifecycle
(`open -> in_progress -> closed`).

## Tech / Design Choices

- **Language:** Go 1.22, using the standard library `net/http` with its
  built-in method+path pattern routing (`mux.HandleFunc("GET /tickets/{id}", ...)`)
  — no web framework needed for an API this size.
- **Storage:** In-memory (protected by a mutex), as explicitly permitted by
  the assignment brief. All state resets on restart; see "Assumptions" below.
- **Auth:** JWT (HS256), implemented directly on top of `crypto/hmac` /
  `crypto/sha256` in `internal/auth/jwt.go`.
- **Passwords:** Hashed with a from-scratch PBKDF2-HMAC-SHA256
  implementation (100,000 iterations, random 16-byte salt per user) in
  `internal/auth/password.go`. Verified in constant time via
  `crypto/subtle.ConstantTimeCompare`.
- **Zero third-party dependencies.** `go.mod` has no `require` entries at
  all. This was a deliberate engineering trade-off for a two-day, tightly
  time-boxed assignment with a mandatory Docker + free-hosting deployment
  step: no dependency means `go build` / `docker build` never depends on
  reaching a module proxy, there's no `go.sum` to get wrong, and the image
  builds identically anywhere. In a larger production codebase I would
  reach for `golang.org/x/crypto/bcrypt` and `github.com/golang-jwt/jwt`
  instead of hand-rolling this — noting that trade-off explicitly here.

## Project Layout

```
.
├── main.go                          # entrypoint: config, routing, server
├── main_test.go                     # end-to-end API tests
├── internal/
│   ├── auth/
│   │   ├── jwt.go                   # HS256 JWT generate/parse
│   │   ├── password.go              # PBKDF2 password hash/verify
│   │   └── auth_test.go
│   ├── models/
│   │   └── models.go                # User, Ticket, status transition rules
│   ├── store/
│   │   └── store.go                 # thread-safe in-memory data store
│   ├── handlers/
│   │   ├── health.go
│   │   ├── auth.go                  # /auth/register, /auth/login
│   │   └── tickets.go               # /tickets... (create/list/get/status)
│   ├── middleware/
│   │   ├── auth.go                  # JWT bearer-token middleware
│   │   └── logging.go               # request logging + panic recovery
│   └── utils/
│       ├── response.go              # JSON response/error helpers
│       └── id.go                    # dependency-free UUID-style ID gen
├── Dockerfile
├── .dockerignore
├── .env.example
├── .gitignore
└── Makefile
```

## API Contract

| Method | Endpoint                | Auth required | Purpose                       |
|--------|--------------------------|:---:|--------------------------------|
| GET    | `/health`                | No  | Health check                   |
| POST   | `/auth/register`         | No  | Register a new user            |
| POST   | `/auth/login`            | No  | Log in, returns a JWT          |
| POST   | `/tickets`               | Yes | Create a ticket                |
| GET    | `/tickets`               | Yes | List the caller's own tickets  |
| GET    | `/tickets/{id}`          | Yes | Get one of the caller's own tickets |
| PATCH  | `/tickets/{id}/status`   | Yes | Update status of an owned ticket |

Authenticated requests must send `Authorization: Bearer <token>`.

### Status flow

```
open -> in_progress -> closed
```

- Skipping a step (e.g. `open -> closed` directly) is rejected with `400`.
- A `closed` ticket can never move back to `open` or `in_progress` — rejected with `400`.
- Setting a ticket to its current status is rejected with `400`.

### Example requests

```bash
# Register
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"hunter22"}'

# Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"hunter22"}'
# -> {"token":"<JWT>"}

TOKEN="<paste JWT here>"

# Create a ticket
curl -X POST http://localhost:8080/tickets \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Server down","description":"prod is on fire"}'

# List my tickets
curl http://localhost:8080/tickets -H "Authorization: Bearer $TOKEN"

# Get one ticket
curl http://localhost:8080/tickets/<id> -H "Authorization: Bearer $TOKEN"

# Move status forward
curl -X PATCH http://localhost:8080/tickets/<id>/status \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress"}'
```

### HTTP status codes used

| Code | When |
|---|---|
| 200 | Successful GET/PATCH |
| 201 | Successful register / ticket creation |
| 400 | Validation error, invalid/missing status, illegal status transition |
| 401 | Missing/invalid/expired token, bad login credentials |
| 403 | Authenticated user does not own the requested ticket |
| 404 | Ticket (or route) not found |
| 409 | Email already registered |
| 500 | Unexpected server error |

## Running Locally (no Docker)

Requires Go 1.22+.

```bash
go run .
# or
go build -o bin/ticket-system .
./bin/ticket-system
```

Optional environment variables (see `.env.example`):

- `PORT` — defaults to `8080`.
- `JWT_SECRET` — **set this** for anything beyond quick local testing; if
  unset, the app logs a warning and uses an insecure development default.

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

## Running Tests

```bash
go test ./... -v
```

Includes a full integration test (`main_test.go`) that walks: register →
duplicate-register conflict → login → wrong-password rejection →
unauthenticated request rejection → create ticket → list tickets →
cross-user 403 checks → full `open -> in_progress -> closed` transition →
rejecting skip/reopen transitions → 404 on unknown ticket. Plus unit tests
for password hashing and JWT generation/validation.

## Running with Docker (Local Run Contract)

```bash
docker build -t ticket-system .
docker run -p 8080:8080 -e JWT_SECRET=some-long-random-secret ticket-system
curl http://localhost:8080/health
```

The Dockerfile is a multi-stage build: a `golang:1.22-alpine` build stage
compiles a static (`CGO_ENABLED=0`) binary, which is then copied into a
minimal `alpine:3.20` runtime image running as a non-root user. No
dependencies need to be fetched by Docker beyond the two base images
(the Go module itself has none), so the build is fast and reliable.

## Deploying (Free-Tier Hosting)

Any platform that can run a Docker image works. Two straightforward
free-tier options:

**Render.com**
1. Push this repo to GitHub.
2. New → Web Service → connect the repo → environment: **Docker**.
3. Set the `JWT_SECRET` environment variable in the dashboard.
4. Render auto-detects the `Dockerfile` and exposed port `8080`.
5. Once deployed, `https://<your-service>.onrender.com/health` should
   return `{"status":"ok"}` publicly.

**Fly.io**
```bash
fly launch --dockerfile Dockerfile --no-deploy
fly secrets set JWT_SECRET=some-long-random-secret
fly deploy
```

After deploying, update this README (or your submission notes) with:
- GitHub repository link
- Deployed application URL
- Public health check URL (e.g. `https://<url>/health`)

## Assumptions

- In-memory storage is acceptable per the assignment scope; **all data is
  lost on process restart**. This is a deliberate simplicity trade-off for
  a two-day assignment with no persistence requirement — swapping in
  SQLite/Postgres later would only mean rewriting `internal/store`, since
  handlers depend only on the `Store` interface-shaped methods, not on
  storage details.
- Email uniqueness is case-insensitive (`Alice@x.com` and `alice@x.com`
  are treated as the same account).
- Passwords must be at least 6 characters (a reasonable minimum for a
  test assignment; not intended as production password policy).
- JWTs are valid for 24 hours and carry no refresh-token flow, since none
  was requested.
- No admin role, ticket assignment, or comments — explicitly out of scope
  per the brief.
- A request to set a ticket to its *current* status is treated as an
  error (`400`) rather than a no-op success, to keep the status-transition
  contract strict and unambiguous for automated grading.
