# AGENTS.md

Task 2 (Offline-First Sync) of the SPHERES take-home — work happens **inside this folder only**. Do not modify anything in `task1-clinic-sync/`, `task3-dinkes-report/`, or any parent-level file unless the user explicitly asks. The parent `README.md` defines submission-level files (`CHOICE.md`, `SKIPPED_TASK.md`) that live outside this folder; if creating them is needed, ask the user first. `task1-clinic-sync/AI_USAGE.md` and `task1-clinic-sync/AGENTS.md` may be read as references, never edited.

## Deliverables (README.md in this folder)

- `DESIGN.md` (≤3 pages) covering the 7 design questions.
- Server-side sync endpoint implementing `API_CONTRACT.md` **exactly** — the grading harness sends HTTP requests, so response field names and per-record statuses must match the contract, not be "improved".
- Tests for the four listed scenarios: duplicate batch, partially invalid batch, duplicate patient, last-stock conflict.
- A "How to run" section: one command to start the server on port 8000, one to run the tests.

## Gotchas (verified against the docs)

- `send_samples.py` calls `/reset` itself before sending payloads in `sample-payloads/` (zero-padded filenames, sent in order 01→07). Testing payloads manually with curl → call `POST /reset` first (body in `SETUP.md`).
- `SETUP.md`/`send_samples.py` seed MED-003 with **20 units** (intentional for the stock-conflict scenario); the `/reset` example in `API_CONTRACT.md` says 2000 — the 20-unit value is the real test scenario.
- Contract traps: `/sync` returns 200 even with rejected records (400 only for malformed top-level structure); a duplicate `batch_id` must replay the original results; dispensing is accepted even when stock goes negative (flagged for reconciliation); `server_received_at` uses the server clock, not `device_recorded_at`.

## Stack (decided by the user — do not switch)

- **Go + SQLite + `.env`, simplest structure possible.** Prefer Go stdlib (`net/http`) over web frameworks and keep the file tree minimal.
- Go 1.27.3 is installed, but **there is no gcc on this machine** — CGO-based SQLite drivers like `mattn/go-sqlite3` will not build; use a pure-Go driver such as `modernc.org/sqlite`.
- Go does not load `.env` automatically — load it (e.g. `github.com/joho/godotenv` or a small stdlib parser) before any package reads config at init time.

## Environment (Windows, PowerShell 7)

- Task 2 is Go — run with `go run .` and test with `go test ./...`; no Makefile, no CI, no lint config yet.
- Nothing is implemented yet — code and tests are greenfield (Go's `testing` package, no external test framework needed).
- `.env` and `*.db` are covered by the parent `.gitignore` — local SQLite DB files showing in the tree are untracked artifacts, safe to leave alone.

## AI Usage Log

This project requires an `AI_USAGE.md` file (see the parent README AI policy). It lives in this folder (`task2-offline-sync/AI_USAGE.md`); use `task1-clinic-sync/AI_USAGE.md` as a read-only format reference and create it from that template when starting work here.

- Read `AI_USAGE.md` at the start of each session.
- Whenever the user rejects, modifies, or corrects something you generated, add an entry to Section 3 using the existing format.
- Whenever you complete a significant task (new feature, tests, debugging), add a line to Section 2.
- Only record what actually happened in this session. Do not invent entries. If the reason for a decision is unclear, write `TODO: ask user`.
- Never delete or rewrite existing entries. Only append.
- Ask the user to confirm the reason before writing it in the "Reason" field.
