# Task 2: Offline-First Sync

## Scenario

Health workers at small Puskesmas and Posyandu use a Flutter app on low-end Android phones (2 GB RAM, intermittent network) to register patients, record visits, and dispense medicine. Devices may be offline for up to 3 days, then sync to a central server.

## Your Task

### Part 1: Design Document

Write `DESIGN.md` (max 3 pages) covering:

1. **Exactly-once delivery.** How does a visit get from phone to server exactly once?
2. **Duplicate patients.** Two offline devices register the same patient: one writes "Siti Aminah", the other "St. Aminah", and only one has a NIK. What happens at sync? What is automatic, and what needs a human?
3. **Stock conflicts.** Two offline devices dispense the last units of the same medicine. What is your policy? Note: in a clinic, you cannot refuse a patient medicine because of a sync conflict.
4. **Wrong device clock.** A phone's clock is set to 2019. How does your design handle it?
5. **Lost device.** A phone is lost with unsynced patient data. What leaks, and what can be recovered?
6. **Schema migration.** An app update changes the local schema while phones still hold unsynced records.
7. **MVP scope.** What would you **not** build for a 2-month MVP?

### Part 2: Server-Side Sync Endpoint

Implement the server-side sync endpoint in **any backend language**, following [API_CONTRACT.md](API_CONTRACT.md).

Write tests for:
- A duplicate batch (same `batch_id` sent twice)
- A partially invalid batch (some valid, some rejected records)
- A possible duplicate patient
- Two devices dispensing the same last stock

### How to Run

In your submission, include a **"How to run"** section: one command (or a short list) that starts your endpoint on port 8000, and one that runs your tests. We will grade your endpoint by sending HTTP requests to it, so it must start cleanly from those instructions.

**Docker is optional.** If you already use it, a `Dockerfile` is welcome, but it isn't required and isn't scored. Clear run instructions are enough.

## What We're Looking For

- **Design:** Practical decisions, not theoretical completeness. "We would do X because Y" beats a list of options.
- **Implementation:** Working code that follows the contract. Per-record status handling, idempotency, and edge cases matter more than features.
- **Tests:** Meaningful assertions, not just "it returns 200."

## Sample Data

The `sample-payloads/` folder contains realistic JSON batches you can use to test your endpoint. See [SETUP.md](SETUP.md) for how to send them.

> **All data is synthetic.** Any resemblance to real people is coincidental.
