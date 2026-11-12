# AI Usage

## 1. Tools Used

| Tool | Model/Version | Used for |
|------|---------------|----------|
| OpenCode | GLM-5.3 Flash | Main code agent |
| (other tool, if any) | | |

## 2. How I Used AI

Be specific: which task, in which part of the project.

- [x] Created `AGENTS.md` for this folder (2026-11-11): investigated `README.md`, `SETUP.md`, `API_CONTRACT.md`, `send_samples.py`, and the sample payloads; verified each claim (payload script auto-calls `/reset`, MED-003 seeded at 20 units vs the contract's 2000 example, Python 3.12.10 with no global pytest, no existing venv/code here) before writing it down
- [x] Created this `AI_USAGE.md` from the task1 template (2026-11-11)
- [x] Recorded the user-chosen stack in `AGENTS.md` (2026-11-11): Go + SQLite + `.env` with the simplest structure; verified Go 1.27.3 is installed and gcc is absent (so only pure-Go SQLite drivers like `modernc.org/sqlite` will build)
- [x] Drafted all 7 sections of `DESIGN.md` (2026-11-11): presented option menus per design question (exactly-once, duplicate patients, stock conflicts, device clock, lost device, schema migration, MVP scope), applied user-selected choices via interactive Q&A, user approved each section before proceeding
- [x] Built the store layer `store.go` + `store_test.go` (2026-11-12, task T2, user-reviewed): SQLite schema (clinics, medicines, append-only stock_ledger with opening rows from `/reset`, client_uuid records registry, patients, visits, dispensings, batch_responses, review_queue), ledger-derived current stock, WAL + busy_timeout + foreign_keys pragmas, hand-rolled UUIDv4/RFC3339 helpers; 10 passing unit tests; user asked about WAL journal mode (already wired) and the store.go-vs-db.go / single-file-vs-split naming choices (kept store.go, single file)
- [x] Built the `/sync` pipeline `sync.go` (2026-11-12, task T3, user-reviewed): malformed-top-level 400 checks, verbatim batch replay by `batch_id`, sequential per-record processing in one transaction each (client_uuid dedup registry → schema_version → type → required fields → reference/medicine resolution → insert → ledger row → negative-stock review item), per-medicine collapsed `stock_updates`, response persisted and replayed byte-identical; validated with a throwaway httptest smoke over payloads 01/02/03/05/06/07 plus a fresh-batch resend (deleted after)
- [x] Implemented duplicate-patient matching `matching.go` + `matching_test.go` (2026-11-12, task T4, user-reviewed): name normalization (lowercase, punctuation→separator, whitespace collapse), Levenshtein ≤ 2 against equal-birth-date patients in any clinic, exact-NIK rule skipping all-zero NIKs, reason strings "Exact NIK match" / "Similar name and birth date"; updated `DESIGN.md` §2 to the final rule; caught and fixed two of my own test-defect guesses (a stray non-ASCII string, a miscomputed distance boundary) before review
- [x] Built auxiliary endpoints `handlers.go` + `main.go` wiring (2026-11-12, task T5, user-reviewed): GET `/patients/{id}` & `/stock/{id}` with contract-exact 404 detail strings, GET `/review-queue` (per-type item shapes, `[]` not null), POST `/reset` (mutex-guarded, counts response); `main.go` loads `.env` via godotenv (optional file) with `PORT` default 8000 and `DB_PATH` default `sync.db`, stdlib method-routed mux; verified with a throwaway mux smoke (incl. 405 on GET /sync) and a real `go build` binary boot on :8000 before cleanup

## 3. Changed / Rejected from AI Output

One entry per occurrence. Use this format:

### AGENTS.md created at the wrong location
- **AI suggestion:** create `AGENTS.md` at the repo root, covering all three task folders
- **Decision:** rejected
- **What I did:** wrote `task2-offline-sync/AGENTS.md` scoped to this folder only and deleted the root-level copy
- **Reason:** keep the submission scoped to the task folder — user wants each task's work (including agent rules/deliverables) confined to its own folder, with parent-level and sibling-task files untouched

### DESIGN.md §1: sync retry policy
- **AI suggestion:** infinite retries with exponential backoff + jitter, auto-run when connectivity returns
- **Decision:** modified
- **What I did:** wrote the retry policy as finite — 5 attempts with backoff, then the batch is flagged "needs attention" and surfaced to the health worker (DESIGN.md §1)
- **Reason:** user preferred this design over AI recommendation

### DESIGN.md §4: device-side clock hygiene
- **AI suggestion:** single mechanism — compare device clock vs server time at sync and prompt if drift > 24h
- **Decision:** modified
- **What I did:** wrote the user's layered, non-blocking design instead: red-flag check on every app open, daily date-confirm prompt, date always visible in the header, drift check vs server at sync (> 24h), no NTP (DESIGN.md §4)
- **Reason:** user preferred this design over AI recommendation

### DESIGN.md: stray conversation text appended after §7 (found in review)
- **AI suggestion:** none — AI defect: lines 137–161 contained leftover working-session text (a "Next up: Q3" note, the Q3 option menu, and a dangling `question` line) appended below the final section
- **Decision:** corrected
- **What I did:** user's review flagged it as CRITICAL; truncated the file after §7's closing paragraph ("…handing offline-first software to health workers safe."), restoring a clean 135-line document
- **Reason:** violate the max-3-pages deliverable requirement and read as corrupted output — user ordered the removal

### DESIGN.md §2: duplicate-patient matching scope
- **AI suggestion:** keep fuzzy matching blocked on same `clinic_id` + equal `birth_date` (as originally drafted and user-approved in DESIGN.md)
- **Decision:** modified
- **What I did:** during Part 2 implementation analysis, found the sample payload 04 (the graded duplicate-patient scenario) puts the near-duplicates in different clinics (PKM-001 vs PKM-002), so same-clinic blocking would flag nothing; offered three options, user chose "drop clinic blocking" + "align DESIGN.md". Final rule: edit distance ≤ 2 + equal birth_date across clinics (contract rule 4); exact-NIK rule excluding all-zero NIKs (payload shares `0000000000000000` as the "no NIK" placeholder); `DESIGN.md` §2 clause rewritten to match
- **Reason:** contract rule 4 has no clinic restriction and the graded payload's scenario is cross-clinic; cross-clinic false positives are absorbed by the intended review queue

### AI_USAGE logging policy note (2026-11-12)
- **AI suggestion:** batch all AI_USAGE.md Section 2 entries into one final entry at T8
- **Decision:** modified
- **What I did:** wrote one Section 2 line per completed user-reviewed task (T2/T3/T4) as the tasks close, keeping the log contemporaneous
- **Reason:** AGENTS.md asks for entries whenever a significant task completes; per-task lines reflect what actually happened at the time

## 4. Additional Notes (optional)

- Parts I wrote entirely on my own without AI:
- Things I verified manually (tests, code review, etc.): cross-checked sample-payload ordering/`/reset` behavior in `send_samples.py`, MED-003 quantity discrepancy between `SETUP.md`/`send_samples.py` (20) and `API_CONTRACT.md` (2000), git branch layout, and Python environment before writing `AGENTS.md`
