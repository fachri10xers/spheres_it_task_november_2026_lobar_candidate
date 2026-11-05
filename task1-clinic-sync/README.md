# Task 1: Clinic Sync Service

> **Setup first?** See [SETUP.md](SETUP.md) for how to run the service and tests.

## Scenario

This FastAPI service receives batches of clinic data from Puskesmas (outpatient visits, bills, and pharmacy dispensing records), saves them to a local database, and submits insurance claims to an external claim API.

A background sync job pushes any pending claims that haven't been submitted yet.

The service is scheduled to go live at **40 clinics next week**.

## Your Task

1. **Review the code.** Write `REVIEW.md`: list the issues you find, ranked by severity. For the **top 5**, explain the real-world impact — what goes wrong, and who notices (patient, clinic staff, finance team, Dinkes).

2. **Fix the top 3 issues** in code. Each fix must include a test that **fails before the fix and passes after**.

3. **List what you deliberately did not fix**, and why.

## What We're Looking For

- **Prioritization.** Not all issues are equal. Explain why your #1 matters more than your #5.
- **Real-world reasoning.** "This could cause data loss" is weaker than "Finance reconciliation will show phantom claims that were never approved, requiring manual correction across 40 clinics."
- **Targeted fixes.** Fix the most impactful issues, not the easiest ones.

## Project Layout

```
app.py              Main FastAPI service (~300 lines)
mock_claim_api.py   Simulated external claim API
models.py           Pydantic request/response models
database.py         SQLite schema and connection management
config.py           Environment-based configuration
tests/              Existing test suite (all passing)
data/               SQLite database (created on startup)
```

## Notes

- The mock claim API is a separate service that simulates the real insurance claim API. It has configurable behavior (timeouts, rate limits, token expiry) — see `.env.example`.
- All existing tests pass. Your fixes should not break them.
- The service handles data from 3 Puskesmas across 2 timezones (WITA and WIB).
