# Jarakey RBAC MVP - Technical Design & Implementation Update

> Golang Backend Engineer Take-Home Challenge  
> Framework: Vein (custom Go backend framework)

## 1. Status
This proposal describes the solution correctly and now reflects the implemented state in this repository.

Implemented project folder: `/Users/macbookpro/Documents/JarakeyTHA/vein-jarakey-rbac-mvp`

## 2. What Was Implemented (Actual)
- Estate-aware context middleware: `withActiveEstate` in `cmd/api/middleware.go`
- Auth middleware now injects `user_id` from JWT `sub`: `cmd/api/auth.go`
- In-memory role store with `sync.RWMutex`: `cmd/api/estate_roles.go`
- Estate authorization middleware: `requireEstateRole("admin")` in `cmd/api/authz_estate.go`
- Gate handler: `openGate` in `cmd/api/gate.go`
- Protected route added: `POST /v1/gate/open` in `cmd/api/routes.go`
- App dependency wiring for role store: `cmd/api/main.go`
- Unit tests: `cmd/api/gate_test.go`
- E2E drift test: `cmd/api/gate_e2e_test.go`
- Work summary: `WORKDONE.md`
- Postman collection: `postman_collection_vein_jarakey_rbac_mvp.json`

## 3. Corrections to Original Proposal

### 3.1 Endpoint path
- Challenge text mentions `/gate/open`.
- Implemented path is versioned in Vein style: `POST /v1/gate/open`.

### 3.2 400 error body wording
The proposal shows:
```json
{ "error": "missing or invalid X-Estate-ID header" }
```
Implementation currently returns:
```json
{ "error": "X-Estate-ID header is required" }
```

### 3.3 Store implementation wording
Proposal future table references `sync.Map` for MVP. Implementation uses:
- nested `map[string]map[string]string`
- guarded by `sync.RWMutex`

### 3.4 Test matrix vs current test set
Implemented and passing:
- admin/non-admin behavior (unit + e2e drift)
- missing estate header => 400
- wrong-estate => 403
- missing token => 401
- expired token => 401
- permission drift: first 200 then 403
- concurrent role store access test
- race run executed: `go test -race ./cmd/api` passed

## 4. API Contract (Implemented)
### Endpoint
`POST /v1/gate/open`

### Required Headers
- `Authorization: Bearer <jwt>`
- `X-Estate-ID: <estate-id>`

### Responses
- `200 OK` -> `{ "message": "gate opened" }`
- `400 Bad Request` -> missing estate header
- `401 Unauthorized` -> token missing/invalid
- `403 Forbidden` -> role missing or not `admin` for active estate

## 5. Drift Behavior (Implemented)
Seed:
- `user-1` has `admin` for `estate-1`

Flow:
1. Request with same token + `estate-1` -> `200`
2. In-memory role update to `resident`
3. Same token + same estate -> `403`

This is validated in `cmd/api/gate_e2e_test.go`.

## 6. Security Boundary (Implemented)
- Identity source: validated JWT `sub`
- Active context source: `X-Estate-ID` header, validated for non-empty
- Authorization source of truth: server-side role store lookup per request
- No role checks in handler (`openGate` is thin)

## 7. Submission Readiness
All previously identified gaps have been addressed:
1. Added `WrongEstateForbidden` test.
2. Added `MissingToken` and `ExpiredToken` tests for `/v1/gate/open`.
3. Ran `go test -race ./cmd/api` and documented the result in README.
4. Added the 7 theory Q&A section directly in README.

## 8. Delivered Artifacts
- Code implementation in `cmd/api/*`
- E2E and unit tests
- `WORKDONE.md`
- `postman_collection_vein_jarakey_rbac_mvp.json`
- This updated proposal document
