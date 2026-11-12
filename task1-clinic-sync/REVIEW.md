# REVIEW.md — Clinic Sync Service

## Issues by severity

| # | Severity | Issue | Location |
|---|---|---|---|
| 1 | Critical | Claims are marked `synced` even when submission fails. `external_claim_id` is also not saved. | `app.py:158–162, 178–188` |
| 2 | Critical | No background sync job. `SYNC_INTERVAL_SECONDS` is configured but never used. | `app.py:194–198, 299–302`; `config.py:6` |
| 3 | Critical | API token is cached forever. Expired tokens are never refreshed, and concurrent refreshes are not protected. | `app.py:20, 25–33, 143–160` |
| 4 | High | No batch idempotency. Re-sending the same `batch_id` creates duplicate records and stock deductions. | `app.py:209–296` |
| 5 | High | Retry logic is incorrect: retries permanent 4xx errors, ignores `Retry-After`, and can create duplicate claims after a timeout. | `app.py:143–160`; `mock_claim_api.py:62–75` |
| 6 | High | Sync holds a database write transaction while making HTTP requests, so concurrent ingestion can fail with `database is locked`. | `app.py:167–189`; `database.py:107–115` |
| 7 | High | Stock can become negative. Quantity is not properly validated, and concurrent dispensing can overwrite stock updates. | `app.py:89–103`; `models.py:28` |
| 8 | Medium | Timestamps are timezone-naive. Clinic timezone settings are not used. | `database.py:84–87`; `app.py:47, 66, 106, 124` |
| 9 | Medium | Patient data (NIK, name, DOB, diagnosis) is written to application logs. | `app.py:54–57, 80–83, 112–116` |
| 10 | Medium | Money is stored as `REAL` and accepts invalid values such as negative amounts and NaN. | `models.py:34`; `database.py:69, 76` |
| 11 | Low | List endpoints do not support pagination. | `app.py:305–333` |
| 12 | Low | Missing indexes for common lookups such as NIK, sync status, and clinic ID. | `database.py` |
| 13 | Low | `batch_id` and `timestamp` are accepted but not stored, so there is no ingestion audit trail. | `app.py:209–296`; `models.py:41` |
| 14 | Low | `LOG_LEVEL` is configured but never applied. | `config.py:7`; `app.py:14–16` |
| 15 | Low | Dispensing only catches `ValueError`. Other errors can fail the entire batch. | `app.py:262–271` |

## Top 5 issues and impact

### 1. Claims are marked as synced even when submission fails — Critical

`submit_claim_to_api` returns `None` after failed attempts, but the sync loop still marks the claim as `synced`. It also logs the claim as submitted even when the API did not accept it. Successful responses do not save `external_claim_id`.

**Impact:**
- **Finance:** Local records show claims as submitted, but the insurer has never received them. Reconciliation requires manual investigation and resubmission.
- **Dinkes:** Claim reports do not match the insurer's records, creating audit issues.
- **Clinic staff:** They cannot easily investigate unpaid claims because the system reports success.
- **Patients:** Claim or reimbursement issues may only be discovered weeks later.

**Why #1:** This silently loses claims and prevents retries. It can also turn other failures, such as token expiry and API timeouts, into permanent claim loss.

### 2. Background sync job is missing — Critical

The README says pending claims are submitted by a background job. However, the app only initializes the database during startup. Claims are submitted only when someone manually calls `POST /sync`. `SYNC_INTERVAL_SECONDS` is never used.

**Impact:**
- **Finance:** Claims remain pending and reimbursement is delayed.
- **Dinkes:** Insurer-side claim volume does not match clinic activity.
- **Clinic staff:** Pending claims keep accumulating, with no automatic process to clear them.
- **Patients:** No immediate impact, but claims remain unsubmitted.

**Why #2:** The main sync process does not run automatically. Unlike #1, pending claims are still visible and can be recovered once the issue is noticed.

### 3. API token is never refreshed — Critical

The token is stored in a global variable and reused forever. The code ignores `expires_in` and does not refresh the token after a 401 response.

**Impact:**
After the configured token expiry, all claim submissions can start failing. Combined with issue #1, those claims are then incorrectly marked as synced and permanently lost.

**Why #3:** It can stop all submissions after token expiry. Once issue #1 is fixed, expired tokens will cause a visible backlog instead of silently losing claims.

### 4. No batch idempotency — High

The service does not check or store `batch_id`. If a Puskesmas retries a batch, the service inserts the visits, bills, claims, and stock deductions again. The client also does not send the `claim_reference` supported by the mock API.

**Impact:**
- **Patients:** Duplicate visit records.
- **Clinic/pharmacy staff:** Stock records no longer match actual medicine usage.
- **Finance:** Duplicate claims may be rejected, flagged, or paid and later clawed back.
- **Dinkes:** Visit and utilization reports become inaccurate.

**Why #4:** Retrying requests is normal, especially when clinic connections are unstable. The resulting duplicates affect both inventory and finance, but they are more likely to be detected than claims silently marked as synced.

### 5. Retry logic can create duplicate claims — High

The client retries every error three times, uses a fixed one-second delay, and ignores `Retry-After`. It also uses a 10-second timeout while the mock API can take 15 seconds to process a request. The API may therefore finish and record a claim after the client has timed out and retried.

**Impact:**
- **Finance:** The insurer may receive multiple claims for one local claim, with external IDs the service never saved.
- **Clinic staff:** Sync becomes slower during API problems.
- **Dinkes:** Duplicate submissions may look like double billing.
- **Patients:** No direct impact.

**Why #5:** It can create duplicate external claims, but the duplicates are more likely to appear during reconciliation. Issue #1 is more serious because it hides missing claims by marking them as successful.

## Other issues

- **#6 — SQLite concurrency:** Sync holds a write transaction while waiting for the external API. Ingestion can fail with `database is locked`.
- **#7 — Stock handling:** Quantity is not properly validated, stock can become negative, and concurrent updates can overwrite each other.
- **#8 — Timezones:** Server timestamps and clinic-local dates are mixed. This can cause reporting and reconciliation issues across WIB/WITA.
- **#9 — Patient data in logs:** NIK, name, DOB, and diagnosis are exposed in application logs. This creates a privacy and compliance risk.
- **#10 — Other medium issues:** Money uses floating-point values.
- **#11–#15 — Low issues:** Missing pagination and indexes, no ingestion audit trail, unused log configuration, and inconsistent exception handling.

## Deliberately not fixed

These are deferred because they need a product/API decision, data migration, or a separate implementation scope.

1. **Batch idempotency (`batch_id`):** Needs agreed replay behavior with the 40 clinic integrations. For example, should a repeated batch return the original result or return a conflict?
2. **Claim idempotency key:** The real claim API's deduplication behavior must be confirmed first. The mock API stores `claim_reference` but does not actually deduplicate claims.
3. **Timezone handling:** The team needs to agree on how claim dates should work across WIB/WITA, then migrate existing timestamps safely.
4. **PHI in logs:** This needs a proper logging policy covering masking, retention, and access control.
5. **Negative-stock policy:** The technical race condition can be fixed separately, but the business rule for insufficient stock must be agreed first.
6. **Money as integer minor units:** Requires a schema and payload migration coordinated with integrations.
7. **Full concurrency hardening:** Row claiming and sync overlap protection need a separate implementation and load test.
8. **Authentication, authorization, pagination, and other deployment hardening:** These should be handled in a separate hardening task.
