# How to Run

**Prerequisites:** Go 1.27+ (`go version`). Everything below is runnable from this folder — SQLite is accessed through a pure-Go driver, so no C compiler is needed.

**1. Start the server on port 8000:**

```bash
go run .
```

**2. Run the tests:**

```bash
go test ./...
```

**3. (Optional) Send the sample payloads** — the script resets the server first (seeding MED-003 at 20 units intentionally), then sends payloads 01–07 in order:

```bash
python send_samples.py          # macOS/Linux: python3 send_samples.py
```

Calling `/reset` before testing manually is required; the payloads assume the seed state.

## Configuration

The server reads two variables, each from the environment or from a `.env` file in this folder (both optional, sensible defaults):

| Variable | Default | Meaning |
|----------|---------|---------|
| `PORT`   | `8000`  | HTTP listen port |
| `DB_PATH` | `./.data/sync.db` — the `.data` directory is auto-created on first run (same folder the docker bind mount uses) | SQLite database file (created automatically; `-wal`/`-shm` sidecars appear next to it; all gitignored) |

## Makefile targets (optional convenience)

The `Makefile` works on Windows, macOS, and Linux (Windows needs GNU Make, e.g. `choco install make` — or just use the raw `go`/`docker` commands above):

| Target | Effect |
|--------|--------|
| `make run` | `go run .` |
| `make test` | `go test ./...` |
| `make vet` | `go vet ./...` |
| `make build` | compile `sync-server(.exe)` |
| `make samples` | send payloads 01–07 to localhost:8000 (`python`/`python3` per OS) |
| `make docker-up` / `docker-down` / `docker-logs` | compose lifecycle |
| `make docker-clean` | stop the stack, wipe local state (`.data/`), rebuild, start fresh |
| `make clean` | delete the local binary + `sync.db` files |

## Docker (optional)

```bash
docker compose up -d --build    # server on port 8000
python send_samples.py http://localhost:8000
docker compose down             # stop (bind-mounted data kept)
```

The image is a two-stage build (`golang:1.27-alpine` → `alpine:3`) with `CGO_ENABLED=0`. The database lives at `/data/sync.db` inside the container — **bind-mounted to `.data/` in this folder**, so `task2-offline-sync\.data\sync.db` exists on your host as soon as the container boots (sidecars `-wal`/`-shm` appear beside it; all gitignored). POSTing `/reset` re-seeds state as usual (`docker compose logs sync` shows `reset: wiped prior state, seeded N clinics, N medicines` for every successful reset). To wipe local state completely in one command, use `make docker-clean` (stops the stack, deletes `.data/`, rebuilds, boots fresh); `docker compose down` alone never deletes the database.

### Browsing the database with DBeaver (or any SQLite client)

**Important — never open `.data\sync.db` from the host while the container is running.** SQLite in WAL mode coordinates connections through the `-shm` shared-memory file, which cannot be shared across the Windows bind-mount boundary. A host client opening the live file recreates the sidecars and orphans the server's data view — rows disappear from the file while the API keeps serving them. (Discovered the hard way; this is a Docker-Desktop-on-Windows limitation, not an application bug.)

**Recommended workflow — stop, browse, resume:**

```bash
docker compose down     # on last connection close, SQLite checkpoints: -wal empties into sync.db
```

1. DBeaver → **New Database Connection → SQLite**
2. *Open file* → `task2-offline-sync\.data\sync.db` (now complete and consistent — the server is stopped)
3. *Test Connection* — DBeaver downloads its bundled SQLite driver on first use — then Finish

```bash
docker compose up -d    # resume; the same .data/sync.db (with your browsed rows) is reopened
```

Data persists across this cycle — the file lives in `.data/`, unaffected by `down`/`up`.

**Alternative — snapshot without stopping the stack** (browse a frozen copy while the server keeps running):

```bash
docker compose cp sync:/data/sync.db ./db-snapshot/sync.db
docker compose cp sync:/data/sync.db-wal ./db-snapshot/sync.db-wal
docker compose cp sync:/data/sync.db-shm ./db-snapshot/sync.db-shm
```

Include all three files, or the snapshot misses transactions still in the WAL. Open `db-snapshot\sync.db` in DBeaver; refresh = repeat the copy.

Tables: `clinics`, `medicines`, `stock_ledger` (append-only ledger; balances are the `SUM(delta)`), `records` (client_uuid idempotency registry), `patients`, `visits`, `dispensings`, `batch_responses` (replay storage), `review_queue`. Note: right after `POST /reset`, `records`/`patients`/`visits`/`review_queue` are empty **by design** — reset wipes sync state and re-seeds only clinics/stock; data appears after `/sync` calls. The server is the intended writer; browse read-only.
