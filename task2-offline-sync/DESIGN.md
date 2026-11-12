# DESIGN.md — Offline-First Sync (Task 2)

Design for a Flutter app on low-end Android phones (2 GB RAM, intermittent network, offline up to 3 days) at Puskesmas/Posyandu, syncing to a central server via the `/sync` contract in `API_CONTRACT.md`.

## 1. Exactly-once delivery

True exactly-once delivery over an intermittent network is impossible: the device can never distinguish "my request was lost" from "my response was lost." We don't try to solve the unsolvable — we make retries harmless.

**Policy: at-least-once delivery + idempotent processing = a visit lands exactly once.**

**Phone side**

- Every record gets a **client-generated UUIDv4 `client_uuid`** at creation time; the server never assigns record identity. Each sync attempt groups all unsynced records under one stable `batch_id`.
- Unsynced records live in a **local outbox table** with states `pending → in_flight → synced` and an attempt counter.
- When connectivity is available, the device sends **one batch containing all unsynced records**, keeping the same `batch_id` until every record in it is acknowledged.
- **Retry policy: finite.** Up to 5 attempts with exponential backoff per batch. After the 5th failure the batch is flagged "needs attention" and surfaced to the health worker rather than silently retrying forever.
- The device counts a record as delivered when the server replies `accepted` **or** `duplicate` — both mean exactly one copy exists server-side. Only `rejected` comes back to the user for correction.

**Server side**

- **Idempotency by `client_uuid`:** a UNIQUE constraint on `client_uuid`. A resent record hits the constraint and returns `duplicate`; no second row is ever created.
- **Idempotency by `batch_id`:** every processed batch's full response is persisted keyed by `batch_id`. A resent batch returns the stored response verbatim, so retries after a lost response replay the original result instead of recomputing it.
- Together, these guarantee that "send until ACK" cannot produce a double visit in the central database.

## 2. Duplicate patients

Scenario: two devices sync after 3 days offline — one registered "Siti Aminah" **with** a NIK, the other "St. Aminah" without one.

**Policy: sync never blocks a registration.** In a clinic, refusing to record a patient "because of a sync conflict" is worse than a duplicate: the system accepts the record and a human resolves the ambiguity from the review queue.

**Automatic (at sync)** — matching runs only for `patient_registration`:

- **Exact NIK:** if the incoming `nik` equals an existing patient's NIK, store the record and flag `flagged_possible_duplicate_patient` with reason "Exact NIK match" (highest confidence; no birth-date/name check needed). An **all-zero NIK** (`0000000000000000` — the sample data's placeholder for "no NIK") is treated as *no NIK* and never triggers an exact match, otherwise two NIK-less patients would false-positive against each other.
- **Fuzzy:** normalize both names (lowercase, strip punctuation and abbreviation dots) and treat edit distance ≤ 2 as similar — `"st aminah"` → `"siti aminah"` is exactly distance 2 — against any existing patient **with exactly equal `birth_date`, regardless of clinic**. The contract's rule 4 names similar name + overlapping birth date with no clinic limit, and the realistic scenario puts the two "Siti Aminah" variants in different clinics (PKM-001 vs PKM-002) — a same-clinic block would flag nothing. Cross-clinic false positives are acceptable here: the review queue exists to absorb them. Match → store + flag with reason "Similar name and birth date".
- No match → plain `accepted`. Flagged records are still stored under their own `server_id` and returned with `existing_patient_id`, per the contract.

**Needs a human (review queue):**

- Every flag becomes one review-queue item (`type: possible_duplicate_patient`). Staff confirms or dismisses.
- **On confirm (same person):** the *older* patient becomes canonical; missing fields are filled from the newer record — the NIK-bearing record wins, so no patient ends up NIK-less; visits/dispensings referencing the superseded patient are remapped to the canonical one; the loser is marked `merged`, never deleted (audit trail).
- **On dismiss:** both records remain independent patients.
- We deliberately do **not** auto-merge or auto-delete: a wrong automatic merge is far harder to undo than an extra patient sitting in a review queue.

## 3. Stock conflicts

Scenario: two devices are offline and each dispenses the last units of the same medicine.

**Policy: never refuse a dispensing.** The patient is already standing at the desk — the conflict is an accounting problem, not a medical one. (The contract mandates this; we agree, and the review queue is what absorbs the mess.)

**Ordering / conflict resolution**

- Dispensings are applied **in server-arrival order** (batches in the order received; within a batch, records in order). The first device gets the units; the second dispense drives `current_stock` negative. We don't invent a "winner" — the ledger shows exactly what happened, and `stock_updates` in each response reflects the running balance.
- Negative stock is flagged `negative_stock` in the review queue per the contract.

**Accounting — append-only stock ledger**

- `/reset` seeds are opening-balance rows; each dispensing inserts a negative ledger row tied to the dispensing's `client_uuid`; each reconciliation inserts an adjustment row.
- `current_stock = opening balance + Σ ledger`. Stock is **derived, never mutated in place** — "who dispensed what, receiving in what order" is always reconstructible, which is exactly what you want when a pharmacy audit shows -5 ORS.

**Phone side**

- The device shows last-known stock with a stale-value warning, cached from the previous sync. It **never blocks** dispensing — cached stock cannot stop a patient from being served.

**Reconciliation (human)**

- Pharmacy staff physically counts the shelf, records the physical count, and the server writes one adjustment row producing the delta automatically; the `negative_stock` flag is cleared. History stays intact — only the balance is corrected.

