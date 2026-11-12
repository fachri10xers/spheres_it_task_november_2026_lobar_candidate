# AGENTS.md

Hiring take-home, Task 1 (see README.md): a deliberately buggy FastAPI clinic-sync service. The committed `app.py` is the buggy baseline; `REVIEW.md` catalogs 15 issues. Use `git diff -- app.py tests/` to check whether the required fixes are already applied rather than assuming from `app.py` itself.

Deliverable rules (README): fixes must be the most impactful issues, and each fix needs a test that fails before the fix and passes after. The existing 11 tests must stay green.

## Commands

- Tests: `make test` (or `.venv\Scripts\python.exe -m pytest tests/ -v`)
- Single test: `.venv\Scripts\python.exe -m pytest tests/test_mock_api.py -v`
- Run both services: `make run` (mock claim API :8001 in background, app :8000 foreground)
- Reset DB: `make reset-db` (recreated from schema on next startup)

`make` works here — the Makefile has an `ifeq ($(OS),Windows_NT)` patch so it uses `.venv\Scripts\python` on Windows. No lint/typecheck config exists.

## Import-time env gotcha

`config.py` and `mock_claim_api.py` read env vars with `os.getenv` at import time, and nothing loads `.env` files. To change behavior (yours or the mock API's), set `os.environ` AND reassign `config.<VAR>` before importing `app` — see how `tests/conftest.py` does it for `DATABASE_PATH`.

Mock API knobs (`MOCK_*` in `.env.example`): `MOCK_TIMEOUT_RATE` defaults to 0.05 (5% of claim calls will randomly hang); tests disable it via conftest.

## Test notes

- Tests run in-process with FastAPI `TestClient` — no servers needed, suite runs in <1s.
- `mock_api_client` fixture clears module-level state in `mock_claim_api.py`; if you test claim submission end-to-end, reuse it instead of hitting a live server.
- `app.py` is imported once per pytest session: globals like `_token` persist across tests.
- No pytest config file; the `asyncio_default_fixture_loop_scope` warning from pytest-asyncio is expected noise.
- Untracked `data/clinic.db-shm` / `data/clinic.db-wal` in `git status` are SQLite WAL artifacts, not a problem.

## Service wiring

- Two separate FastAPI apps: `app.py` (service) and `mock_claim_api.py` (simulated insurer API). The app calls `POST {CLAIM_API_URL}/token` then `POST /claims` via httpx; `CLAIM_API_URL` defaults to `http://localhost:8001`.
- `POST /sync` uses FastAPI `BackgroundTasks` — there is no real background scheduler (REVIEW.md issue #2).
- SQLite at `DATABASE_PATH` with `PRAGMA foreign_keys=OFF`: the FK constraints in the schema are not enforced.
- Known clinic IDs `PKM-001`..`PKM-003` and medicines `MED-001`..`MED-008` are seeded by `INSERT OR IGNORE` inside `database.py:SCHEMA`; other IDs are rejected (e.g. clinic → 400).

## AI Usage Log

This project requires an `AI_USAGE.md` file (see the project's AI policy).

- Read `AI_USAGE.md` at the start of each session.
- Whenever the user rejects, modifies, or corrects something you generated, add an entry to Section 3 using the existing format.
- Whenever you complete a significant task (new feature, tests, debugging), add a line to Section 2.
- Only record what actually happened in this session. Do not invent entries. If the reason for a decision is unclear, write `TODO: ask user`.
- Never delete or rewrite existing entries. Only append.
- Ask the user to confirm the reason before writing it in the "Reason" field.
