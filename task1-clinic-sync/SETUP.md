# Task 1: Setup Guide

## Prerequisites

Choose one:
- **Docker Desktop** (or Docker Engine with Compose v2)
- **Python 3.11+** with pip

## Option A: Docker

### Start the services

```bash
docker compose up --build
```

### Verify it works

```bash
curl http://localhost:8000/health
# {"status":"ok","version":"1.0.0"}

curl http://localhost:8001/health
# {"status":"ok","version":"1.0.0"}
```

### Run tests

```bash
docker compose run --rm clinic-sync pytest tests/ -v
```

### View logs

```bash
docker compose logs -f clinic-sync
docker compose logs -f mock-claim-api
```

### Reset the database

```bash
docker compose run --rm clinic-sync rm -f data/clinic.db
docker compose restart clinic-sync
```

### Stop everything

```bash
docker compose down
```

To also remove the data volume:

```bash
docker compose down -v
```

## Option B: Local Python

### Setup

```bash
make setup
```

### Start both services

```bash
make run
```

This starts the mock claim API on port 8001 (background) and the clinic sync service on port 8000.

### Run tests

```bash
make test
```

### Reset the database

```bash
make reset-db
```

## Changing Mock API Behavior

The mock claim API has configurable settings via environment variables. Edit `.env.example` or pass them directly:

| Variable | Default | Description |
|----------|---------|-------------|
| `MOCK_TOKEN_EXPIRY` | `3600` | Token validity in seconds |
| `MOCK_TIMEOUT_RATE` | `0.05` | Probability of simulated timeout (0.0-1.0) |
| `MOCK_RATE_LIMIT_WINDOW` | `1.0` | Rate limit window in seconds |
| `MOCK_RATE_LIMIT_MAX` | `10` | Max requests per window |

**Docker:** Edit the values in `docker-compose.yml` under the `mock-claim-api` service, then `docker compose restart mock-claim-api`.

**Local:** Export the variables before running, e.g.:

```bash
MOCK_TOKEN_EXPIRY=5 make run
```

## Try It Out

Send a sample batch:

```bash
curl -X POST http://localhost:8000/batch \
  -H "Content-Type: application/json" \
  -d '{
    "clinic_id": "PKM-001",
    "batch_id": "TEST-001",
    "timestamp": "2026-10-15T10:00:00",
    "patients": [{
      "nik": "5201141506900001",
      "name": "Ahmad Fauzi",
      "birth_date": "1990-06-15",
      "gender": "M",
      "address": "Jl. Raya Sandik No. 42",
      "clinic_id": "PKM-001"
    }],
    "visits": [{
      "patient_nik": "5201141506900001",
      "clinic_id": "PKM-001",
      "polyclinic": "umum",
      "visit_date": "2026-10-15T09:30:00",
      "diagnosis": "Common cold"
    }],
    "dispensing": [],
    "bills": [{
      "visit_id": "v-001",
      "patient_nik": "5201141506900001",
      "amount": 75000.0,
      "description": "Consultation"
    }]
  }'
```

Check claims:

```bash
curl http://localhost:8000/claims
```

Check stock:

```bash
curl http://localhost:8000/stock
```

## Project Layout

```
.
├── app.py               # Main clinic sync service
├── mock_claim_api.py    # Simulated claim API
├── models.py            # Pydantic models
├── database.py          # SQLite schema and helpers
├── config.py            # Configuration from env vars
├── tests/               # Test suite
├── data/                # SQLite database (auto-created)
├── docker-compose.yml   # Docker setup
├── Dockerfile           # Container image
├── Makefile             # Local development commands
├── requirements.txt     # Python dependencies
└── .env.example         # All configurable values
```

## Troubleshooting

**Port already in use:** Change the port mapping in `docker-compose.yml` (e.g., `"8080:8000"`) or set `PORT` when running locally.

**Docker on Apple Silicon:** The Python slim image supports arm64. No special configuration needed.

**Windows line endings:** The `.gitattributes` file enforces LF. If you cloned with CRLF, run `git checkout -- .` to re-checkout.

**Permission issues with data folder:** On Linux, the mounted volume may have root ownership. Run `sudo chown -R $(id -u):$(id -g) data/` to fix.
