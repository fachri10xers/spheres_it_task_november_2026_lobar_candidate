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

## 4. Additional Notes (optional)

- Parts I wrote entirely on my own without AI:
- Things I verified manually (tests, code review, etc.): cross-checked sample-payload ordering/`/reset` behavior in `send_samples.py`, MED-003 quantity discrepancy between `SETUP.md`/`send_samples.py` (20) and `API_CONTRACT.md` (2000), git branch layout, and Python environment before writing `AGENTS.md`
