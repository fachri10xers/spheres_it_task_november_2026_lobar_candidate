package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS clinics (
	id       TEXT PRIMARY KEY,
	name     TEXT NOT NULL,
	timezone TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS medicines (
	id   TEXT PRIMARY KEY,
	name TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS stock_ledger (
	seq             INTEGER PRIMARY KEY AUTOINCREMENT,
	medicine_id     TEXT NOT NULL REFERENCES medicines(id),
	delta           INTEGER NOT NULL,
	ref_client_uuid TEXT UNIQUE,
	created_at      TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS records (
	client_uuid TEXT PRIMARY KEY,
	type        TEXT NOT NULL,
	server_id   TEXT NOT NULL,
	received_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS patients (
	server_id    TEXT PRIMARY KEY,
	client_uuid  TEXT NOT NULL UNIQUE,
	nik          TEXT NOT NULL DEFAULT '',
	name         TEXT NOT NULL,
	birth_date   TEXT NOT NULL,
	gender       TEXT NOT NULL,
	address      TEXT NOT NULL DEFAULT '',
	clinic_id    TEXT NOT NULL,
	device_recorded_at TEXT NOT NULL DEFAULT '',
	received_at  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS visits (
	server_id         TEXT PRIMARY KEY,
	client_uuid       TEXT NOT NULL UNIQUE,
	patient_server_id TEXT NOT NULL,
	clinic_id         TEXT NOT NULL,
	polyclinic        TEXT NOT NULL,
	visit_date        TEXT NOT NULL,
	diagnosis         TEXT NOT NULL DEFAULT '',
	notes             TEXT NOT NULL DEFAULT '',
	device_recorded_at TEXT NOT NULL DEFAULT '',
	received_at       TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS dispensings (
	server_id       TEXT PRIMARY KEY,
	client_uuid     TEXT NOT NULL UNIQUE,
	visit_server_id TEXT NOT NULL,
	medicine_id     TEXT NOT NULL,
	quantity        INTEGER NOT NULL,
	device_recorded_at TEXT NOT NULL DEFAULT '',
	received_at     TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS batch_responses (
	batch_id    TEXT PRIMARY KEY,
	device_id   TEXT NOT NULL,
	received_at TEXT NOT NULL,
	response    TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS review_queue (
	seq                INTEGER PRIMARY KEY AUTOINCREMENT,
	type               TEXT NOT NULL,
	new_patient_id     TEXT NOT NULL DEFAULT '',
	existing_patient_id TEXT NOT NULL DEFAULT '',
	reason             TEXT NOT NULL DEFAULT '',
	medicine_id        TEXT NOT NULL DEFAULT '',
	current_stock      INTEGER NOT NULL DEFAULT 0,
	created_at         TEXT NOT NULL
);
`

type store struct {
	db *sql.DB
	mu sync.Mutex
}

type Clinic struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
}

type MedicineSeed struct {
	ID       string `json:"medicine_id"`
	Name     string `json:"name"`
	Quantity int64  `json:"quantity"`
}

type Patient struct {
	ServerID         string
	ClientUUID       string
	NIK              string
	Name             string
	BirthDate        string
	Gender           string
	Address          string
	ClinicID         string
	DeviceRecordedAt string
	ReceivedAt       string
}

type Visit struct {
	ServerID         string
	ClientUUID       string
	PatientServerID  string
	ClinicID         string
	Polyclinic       string
	VisitDate        string
	Diagnosis        string
	Notes            string
	DeviceRecordedAt string
	ReceivedAt       string
}

type Dispensing struct {
	ServerID         string
	ClientUUID       string
	VisitServerID    string
	MedicineID       string
	Quantity         int64
	DeviceRecordedAt string
	ReceivedAt       string
}

type ReviewItem struct {
	Type              string
	NewPatientID      string
	ExistingPatientID string
	Reason            string
	MedicineID        string
	CurrentStock      int64
	CreatedAt         string
}

type recordEntry struct {
	ClientUUID string
	Type       string
	ServerID   string
	ReceivedAt string
}

func openStore(path string) (*store, error) {
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &store{db: db}, nil
}

func (s *store) close() error {
	return s.db.Close()
}

func (s *store) reset(clinics []Clinic, stock []MedicineSeed) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin reset: %w", err)
	}
	defer tx.Rollback()
	for _, table := range []string{
		"review_queue", "batch_responses", "dispensings", "visits",
		"patients", "stock_ledger", "records", "medicines", "clinics",
	} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}
	now := nowRFC3339()
	for _, c := range clinics {
		if _, err := tx.Exec(
			`INSERT INTO clinics (id, name, timezone) VALUES (?, ?, ?)`,
			c.ID, c.Name, c.Timezone,
		); err != nil {
			return fmt.Errorf("seed clinic %s: %w", c.ID, err)
		}
	}
	for _, m := range stock {
		if _, err := tx.Exec(
			`INSERT INTO medicines (id, name) VALUES (?, ?)`,
			m.ID, m.Name,
		); err != nil {
			return fmt.Errorf("seed medicine %s: %w", m.ID, err)
		}
		if err := addLedgerTx(tx, m.ID, m.Quantity, nil, now); err != nil {
			return fmt.Errorf("seed opening balance for %s: %w", m.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reset: %w", err)
	}
	return nil
}

func (s *store) currentStock(medicineID string) (int64, bool, error) {
	var sum int64
	var n int
	err := s.db.QueryRow(
		`SELECT (SELECT COALESCE(SUM(delta), 0) FROM stock_ledger WHERE medicine_id = ?),
		        (SELECT COUNT(*) FROM medicines WHERE id = ?)`,
		medicineID, medicineID,
	).Scan(&sum, &n)
	if err != nil {
		return 0, false, fmt.Errorf("read stock: %w", err)
	}
	if n == 0 {
		return 0, false, nil
	}
	return sum, true, nil
}

func (s *store) lookupRecord(clientUUID string) (recordEntry, bool, error) {
	var e recordEntry
	err := s.db.QueryRow(
		`SELECT client_uuid, type, server_id, received_at FROM records WHERE client_uuid = ?`,
		clientUUID,
	).Scan(&e.ClientUUID, &e.Type, &e.ServerID, &e.ReceivedAt)
	if err == sql.ErrNoRows {
		return recordEntry{}, false, nil
	}
	if err != nil {
		return recordEntry{}, false, fmt.Errorf("lookup record: %w", err)
	}
	return e, true, nil
}

func (s *store) saveBatchResponse(batchID, deviceID, receivedAt, response string) error {
	if _, err := s.db.Exec(
		`INSERT INTO batch_responses (batch_id, device_id, received_at, response) VALUES (?, ?, ?, ?)`,
		batchID, deviceID, receivedAt, response,
	); err != nil {
		return fmt.Errorf("save batch response: %w", err)
	}
	return nil
}

func (s *store) lookupBatchResponse(batchID string) (string, bool, error) {
	var response string
	err := s.db.QueryRow(
		`SELECT response FROM batch_responses WHERE batch_id = ?`,
		batchID,
	).Scan(&response)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("lookup batch response: %w", err)
	}
	return response, true, nil
}

func (s *store) listReviewItems() ([]ReviewItem, error) {
	rows, err := s.db.Query(
		`SELECT type, new_patient_id, existing_patient_id, reason, medicine_id, current_stock, created_at
		 FROM review_queue ORDER BY seq`,
	)
	if err != nil {
		return nil, fmt.Errorf("list review items: %w", err)
	}
	defer rows.Close()
	var items []ReviewItem
	for rows.Next() {
		var it ReviewItem
		if err := rows.Scan(&it.Type, &it.NewPatientID, &it.ExistingPatientID, &it.Reason, &it.MedicineID, &it.CurrentStock, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan review item: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *store) patientByServerID(serverID string) (Patient, bool, error) {
	var p Patient
	err := s.db.QueryRow(
		`SELECT server_id, client_uuid, nik, name, birth_date, gender, address, clinic_id, received_at
		 FROM patients WHERE server_id = ?`,
		serverID,
	).Scan(&p.ServerID, &p.ClientUUID, &p.NIK, &p.Name, &p.BirthDate, &p.Gender, &p.Address, &p.ClinicID, &p.ReceivedAt)
	if err == sql.ErrNoRows {
		return Patient{}, false, nil
	}
	if err != nil {
		return Patient{}, false, fmt.Errorf("query patient: %w", err)
	}
	return p, true, nil
}

func (s *store) patientsByBirthDate(birthDate string) ([]Patient, error) {
	rows, err := s.db.Query(
		`SELECT server_id, client_uuid, nik, name, birth_date, gender, address, clinic_id, received_at
		 FROM patients WHERE birth_date = ? ORDER BY received_at ASC, rowid ASC`,
		birthDate,
	)
	if err != nil {
		return nil, fmt.Errorf("query patients by birth date: %w", err)
	}
	defer rows.Close()
	var out []Patient
	for rows.Next() {
		var p Patient
		if err := rows.Scan(&p.ServerID, &p.ClientUUID, &p.NIK, &p.Name, &p.BirthDate, &p.Gender, &p.Address, &p.ClinicID, &p.ReceivedAt); err != nil {
			return nil, fmt.Errorf("scan patient: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *store) patientsByNIK(nik string) ([]Patient, error) {
	rows, err := s.db.Query(
		`SELECT server_id, client_uuid, nik, name, birth_date, gender, address, clinic_id, received_at
		 FROM patients WHERE nik = ? ORDER BY received_at ASC, rowid ASC`,
		nik,
	)
	if err != nil {
		return nil, fmt.Errorf("query patients by NIK: %w", err)
	}
	defer rows.Close()
	var out []Patient
	for rows.Next() {
		var p Patient
		if err := rows.Scan(&p.ServerID, &p.ClientUUID, &p.NIK, &p.Name, &p.BirthDate, &p.Gender, &p.Address, &p.ClinicID, &p.ReceivedAt); err != nil {
			return nil, fmt.Errorf("scan patient: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func beginTx(s *store) (*sql.Tx, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	return tx, nil
}

func addLedgerTx(tx *sql.Tx, medicineID string, delta int64, ref *string, createdAt string) error {
	if _, err := tx.Exec(
		`INSERT INTO stock_ledger (medicine_id, delta, ref_client_uuid, created_at) VALUES (?, ?, ?, ?)`,
		medicineID, delta, ref, createdAt,
	); err != nil {
		return fmt.Errorf("append ledger row: %w", err)
	}
	return nil
}

func insertRecordTx(tx *sql.Tx, e recordEntry) error {
	if _, err := tx.Exec(
		`INSERT INTO records (client_uuid, type, server_id, received_at) VALUES (?, ?, ?, ?)`,
		e.ClientUUID, e.Type, e.ServerID, e.ReceivedAt,
	); err != nil {
		return fmt.Errorf("register record: %w", err)
	}
	return nil
}

func insertPatientTx(tx *sql.Tx, p Patient) error {
	if _, err := tx.Exec(
		`INSERT INTO patients (server_id, client_uuid, nik, name, birth_date, gender, address, clinic_id, device_recorded_at, received_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ServerID, p.ClientUUID, p.NIK, p.Name, p.BirthDate, p.Gender, p.Address, p.ClinicID, p.DeviceRecordedAt, p.ReceivedAt,
	); err != nil {
		return fmt.Errorf("insert patient: %w", err)
	}
	return nil
}

func insertVisitTx(tx *sql.Tx, v Visit) error {
	if _, err := tx.Exec(
		`INSERT INTO visits (server_id, client_uuid, patient_server_id, clinic_id, polyclinic, visit_date, diagnosis, notes, device_recorded_at, received_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ServerID, v.ClientUUID, v.PatientServerID, v.ClinicID, v.Polyclinic, v.VisitDate, v.Diagnosis, v.Notes, v.DeviceRecordedAt, v.ReceivedAt,
	); err != nil {
		return fmt.Errorf("insert visit: %w", err)
	}
	return nil
}

func insertDispensingTx(tx *sql.Tx, d Dispensing) error {
	if _, err := tx.Exec(
		`INSERT INTO dispensings (server_id, client_uuid, visit_server_id, medicine_id, quantity, device_recorded_at, received_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		d.ServerID, d.ClientUUID, d.VisitServerID, d.MedicineID, d.Quantity, d.DeviceRecordedAt, d.ReceivedAt,
	); err != nil {
		return fmt.Errorf("insert dispensing: %w", err)
	}
	return nil
}

func addReviewItemTx(tx *sql.Tx, item ReviewItem) error {
	if _, err := tx.Exec(
		`INSERT INTO review_queue (type, new_patient_id, existing_patient_id, reason, medicine_id, current_stock, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		item.Type, item.NewPatientID, item.ExistingPatientID, item.Reason, item.MedicineID, item.CurrentStock, item.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert review item: %w", err)
	}
	return nil
}

func newUUIDv4() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Errorf("generate uuid: %w", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
