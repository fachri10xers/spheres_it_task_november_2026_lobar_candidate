import os


CLAIM_API_URL = os.getenv("CLAIM_API_URL", "http://localhost:8001")
DATABASE_PATH = os.getenv("DATABASE_PATH", "data/clinic.db")
SYNC_INTERVAL_SECONDS = int(os.getenv("SYNC_INTERVAL_SECONDS", "30"))
LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO")
