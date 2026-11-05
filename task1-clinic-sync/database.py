import sqlite3
from contextlib import contextmanager

from config import DATABASE_PATH


SCHEMA = """
CREATE TABLE IF NOT EXISTS clinics (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    timezone TEXT NOT NULL,
    region_code TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS patients (
    id TEXT PRIMARY KEY,
    nik TEXT NOT NULL,
    name TEXT NOT NULL,
    birth_date TEXT NOT NULL,
    gender TEXT NOT NULL,
    address TEXT,
    clinic_id TEXT REFERENCES clinics(id),
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS registrations (
    id TEXT PRIMARY KEY,
    registration_number TEXT NOT NULL,
    patient_id TEXT REFERENCES patients(id),
    clinic_id TEXT REFERENCES clinics(id),
    registered_at TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    polyclinic TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS visits (
    id TEXT PRIMARY KEY,
    registration_id TEXT REFERENCES registrations(id),
    patient_id TEXT REFERENCES patients(id),
    clinic_id TEXT REFERENCES clinics(id),
    polyclinic TEXT NOT NULL,
    visit_date TEXT NOT NULL,
    diagnosis TEXT,
    notes TEXT
);

CREATE TABLE IF NOT EXISTS medicines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    unit TEXT NOT NULL,
    stock INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS dispensing (
    id TEXT PRIMARY KEY,
    visit_id TEXT REFERENCES visits(id),
    patient_id TEXT REFERENCES patients(id),
    medicine_id TEXT REFERENCES medicines(id),
    quantity INTEGER NOT NULL,
    dispensed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS bills (
    id TEXT PRIMARY KEY,
    visit_id TEXT REFERENCES visits(id),
    patient_id TEXT REFERENCES patients(id),
    amount REAL NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS claims (
    id TEXT PRIMARY KEY,
    bill_id TEXT REFERENCES bills(id),
    patient_id TEXT REFERENCES patients(id),
    claim_amount REAL NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    synced INTEGER NOT NULL DEFAULT 0,
    external_claim_id TEXT,
    submitted_at TEXT,
    created_at TEXT NOT NULL
);

INSERT OR IGNORE INTO clinics (id, name, timezone, region_code) VALUES
    ('PKM-001', 'Puskesmas Sandik Raya', 'Asia/Makassar', '5201'),
    ('PKM-002', 'Puskesmas Gunungsari Timur', 'Asia/Makassar', '5201'),
    ('PKM-003', 'Puskesmas Kalimanah Wetan', 'Asia/Jakarta', '3303');

INSERT OR IGNORE INTO medicines (id, name, unit, stock) VALUES
    ('MED-001', 'Paracetamol 500mg', 'tablet', 5000),
    ('MED-002', 'Amoxicillin 500mg', 'kaplet', 3000),
    ('MED-003', 'ORS (Oralit)', 'sachet', 2000),
    ('MED-004', 'Ibuprofen 400mg', 'tablet', 4000),
    ('MED-005', 'Metformin 500mg', 'tablet', 2500),
    ('MED-006', 'Salbutamol Inhaler', 'botol', 500),
    ('MED-007', 'Vitamin B Complex', 'tablet', 6000),
    ('MED-008', 'Antasida DOEN', 'tablet', 3500);
"""


def init_db():
    conn = sqlite3.connect(DATABASE_PATH)
    conn.executescript(SCHEMA)
    conn.close()


@contextmanager
def get_db():
    conn = sqlite3.connect(DATABASE_PATH)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA foreign_keys=OFF")
    try:
        yield conn
    finally:
        conn.close()
