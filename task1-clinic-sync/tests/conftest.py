import os
import sys
import tempfile

import pytest
from fastapi.testclient import TestClient

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

os.environ["MOCK_TIMEOUT_RATE"] = "0"
os.environ["MOCK_RATE_LIMIT_MAX"] = "1000"
os.environ["MOCK_TOKEN_EXPIRY"] = "3600"


@pytest.fixture(autouse=True)
def temp_database(tmp_path):
    db_path = str(tmp_path / "test.db")
    os.environ["DATABASE_PATH"] = db_path
    import config
    config.DATABASE_PATH = db_path
    yield db_path


@pytest.fixture
def app_client(temp_database):
    from app import app
    from database import init_db
    init_db()
    return TestClient(app)


@pytest.fixture
def mock_api_client():
    from mock_claim_api import app as mock_app, tokens, claims, request_timestamps
    tokens.clear()
    claims.clear()
    request_timestamps.clear()
    return TestClient(mock_app)


@pytest.fixture
def sample_batch():
    return {
        "clinic_id": "PKM-001",
        "batch_id": "BATCH-001",
        "timestamp": "2026-10-15T10:00:00",
        "patients": [
            {
                "nik": "5201141506900001",
                "name": "Ahmad Fauzi",
                "birth_date": "1990-06-15",
                "gender": "M",
                "address": "Jl. Raya Sandik No. 42",
                "clinic_id": "PKM-001",
            }
        ],
        "visits": [
            {
                "patient_nik": "5201141506900001",
                "clinic_id": "PKM-001",
                "polyclinic": "umum",
                "visit_date": "2026-10-15T09:30:00",
                "diagnosis": "Common cold",
                "notes": "Follow up in 3 days",
            }
        ],
        "dispensing": [],
        "bills": [],
    }
