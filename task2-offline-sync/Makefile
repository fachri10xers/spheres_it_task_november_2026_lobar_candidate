.DEFAULT_GOAL := help

ifeq ($(OS),Windows_NT)
BIN := sync-server.exe
PY  := python
else
BIN := sync-server
PY  := python3
endif

DB_FILES := sync.db, sync.db-wal, sync.db-shm

.PHONY: help run test vet build docker-build docker-up docker-down docker-logs docker-clean samples clean

help:
	@echo Targets:
	@echo   run           start the sync server on port 8000 (go run .)
	@echo   test          run all tests (go test ./...)
	@echo   vet           static analysis (go vet ./...)
	@echo   build         compile $(BIN)
	@echo   docker-build  build the container image
	@echo   docker-up     start the server in docker on port 8000
	@echo   docker-down   stop the docker server (bind-mounted data kept)
	@echo   docker-logs   follow docker server logs
	@echo   docker-clean  down + wipe local state + rebuild + start fresh
	@echo   samples       send the sample payloads to localhost:8000
	@echo   clean         remove the binary and local sqlite files

run:
	go run .

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -o $(BIN) .

docker-build:
	docker compose build

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-clean:
	docker compose down
	$(MAKE) clean
	docker compose up -d --build

samples:
	$(PY) send_samples.py http://localhost:8000

clean:
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -Command "Remove-Item -Force -ErrorAction Ignore $(BIN), syncserver.exe, $(DB_FILES); Remove-Item -Force -Recurse -ErrorAction Ignore .data; exit 0"
else
	rm -f $(BIN) syncserver syncserver.exe sync.db sync.db-wal sync.db-shm
	rm -rf .data
endif
