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

## 3. Changed / Rejected from AI Output

One entry per occurrence. Use this format:

### AGENTS.md created at the wrong location
- **AI suggestion:** create `AGENTS.md` at the repo root, covering all three task folders
- **Decision:** rejected
- **What I did:** wrote `task2-offline-sync/AGENTS.md` scoped to this folder only and deleted the root-level copy
- **Reason:** keep the submission scoped to the task folder — user wants each task's work (including agent rules/deliverables) confined to its own folder, with parent-level and sibling-task files untouched

## 4. Additional Notes (optional)

- Parts I wrote entirely on my own without AI:
- Things I verified manually (tests, code review, etc.): cross-checked sample-payload ordering/`/reset` behavior in `send_samples.py`, MED-003 quantity discrepancy between `SETUP.md`/`send_samples.py` (20) and `API_CONTRACT.md` (2000), git branch layout, and Python environment before writing `AGENTS.md`
