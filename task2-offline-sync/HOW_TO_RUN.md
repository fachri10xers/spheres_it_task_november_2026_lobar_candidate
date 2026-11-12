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
| `DB_PATH`| `sync.db` | SQLite database file (created automatically; `-wal`/`-shm` sidecars appear next to it; all gitignored) |

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
| `make clean` | delete the local binary + `sync.db` files |

## Docker (optional)

```bash
docker compose up -d --build    # server on port 8000, sqlite stored in a named volume
python send_samples.py http://localhost:8000
docker compose down             # stop (volume kept; add -v to wipe data)
```

The image is a two-stage build (`golang:1.27-alpine` → `alpine:3`) with `CGO_ENABLED=0`; the database lives at `/data/sync.db` inside the container and persists across restarts. POSTing `/reset` re-seeds state as usual.
