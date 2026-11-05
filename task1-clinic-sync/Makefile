.PHONY: setup run run-mock run-app test reset-db clean

VENV := .venv
PYTHON := $(VENV)/bin/python
PIP := $(VENV)/bin/pip

setup:
	python3 -m venv $(VENV)
	$(PIP) install -r requirements.txt
	mkdir -p data

run-mock:
	$(PYTHON) -m uvicorn mock_claim_api:app --host 0.0.0.0 --port 8001 --reload &

run-app:
	$(PYTHON) -m uvicorn app:app --host 0.0.0.0 --port 8000 --reload

run: run-mock
	@sleep 1
	@$(MAKE) run-app

test:
	$(PYTHON) -m pytest tests/ -v

reset-db:
	rm -f data/clinic.db
	@echo "Database reset. It will be recreated on next startup."

clean:
	rm -rf $(VENV) __pycache__ .pytest_cache data/clinic.db
