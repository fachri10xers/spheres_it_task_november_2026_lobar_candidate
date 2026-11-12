# AI Usage

Per-task detail lives inside each task folder; this file is the submission-level summary.

## 1. Tools Used

| Tool | Model/Version | Used for |
|------|---------------|----------|
| OpenCode | GLM-5.3 Flash | Main coding agent for both Task 1 and Task 2 |

No other AI tools were used.

## 2. How I Used AI (summary)

### Task 1 - Clinic Sync (detail: `task1-clinic-sync/AI_USAGE.md`)

- Investigated the deliberately buggy baseline (`app.py`, `database.py`, mock claim API, test suite) before touching code.
- Fixed the top 3 issues from `REVIEW.md`, each with tests that fail on the buggy baseline and pass after the fix: (1) the sync loop marking failed claim submissions as synced, (2) no background sync job running automatically, (3) the auth-token cache ignoring `expires_in` and not refreshing/re-fetching on 401.
- Kept all 11 pre-existing tests green; 19 tests passing at the end.

### Task 2 - Offline-First Sync (detail: `task2-offline-sync/AI_USAGE.md`)

- Drafted all 7 sections of `DESIGN.md` through per-question option menus, with my approval on each section before it was written.
- Implemented the server-side `/sync` endpoint in Go per `API_CONTRACT.md`: SQLite store + append-only stock ledger, verbatim batch replay by `batch_id`, per-record statuses, cross-clinic duplicate-patient matching, review queue, auxiliary endpoints.
- Added tooling (Makefile, multi-stage Dockerfile, docker-compose) and a 33-test Go suite: 15 HTTP API tests over the real mux (covering the four graded scenarios) plus 18 unit tests (11 store, 7 duplicate-patient matching); verified end-to-end with the grader's own `send_samples.py` on a booted binary.

Where I changed or rejected AI output (e.g. keeping the device retry policy finite instead of infinite backoff, re-opening duplicate-patient matching scope against contract rule 4), the reasoning is logged per task in each task file's Section 3.

## 3. Rejected / Changed AI Output

- `task1-clinic-sync/AI_USAGE.md` §3: no entries.
- `task2-offline-sync/AI_USAGE.md` §3: 5 entries.

## 4. Disclosure

This root file itself was drafted with AI assistance and reviewed by me.
