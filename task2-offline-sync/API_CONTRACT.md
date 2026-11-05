# Offline Sync API Contract

This document specifies the HTTP API your server must implement. The grading harness sends requests to your server and checks responses against this contract.

## `POST /sync`

Accept a batch of records from an offline device.

### Request Body

```json
{
  "device_id": "device-abc-123",
  "batch_id": "batch-uuid-here",
  "records": [
    {
      "type": "patient_registration",
      "client_uuid": "uuid-generated-on-device",
      "device_recorded_at": "2026-10-15T09:30:00",
      "schema_version": 1,
      "data": {
        "nik": "5201141506900001",
        "name": "Siti Aminah",
        "birth_date": "1990-06-15",
        "gender": "F",
        "address": "Jl. Raya Sandik No. 42",
        "clinic_id": "PKM-001"
      }
    },
    {
      "type": "visit",
      "client_uuid": "uuid-generated-on-device",
      "device_recorded_at": "2026-10-15T09:35:00",
      "schema_version": 1,
      "data": {
        "patient_client_uuid": "uuid-of-patient-from-this-or-prior-batch",
        "clinic_id": "PKM-001",
        "polyclinic": "umum",
        "visit_date": "2026-10-15T09:30:00",
        "diagnosis": "ISPA",
        "notes": "Follow up in 3 days"
      }
    },
    {
      "type": "dispensing",
      "client_uuid": "uuid-generated-on-device",
      "device_recorded_at": "2026-10-15T09:40:00",
      "schema_version": 1,
      "data": {
        "visit_client_uuid": "uuid-of-visit-from-this-or-prior-batch",
        "medicine_id": "MED-001",
        "quantity": 10
      }
    }
  ]
}
```

### Record Types

| Type | Required `data` fields |
|------|----------------------|
| `patient_registration` | `nik`, `name`, `birth_date`, `gender`, `clinic_id` |
| `visit` | `patient_client_uuid`, `clinic_id`, `polyclinic`, `visit_date` |
| `dispensing` | `visit_client_uuid`, `medicine_id`, `quantity` |

### Response (HTTP 200)

Always return 200 for a valid batch structure, even if some records are rejected. Return 400 only if the top-level structure is malformed (missing `device_id`, `batch_id`, or `records`).

```json
{
  "batch_id": "batch-uuid-here",
  "server_received_at": "2026-10-15T16:30:00Z",
  "results": [
    {
      "client_uuid": "uuid-of-record",
      "status": "accepted",
      "server_id": "server-assigned-uuid"
    },
    {
      "client_uuid": "uuid-of-duplicate-record",
      "status": "duplicate"
    },
    {
      "client_uuid": "uuid-of-bad-record",
      "status": "rejected",
      "reason": "missing required field: name"
    },
    {
      "client_uuid": "uuid-of-fuzzy-match",
      "status": "flagged_possible_duplicate_patient",
      "server_id": "server-assigned-uuid",
      "existing_patient_id": "uuid-of-existing-patient"
    }
  ],
  "stock_updates": [
    {
      "medicine_id": "MED-001",
      "current_stock": 4990
    }
  ]
}
```

### Per-Record Status Values

| Status | Meaning |
|--------|---------|
| `accepted` | Record stored successfully |
| `duplicate` | A record with this `client_uuid` was already received |
| `rejected` | Record is invalid (missing fields, unknown type, etc.) |
| `flagged_possible_duplicate_patient` | Patient registered but flagged for human review (possible duplicate of existing patient) |

### Behavior Rules

1. **Idempotency by `client_uuid`:** If a record with the same `client_uuid` has been received before, return `duplicate`. Do not create a second record.
2. **Batch idempotency by `batch_id`:** If the entire batch (same `batch_id`) is sent again, return the same results as the first time.
3. **Partial success:** A batch with some invalid records must still process the valid ones. Return 200 with per-record results.
4. **Possible duplicate patients:** When a new `patient_registration` has similar name and overlapping birth date with an existing patient, accept it but flag it as `flagged_possible_duplicate_patient` and add it to the review queue.
5. **Stock policy:** Dispensing is always accepted, even if it makes stock go negative. Include updated stock in `stock_updates`. Negative stock should be flagged for reconciliation (visible in the review queue).
6. **Server timestamp:** Use `server_received_at` (server clock) for ordering, not `device_recorded_at` (which may be wrong).
7. **Schema version:** Accept records with `schema_version` 1. Records with unknown schema versions should be rejected with reason `"unsupported schema_version"`.

---

## `GET /patients/{patient_id}`

Return a patient by server-assigned ID.

### Response (HTTP 200)

```json
{
  "id": "server-uuid",
  "client_uuid": "device-uuid",
  "nik": "5201141506900001",
  "name": "Siti Aminah",
  "birth_date": "1990-06-15",
  "gender": "F",
  "address": "Jl. Raya Sandik No. 42",
  "clinic_id": "PKM-001",
  "received_at": "2026-10-15T16:30:00Z"
}
```

### Response (HTTP 404)

```json
{"detail": "Patient not found"}
```

---

## `GET /stock/{medicine_id}`

Return current stock for a medicine.

### Response (HTTP 200)

```json
{
  "medicine_id": "MED-001",
  "current_stock": 4990
}
```

### Response (HTTP 404)

```json
{"detail": "Medicine not found"}
```

---

## `GET /review-queue`

Return items flagged for human review.

### Response (HTTP 200)

```json
{
  "items": [
    {
      "type": "possible_duplicate_patient",
      "new_patient_id": "server-uuid-new",
      "existing_patient_id": "server-uuid-existing",
      "reason": "Similar name and birth date",
      "created_at": "2026-10-15T16:30:00Z"
    },
    {
      "type": "negative_stock",
      "medicine_id": "MED-003",
      "current_stock": -5,
      "created_at": "2026-10-15T16:31:00Z"
    }
  ]
}
```

---

## `POST /reset`

Reset the server to a known state for testing. This seeds stock and clinics.

### Request Body

```json
{
  "clinics": [
    {"id": "PKM-001", "name": "Puskesmas Sandik Raya", "timezone": "Asia/Makassar"},
    {"id": "PKM-002", "name": "Puskesmas Gunungsari Timur", "timezone": "Asia/Makassar"},
    {"id": "PKM-003", "name": "Puskesmas Kalimanah Wetan", "timezone": "Asia/Jakarta"}
  ],
  "stock": [
    {"medicine_id": "MED-001", "name": "Paracetamol 500mg", "quantity": 5000},
    {"medicine_id": "MED-002", "name": "Amoxicillin 500mg", "quantity": 3000},
    {"medicine_id": "MED-003", "name": "ORS (Oralit)", "quantity": 2000}
  ]
}
```

### Response (HTTP 200)

```json
{"status": "reset", "clinics": 3, "medicines": 3}
```

---

## Status Codes Summary

| Endpoint | Success | Error |
|----------|---------|-------|
| `POST /sync` | 200 (always, with per-record results) | 400 (malformed top-level structure) |
| `GET /patients/{id}` | 200 | 404 |
| `GET /stock/{id}` | 200 | 404 |
| `GET /review-queue` | 200 | - |
| `POST /reset` | 200 | - |