## 4. Wrong device clock

Scenario: a phone's clock is set to 2019.

**Server side — server time is the only authority for machine decisions**

- Ordering, conflict resolution, and every stored timestamp (`server_received_at`, `received_at`) use the **server clock** (contract rule 6). This makes a wrong device clock harmless *by construction*: a 2019 record simply arrives "now" and is processed in arrival order against current stock and current patient matching.
- `device_recorded_at` is stored as-is for human reference ("the nurse recorded this at X"), but it is **never validated, never used for ordering or dedup, and never causes rejection** — discarding good clinical data over a metadata problem is the worst outcome in a clinic.

**Device side — layered, non-blocking clock hygiene** (no NTP, nothing that blocks care):

- **On every app open:** red-flag check — if the system date is implausible (e.g., far in the past), show a persistent warning banner.
- **Daily confirm prompt:** once a day, ask "today is <date>?" as a 2-second confirmation to catch silent drift.
- **Header clock:** the current date is always visible in the app header, so a wrong date is noticed during normal use.
- **At sync:** compare the device clock against server time; drift > 24h → warn the health worker to fix the phone's clock (the usual real-world cure is a human re-setting it).

## 5. Lost device

Scenario: a phone is lost while it still holds unsynced patient data.

**What leaks**

- The phone holds the local SQLite DB. **With SQLCipher at rest** (key in Android Keystore, app PIN required), a lost phone is an anti-theft problem, not a data breach: the file is unreadable without the device's Keystore.
- Residual risk we accept and say plainly: an already-unlocked, powered-on phone leaks whatever is on screen or in memory. Screen lock + PIN keep this window short.

**What can be recovered**

- **Everything already synced:** full history lives on the server; a replacement device authenticates, syncs down, and resumes.
- **Records never synced:** unrecoverable — full stop. We state this honestly to stakeholders instead of pretending a backup exists.
- **Indirect recovery for dispensings:** their absence shows up as server stock being higher than the physical shelf → caught at reconciliation (§3). Clinical records (patients, visits) have no such safety net — every avoided risk there is worth more than any backup feature.

**Mitigations**

- SQLCipher-encrypted DB; Keystore-held key; app PIN.
- **Shrink the loss window:** auto-sync whenever connectivity is detected + an end-of-day "sync before closing" nudge. The everyday exposure is minutes of data, not 3 days.
- Remote wipe: **out of MVP scope** (needs a device registry + push infrastructure); recorded as future work, not silently ignored.

## 6. Schema migration

Scenario: an app update changes the local schema while phones still hold unsynced records.

**Core rule: two version numbers, kept separate.**

- **Local DB schema version** — private to the phone, migrates at startup, never appears in sync traffic.
- **Wire `schema_version`** — the server contract's version. After any local schema change, the device still sends the version **the server supports** (v1 today; the server rejects unknown versions with `"unsupported schema_version"`). New app-only fields stay local and simply don't ride the wire until the server contract gains them. This rule alone prevents the worst failure mode: an app update whose records bounce off the server forever.

**Migration mechanics**

- Migrations run **at app startup, before any write or sync**, forward-only and additive; unsynced outbox rows are migrated in place with their payloads intact — a health worker never has to know a migration happened.
- An app update may also land while a batch is `in_flight`. On next attempt that batch is reset to `pending` with a freshly minted `batch_id`; any overlap is self-healing because already-delivered records return as `duplicate` via `client_uuid` (§1). No special "resume protocol" is needed — dedup absorbs it.
- **No downgrade support:** forward-only. An older app opening a migrated DB refuses to run and asks to sync first, so data reaches the server before any rollback attempt.

**Why this is safe for unsynced data:** migrations transform rows, not truths — `client_uuid`s and payloads survive verbatim, so idempotency guarantees continue to hold across the upgrade.

## 7. MVP scope — what we would NOT build

Rule: during the 2-month MVP we build only what (a) serves the offline-first sync loop or (b) prevents unrecoverable clinical data loss. Both criteria come from "a patient in front of a nurse must never be blocked" (§3) and "data must never bounce" (§6).

**Cut — and the trigger that adds it back:**

- **Advanced/ML patient matching** — edit distance + NIK + birth-date blocking covers realistic Puskesmas volume; embeddings need labeled data we don't have. *Trigger: the review queue routinely carries misses between real duplicates.*
- **Appointment scheduling, cross-clinic patient roaming, multi-user roles** — the MVP workflow is exactly register → visit → dispense → sync. *Trigger: a real referral workflow requirement from the district.*
- **Full review-queue workflow app** — the queue ships as the contract's list + confirm/dismiss endpoints; no assignments, SLAs, or dashboards. *Trigger: more than one person processes queues daily.*
- **Observability stack / audit-log UI** — server logs plus the review queue are enough to operate one clinic cluster. *Trigger: scale beyond a rough few-dozen devices or a second site.*
- **Remote wipe, device registry, key rotation** — future work (§5); they need push infrastructure we're not building. *Trigger: the first lost device, or a compliance audit.*
- **Delta/partial sync, multi-server topology** — one-batch sync is trivially correct; partial sync adds resumption state to a system that dedup already makes self-healing. *Trigger: batches routinely exceed a few MB on the worst realistic link.*

**Kept — though tempting to cut:** idempotency tables (`client_uuid` + batch replay), the review queue and its flags, negative-stock handling with the ledger, server-clock ordering, encrypted local DB, forward-only migrations. These are not features; they are the floor that makes handing offline-first software to health workers safe.
