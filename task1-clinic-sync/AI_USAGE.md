# AI Usage

## 1. Tools Used

| Tool | Model/Version | Used for |
|------|---------------|----------|
| OpenCode | GLM-5.3 Flash | Main code agent |
| (other tool, if any) | | |

## 2. How I Used AI

Be specific: which task, in which part of the project.

- [x] Created `AGENTS.md` (2026-11-10): investigated README/SETUP.md/REVIEW.md, Makefile, `config.py`, `database.py`, mock API, and test suite; verified each claim by running `.venv\Scripts\python.exe -m pytest tests/ -v` and `make test` (11 passed)
- [x] Added the "AI Usage Log" policy to `AGENTS.md` and created this `AI_USAGE.md` from the provided template (2026-11-10)
- [x] Fixed REVIEW.md issue #1 (2026-11-10): sync loop in `app.py:sync_pending_claims` no longer marks failed submissions as synced; saves `external_claim_id` on success; added `tests/test_sync.py` (3 tests) that fail on the buggy baseline and pass after the fix (14/15 tests passing, 11 existing stay green)
- [x] Fixed REVIEW.md issue #2 (2026-11-10): added `run_periodic_sync` background job wired into the app lifespan using `SYNC_INTERVAL_SECONDS` (claims auto-submit at startup and on each interval); added 2 tests (`test_background_sync_submits_pending_claims_automatically`, `test_startup_syncs_pending_backlog`) that fail on the baseline without the job; reset `app._token` in the mock shim helper to fix cross-test token cache pollution (16 tests passing)
- [x] Fixed REVIEW.md issue #3 (2026-11-10): token cache in `app.py` now honors `expires_in` (with 5s skew margin), refreshes proactively when expired, invalidates and refetches on 401 during claim submission, and serializes concurrent refreshes with `threading.Lock`; added `tests/test_token.py` (3 tests: 401-resubmission, proactive expiry refresh, single-fetch-under-concurrency) that fail on the baseline; fixed app.py state by restoring from temp backup after discovering a user commit of Issue #1 changed the meaning of `git checkout HEAD` (19 tests passing)

## 3. Changed / Rejected from AI Output

One entry per occurrence. Use this format:

No entries yet (as of 2026-11-10).

### (short title)
- **AI suggestion:** what was proposed or generated
- **Decision:** accepted with changes / rejected
- **What I did:** what I changed, or what I replaced it with
- **Reason:** why (performance, security, complexity, didn't fit the requirements, bug, etc.)

## 4. Additional Notes (optional)

- Parts I wrote entirely on my own without AI:
- Things I verified manually (tests, code review, etc.):
