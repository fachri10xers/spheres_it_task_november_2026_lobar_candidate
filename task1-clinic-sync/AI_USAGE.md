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
