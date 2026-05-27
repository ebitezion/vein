# Jarakey RBAC MVP - Technical Design & Proposal

> **Golang Backend Engineer Take-Home Challenge**
> Submitted by: Ogochukwu Zion Ebite
> Framework: Vein (custom Go backend framework)
> Base framework repo: [ebitezion/vein](https://github.com/ebitezion/vein)

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Problem Statement](#2-problem-statement)
3. [Functional Requirements](#3-functional-requirements)
4. [Non-Functional Requirements](#4-non-functional-requirements)
5. [System Design](#5-system-design)
   - [Architecture Overview](#51-architecture-overview)
   - [Component Breakdown](#52-component-breakdown)
   - [Request Lifecycle](#53-request-lifecycle)
   - [Data Model](#54-data-model)
   - [Middleware Chain](#55-middleware-chain)
6. [API Contract](#6-api-contract)
7. [Permission Drift Simulation](#7-permission-drift-simulation)
8. [Security & Trust-Boundary Decisions](#8-security--trust-boundary-decisions)
9. [Testing Strategy](#9-testing-strategy)
10. [Implementation Plan (File-by-File)](#10-implementation-plan-file-by-file)
11. [Limitations & Future Evolution](#11-limitations--future-evolution)
12. [Theory & Explanation (README Answers)](#12-theory--explanation-readme-answers)
13. [Postman Collection](#13-postman-collection)
14. [Verification Results](#14-verification-results)
15. [How to Run](#15-how-to-run)

---

## 1. Executive Summary

This document presents the design and implementation plan for the Jarakey RBAC MVP challenge, built by adapting **Vein** — a custom Go backend framework I developed from scratch.

The core challenge is: **enforce authorization using fresh server-side state on every request**, particularly after role changes (permission drift), while maintaining strict trust boundaries and clean architectural separation between authentication, context injection, and authorization.

Vein already provides strong backend primitives (JWT auth, middleware chaining, security defaults, error handling, and test scaffolding), so the work here is focused precisely on what the challenge evaluates:

- Multi-tenant RBAC correctness
- Active context handling via `X-Estate-ID`
- Permission drift safety
- Authorization outside HTTP handlers

---

## 2. Problem Statement

Jarakey is a **multi-tenant access-control system** where:

- A **user** may belong to multiple **estates** and hold different roles per estate.
- Roles can change at any time (permission drift).
- The active estate context is supplied per-request via the `X-Estate-ID` header.
- Authorization must be enforced **server-side**, never relying on stale or client-supplied role data.

### Challenge Requirements Summary

| Requirement | Description |
|---|---|
| Endpoint | `POST /v1/gate/open` |
| Auth | User is pre-authenticated (simulated JWT) |
| Active Context | `X-Estate-ID` header |
| Success | `200 OK` if user is Admin for active estate |
| Failure | `403 Forbidden` otherwise |
| Drift | Role downgrade must take effect immediately on next request |
| Boundary | Authorization logic must NOT live inside HTTP handlers |

---

## 3. Functional Requirements

### FR-1: Active Context Injection
- The system MUST read `X-Estate-ID` from every request header.
- The system MUST reject requests missing this header with `400 Bad Request`.
- The estate ID MUST be injected into the request context for downstream use.

### FR-2: Identity Extraction
- The system MUST extract user identity from a validated JWT token.
- The `sub` (subject) claim of the JWT MUST be used as the canonical user identifier.
- No identity data from the request body or headers (other than `Authorization`) may be trusted.

### FR-3: Role Evaluation
- The system MUST evaluate the user's role for the active estate from **current in-memory server-side state**.
- Role evaluation MUST happen on every request; results MUST NOT be cached.
- Only users with the `admin` role MUST be permitted through `POST /v1/gate/open`.

### FR-4: Permission Drift
- The system MUST reflect role changes immediately.
- A user demoted from `admin` to `resident` MUST receive `403` on their very next request, even if their JWT has not changed.

### FR-5: Authorization Boundary
- Authorization logic MUST be enforced in middleware, not inside HTTP handlers.
- HTTP handlers MUST remain thin (orchestration + response only).

### FR-6: Gate Endpoint
- `POST /v1/gate/open` MUST return `200 OK` with `{ "message": "gate opened" }` for authorized requests.
- The handler itself MUST contain zero role or permission checks.

---

## 4. Non-Functional Requirements

| Category | Requirement |
|---|---|
| **Concurrency Safety** | In-memory membership store must be protected by `sync.RWMutex` to support concurrent reads/writes safely |
| **Security** | JWT must be validated (signature, issuer, audience, expiry) before any identity is trusted |
| **Separation of Concerns** | AuthN, context injection, and AuthZ must be distinct, composable middleware layers |
| **Testability** | All authorization behavior (success, failure, drift, tenant boundary) must be verifiable via unit/integration tests |
| **Observability** | Every request must be logged with request ID, method, path, status, and latency (provided by Vein) |
| **Maintainability** | No authorization logic may leak into handlers; future endpoints should be securable by composing existing middleware |
| **Performance** | In-memory role lookup is O(1); no I/O latency for MVP |
| **Portability** | Design must be replaceable — in-memory store can be swapped for a durable store without changing middleware contracts |

---

## 5. System Design

### 5.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP Request                             │
│          POST /v1/gate/open                                     │
│          Authorization: Bearer <jwt>                            │
│          X-Estate-ID: estate-1                                  │
└─────────────────────┬───────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Vein Middleware Stack                          │
│                                                                 │
│  ┌──────────────┐   ┌────────────────┐   ┌──────────────────┐  │
│  │  Recovery +  │──▶│  Rate Limit +  │──▶│  CORS + Security │  │
│  │  Request ID  │   │  Idempotency   │   │  Headers         │  │
│  └──────────────┘   └────────────────┘   └──────────────────┘  │
│                                                  │              │
│                                                  ▼              │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              authenticate (AuthN Middleware)              │  │
│  │  - Validate JWT (sig, iss, aud, exp)                     │  │
│  │  - Inject user_id (claims.Subject) → context             │  │
│  │  - 401 on invalid/missing/expired token                  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                  │              │
│                                                  ▼              │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │           withActiveEstate (Context Middleware)           │  │
│  │  - Read X-Estate-ID header                               │  │
│  │  - Validate non-empty                                    │  │
│  │  - Inject estate_id → context                            │  │
│  │  - 400 on missing/invalid header                         │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                  │              │
│                                                  ▼              │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │        requireEstateRole("admin") (AuthZ Middleware)     │  │
│  │  - Read user_id + estate_id from context                 │  │
│  │  - Fetch CURRENT role from EstateRoleStore               │  │
│  │  - 403 if role ≠ "admin"                                │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                  │              │
└──────────────────────────────────────────────────┼─────────────┘
                                                   │
                                                   ▼
┌──────────────────────────────────────────────────────────────┐
│                    openGate Handler                           │
│  - No role checks                                            │
│  - Returns 200 + { "message": "gate opened" }               │
└──────────────────────────────────────────────────────────────┘
```

### 5.2 Component Breakdown

#### EstateRoleStore (`estate_roles.go`)
Thread-safe in-memory store for user-estate-role mappings.

```
EstateRoleStore
├── mu         sync.RWMutex
├── roles      map[userID]map[estateID]role (string)
├── GetRole(userID, estateID) → (string, bool)
└── SetRole(userID, estateID, role)
```

**Seed data:**
```
"user-1" → "estate-1" → "admin"
```

#### authenticate Middleware (`auth.go`)
- Validates JWT (HS256, issuer, audience, expiry)
- Extracts `claims.Subject` → injects as `userIDContextKey` in context
- Returns `401` on any token failure

#### withActiveEstate Middleware (`middleware.go`)
- Reads `X-Estate-ID` request header
- Validates presence and non-empty
- Injects as `estateIDContextKey` in context
- Returns `400` on missing value

#### requireEstateRole Middleware (`authz_estate.go`)
- Reads `user_id` and `estate_id` from context
- Calls `EstateRoleStore.GetRole(userID, estateID)`
- Returns `403` unless role exactly equals `"admin"`
- **This is the sole authorization decision point**

#### openGate Handler (`gate.go`)
- Zero authorization logic
- Returns `200 OK` with `{ "message": "gate opened" }`
- Only reached after all middleware has passed

### 5.3 Request Lifecycle

```
Incoming Request
      │
      ▼
[1] Recover + RequestID + Logger + Security Headers + CORS + Rate Limiter
      │
      ▼ (always passes unless rate-limited)
[2] authenticate
      │── JWT missing/invalid/expired ──► 401 Unauthorized
      │
      ▼ (user_id injected into context)
[3] withActiveEstate
      │── X-Estate-ID missing/empty ──► 400 Bad Request
      │
      ▼ (estate_id injected into context)
[4] requireEstateRole("admin")
      │── role ≠ "admin" ──────────────► 403 Forbidden
      │── user not found in store ──────► 403 Forbidden
      │
      ▼ (authorized)
[5] openGate handler
      │
      └──────────────────────────────────► 200 OK { "message": "gate opened" }
```

### 5.4 Data Model

```
UserID      string    // JWT sub claim (e.g. "user-1")
EstateID    string    // X-Estate-ID header value (e.g. "estate-1")
Role        string    // "admin" | "resident" | "guest" | ...

MembershipKey = (UserID, EstateID)
MembershipRecord {
    UserID   string
    EstateID string
    Role     string
}

// In-memory representation:
roles: map[UserID → map[EstateID → Role]]
```

### 5.5 Middleware Chain

The route is composed as:

```go
POST /v1/gate/open →
    authenticate(
        withActiveEstate(
            requireEstateRole("admin")(
                openGate
            )
        )
    )
```

Each layer has a single responsibility and fails fast with the appropriate HTTP status code, never allowing execution to proceed to the next layer unless its own check passes.

---

## 6. API Contract

### Endpoint

```
POST /v1/gate/open
```

### Request Headers

| Header | Required | Description |
|---|---|---|
| `Authorization` | Yes | `Bearer <jwt>` — valid signed JWT |
| `X-Estate-ID` | Yes | Active estate context identifier |
| `Content-Type` | No | Not required for this endpoint |

### Responses

#### 200 OK — Authorized

```json
{
  "message": "gate opened"
}
```

#### 400 Bad Request — Missing/Invalid Estate Header

```json
{
  "error": "X-Estate-ID header is required"
}
```

#### 401 Unauthorized — Invalid/Expired Token

```json
{
  "error": "unauthorized"
}
```

#### 403 Forbidden — Insufficient Role

```json
{
  "error": "forbidden"
}
```

Note:
- This challenge can be tested without DB setup.
- In DB-free mode, `POST /v1/auth/token` accepts:
  - `email`: `admin@vein.dev`
  - `password`: `VeinPass#2026!`
- `GET /v1/users` is DB-backed and returns `503` when DB is not configured.

---

## 7. Permission Drift Simulation

The drift scenario proves that authorization is evaluated from fresh server-side state — not from JWT claims and not from any cached value.

### Sequence

```
Step 1: Seed state
        roles["user-1"]["estate-1"] = "admin"

Step 2: First request
        POST /v1/gate/open
        Authorization: Bearer <token for user-1>
        X-Estate-ID: estate-1
        → 200 OK ✓

Step 3: Drift event (in memory)
        store.SetRole("user-1", "estate-1", "resident")

Step 4: Second request (SAME token, SAME estate)
        POST /v1/gate/open
        Authorization: Bearer <token for user-1>   ← token unchanged
        X-Estate-ID: estate-1
        → 403 Forbidden ✓

Proof: The JWT did not change. Only the server-side role changed.
       Authorization reflects current state, not stale token data.
```

### Why This Matters

A naive implementation might cache the role in the JWT claims or in a per-session cache keyed to the token. Either approach would allow a demoted user to continue accessing protected resources until their token expires — which could be hours or days.

By re-evaluating from the membership store on **every request**, demotion is effective **immediately**.

---

## 8. Security & Trust-Boundary Decisions

| Decision | Rationale |
|---|---|
| **Never trust client-supplied role** | JWT role claims (if present) are treated as informational only. The membership store is the sole source of truth for authorization. |
| **Trust identity only after full token validation** | `sub` is only accepted after signature verification, issuer/audience checks, and expiry validation. |
| **Validate and sanitize `X-Estate-ID`** | Header value is validated for presence before use. An empty or missing header is a `400`, not silently ignored. |
| **Evaluate `(user_id, estate_id)` pair — not user alone** | This prevents cross-tenant privilege escalation. Being admin in estate-A grants nothing in estate-B. |
| **Authorization lives only in middleware** | No handler contains role logic. This eliminates the risk of accidentally bypassing authorization by adding a new handler without properly composing the middleware. |
| **In-memory store uses `sync.RWMutex`** | Concurrent reads are lock-free; writes acquire an exclusive lock. This prevents data races in a multi-goroutine server. |

---

## 9. Testing Strategy

### Test Priority Order

1. **Permission Drift (highest priority)**
   - Validates the core challenge requirement
   - Seed admin → first request → downgrade role → second request must be 403

2. **Authorization Success**
   - Admin in active estate receives 200

3. **Authorization Failure**
   - Non-admin in active estate receives 403

4. **Tenant Boundary Correctness**
   - User is admin in estate-A but requests with X-Estate-ID: estate-B → 403

5. **Context Validation**
   - Missing `X-Estate-ID` header → 400
   - Empty `X-Estate-ID` header → 400

6. **Authentication Failures**
   - Missing `Authorization` header → 401
   - Expired JWT → 401

7. **Concurrency Safety**
   - Parallel goroutines calling `GetRole` and `SetRole` under race detector

### Test Matrix

| Test Case | Input | Expected |
|---|---|---|
| `AdminCanOpenGate` | Valid JWT (user-1) + X-Estate-ID: estate-1 (admin) | 200 |
| `NonAdminForbidden` | Valid JWT (user-1) + X-Estate-ID: estate-1 (resident) | 403 |
| `MissingEstateHeader` | Valid JWT + no X-Estate-ID | 400 |
| `WrongEstateForbidden` | Valid JWT (admin in estate-1) + X-Estate-ID: estate-2 | 403 |
| `PermissionDrift` | 200 first, downgrade, 403 second | 200 then 403 |
| `MissingToken` | No Authorization header | 401 |
| `ExpiredToken` | Expired JWT | 401 |
| `ConcurrentRoleAccess` | 100 goroutines R/W under `-race` | No data race |

---

## 10. Implementation Plan (File-by-File)

### Suggested Execution Order

```
1. estate_roles.go          ← data layer first
2. middleware.go + auth.go  ← identity & context injection
3. authz_estate.go          ← authorization policy
4. gate.go + routes.go      ← endpoint registration
5. gate_test.go             ← verification
6. README.md updates        ← documentation
```

### File Manifest

| File | Change Type | Purpose |
|---|---|---|
| `estate_roles.go` | New | In-memory role store with `GetRole` / `SetRole` + mutex |
| `middleware.go` | Modified | Add `withActiveEstate`, context keys, getter helpers |
| `auth.go` | Modified | Store `claims.Subject` as `user_id` in context |
| `authz_estate.go` | New | `requireEstateRole("admin")` middleware factory |
| `gate.go` | New | `openGate` handler — thin, no role checks |
| `routes.go` | Modified | Register `POST /v1/gate/open` with full middleware chain |
| `main.go` | Modified | Extend `application` struct with `estateRoles` dependency; seed data |
| `gate_test.go` | New | Acceptance tests including drift + auth failure + wrong-estate |
| `gate_e2e_test.go` | New | End-to-end permission drift test |

---

## 11. Limitations & Future Evolution

### Current Limitations (MVP Scope)

| Limitation | Impact | Acceptable for MVP? |
|---|---|---|
| **In-memory role store** | State lost on restart; no persistence | Yes — challenge explicitly allows this |
| **No role invalidation events** | Role changes are only observable via direct `SetRole` calls | Yes — drift is simulated not event-driven |
| **Single simulated user** | Only `user-1` and `estate-1` are seeded | Yes — challenge allows hardcoded identity |
| **No refresh token / token revocation** | A stolen JWT can be used until expiry | Acceptable for MVP; production needs a blocklist |
| **No audit logging** | Authorization decisions are not persisted | Acceptable for MVP |
| **No HTTPS enforcement** | TLS is not configured in local server | Expected for local challenge submission |

### Production Evolution Path

| Concern | MVP | Production |
|---|---|---|
| Role store | `map[userID]map[estateID]role` + `sync.RWMutex` | PostgreSQL / Redis with cache invalidation |
| Token revocation | Not implemented | JWT blocklist in Redis, or switch to opaque tokens |
| Role change propagation | Direct `SetRole` call | Event bus (Kafka/NATS) → invalidate cached roles |
| Audit trail | None | Append-only audit log per access decision |
| Multiple users/estates | Single seed | Dynamic enrollment via estate membership API |
| Token rotation | Not implemented | Short-lived access tokens (15min) + refresh tokens |

The key design strength: **middleware/service contracts do not change** when the store is replaced. `requireEstateRole` calls `store.GetRole(userID, estateID)` — swapping from in-memory to Redis-backed implementation requires only a new struct implementing the same interface.

---

## 12. Theory & Explanation (README Answers)

### Q1. What is the minimum data required to authorize a request in this system?

Three pieces of data are required:

1. **Verified user identity** (`user_id`) — extracted from a validated JWT `sub` claim
2. **Active estate context** (`estate_id`) — from the `X-Estate-ID` request header
3. **Current server-side role** — fetched from the membership store using `(user_id, estate_id)` as a composite key

Nothing else. Specifically: the request body, any role field in the JWT, and any other header are irrelevant to the authorization decision.

### Q2. Where should authorization logic live in a Go API, and why?

Authorization logic belongs in **middleware**, not in handlers.

**Why:** Handlers should only be responsible for request parsing and response formatting. When authorization lives in middleware, it is enforced before the handler executes, it is reusable across multiple routes, it cannot be accidentally bypassed by a developer who adds a new handler and forgets to add a check, and it is independently testable. Centralizing the decision point also makes auditing and reasoning about security posture straightforward.

### Q3. What is the most dangerous RBAC bug you can imagine in a system like this?

**Trusting a stale or client-supplied role.** Specifically: reading the role from a JWT claim (or any client-controlled field) rather than from the current server-side state.

This would mean a user whose role was revoked could continue accessing protected resources until their token expires — potentially hours or days. In an access-control system like Jarakey, this is catastrophic: a fired employee or a compromised account could retain admin access to physical gates long after revocation.

A close second is a **missing tenant boundary check** — evaluating `GetRole(userID)` without scoping to `estateID`, which would cause admin privileges in one estate to grant access to all others.

### Q4. What should the backend never trust from the client?

- **Role or permission data** — never read role from a JWT claim, request header, or body field
- **User identity** — never accept a user ID from the request body; extract it only from a verified token
- **Active estate validity** — the estate ID provided must be treated as an untrusted input; the server determines what roles the user holds for it
- **Token claims beyond `sub` and standard claims** — custom claims should not be used to make authorization decisions

The general rule: the client is untrusted. Every security-relevant decision must derive from server-side state, with client input used only after validation and sanitization.

### Q5. How did AI help you complete this challenge?

AI assisted with:
- Drafting the technical proposal structure and README template
- Generating boilerplate for middleware signatures, context key declarations, and test scaffolding
- Producing the implementation checklist (file-by-file mapping) from the proposal
- Reviewing the security boundary decisions for gaps

AI was not used to design the authorization architecture or decide where trust boundaries should be drawn — those decisions require engineering judgment applied to the specific problem.

### Q6. Where did you intentionally apply guardrails when using AI?

- **Authorization design**: Verified that the AI-generated middleware never reads the role from the JWT. Any suggestion to use JWT role claims for authorization was explicitly rejected.
- **Trust boundaries**: Manually reviewed every context key propagation to ensure `user_id` comes only from the validated token `sub`.
- **Drift simulation**: Checked that the proposed test structure actually proves freshness — not just that the first call returns 200.
- **Handler purity**: Reviewed the generated `openGate` handler to confirm zero role checks existed inside it.

The guardrail principle: AI accelerates implementation; the engineer is responsible for correctness of security-sensitive decisions.

### Q7. What is the most important test you would write first, and why?

**The permission drift test.**

```go
func TestPermissionDrift(t *testing.T) {
    // First request: user is admin -> expect 200
    // Downgrade role in store
    // Second request: same token, same estate -> expect 403
}
```

This test is most important because it directly validates the system's core guarantee: that authorization is evaluated from current server-side state, not stale data. If this test passes, it proves that the implementation does not cache roles in the JWT, does not cache the authorization decision, and the membership store is the actual source of truth. All other tests verify individual layers; this test verifies the architecture end-to-end.

---

## 13. Postman Collection

Import this file into Postman:
- `postman_collection_vein_jarakey_rbac_mvp.json`
- `postman_environment_vein_jarakey_rbac_mvp.json`

Included requests:
- `GET /healthcheck`
- `POST /v1/auth/token` (captures token into collection variable)
- `POST /v1/gate/open` (expected `200` before downgrade)
- `POST /v1/challenge/downgrade` (simulates in-memory role downgrade)
- `POST /v1/gate/open` (expected `403` after downgrade)
- `POST /v1/gate/open` without estate header (expected `400`)

Quick copy/paste variable values:
```env
baseUrl=http://localhost:4000
email=admin@vein.dev
password=VeinPass#2026!
estateId=estate-1
token=
```

---

## 14. Verification Results

Commands executed:
- `go test ./...`
- `go test -race ./cmd/api`

Result:
- Both passed successfully.

---

## 15. How to Run

1. From the project folder:
```bash
cd /Users/macbookpro/Documents/JarakeyTHA/vein-jarakey-rbac-mvp
```

2. Create your env file:
```bash
cp .env.example .env
```

3. Start the API:
```bash
make run
```

4. For this challenge, DB setup is optional:
- leave `DB_DSN` empty in `.env` for DB-free mode
- use `POST /v1/auth/token` with:
  - `email`: `admin@vein.dev`
  - `password`: `VeinPass#2026!`

5. Then run the Postman collection flow (token -> gate open 200 -> downgrade -> gate open 403).

---

*Document version: 1.1 | Challenge: Jarakey Golang Backend Engineer Take-Home MVP*
