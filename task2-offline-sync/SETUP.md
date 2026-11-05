# Task 2: Sample Payloads

## What's Included

The `sample-payloads/` folder contains 7 JSON files representing realistic sync scenarios. They are numbered in the order they should be sent:

| File | Scenario |
|------|----------|
| `01-normal-batch.json` | A normal batch from one device |
| `02-duplicate-batch.json` | The same batch sent again (identical `batch_id`) |
| `03-partial-invalid.json` | 2 invalid records among valid ones |
| `04-duplicate-patients.json` | Near-duplicate patients from different devices |
| `05-stock-conflict.json` | Two devices dispensing the last stock |
| `06-wrong-clock.json` | A device with its clock set to 2019 |
| `07-old-schema-version.json` | Records with an older schema version |

## Sending the Payloads

### Using the shell script (macOS/Linux)

```bash
chmod +x send-samples.sh
./send-samples.sh
```

To use a different port:

```bash
./send-samples.sh http://localhost:9000
```

### Using Python (all platforms)

```bash
python3 send_samples.py
```

To use a different port:

```bash
python3 send_samples.py http://localhost:9000
```

### Manually

```bash
curl -X POST http://localhost:8000/sync \
  -H "Content-Type: application/json" \
  -d @sample-payloads/01-normal-batch.json
```

## Before Sending Payloads

Your server must be running on port 8000 and the `/reset` endpoint must be called first to seed initial data:

```bash
curl -X POST http://localhost:8000/reset \
  -H "Content-Type: application/json" \
  -d '{
    "clinics": [
      {"id": "PKM-001", "name": "Puskesmas Sandik Raya", "timezone": "Asia/Makassar"},
      {"id": "PKM-002", "name": "Puskesmas Gunungsari Timur", "timezone": "Asia/Makassar"},
      {"id": "PKM-003", "name": "Puskesmas Kalimanah Wetan", "timezone": "Asia/Jakarta"}
    ],
    "stock": [
      {"medicine_id": "MED-001", "name": "Paracetamol 500mg", "quantity": 5000},
      {"medicine_id": "MED-002", "name": "Amoxicillin 500mg", "quantity": 3000},
      {"medicine_id": "MED-003", "name": "ORS (Oralit)", "quantity": 20}
    ]
  }'
```

Note: MED-003 has only 20 units - this is intentional for the stock conflict scenario.
