# Work Done: Vein RBAC-Aware Gate MVP

## Project Name
`vein-jarakey-rbac-mvp`

## Summary
This implementation adapts the Vein framework to solve a multi-tenant RBAC challenge where authorization depends on active estate context (`X-Estate-ID`) and must remain correct under permission drift.

## What Was Implemented

1. Added estate-aware request context handling:
- Middleware reads and validates `X-Estate-ID`.
- Estate is attached to request context.
- Missing header returns `400`.

2. Strengthened authentication context:
- `authenticate` now stores authenticated `user_id` (`JWT sub`) in context.
- Existing behavior for token validation remains unchanged.

3. Added in-memory estate role store:
- Thread-safe store using `sync.RWMutex`.
- Role keying is per `(user_id, estate_id)`.
- Seeded default membership: `user-1` is `admin` in `estate-1`.

4. Added authorization boundary outside handlers:
- New middleware `requireEstateRole("admin")`.
- Looks up current role from server-side store on every request.
- Returns `403` when role is missing or insufficient.

5. Added protected gate endpoint:
- `POST /v1/gate/open`.
- Route chain: `authenticate -> withActiveEstate -> requireEstateRole("admin") -> openGate`.
- Handler stays thin and contains no RBAC logic.

6. Added tests:
- Unit tests for missing estate header (`400`).
- Unit tests for non-admin access (`403`).
- E2E drift test showing first `200`, then role downgrade, then `403`.

## Why This Solves the Challenge
- Active tenant context is explicitly provided and validated per request.
- Authorization is server-side and context-aware.
- Permission drift is handled by fresh role lookup each request.
- Authorization logic is outside handlers as required.
- Implementation remains MVP-friendly and aligned with Vein architecture.

## Key Security Decisions
- Role claim in token is not trusted for estate-level authorization.
- Only authenticated identity (`sub`) is trusted from JWT.
- Effective role is read from server-owned membership state.

