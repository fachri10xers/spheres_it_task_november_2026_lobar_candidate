# Schema Reference

## clinics

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT | Facility identifier (e.g., `PKM-001`) |
| name | TEXT | Facility name |
| timezone | TEXT | IANA timezone (e.g., `Asia/Makassar`) |
| region_code | TEXT | Administrative region code |

## patients

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT | Patient UUID |
| nik | TEXT | NIK (Nomor Induk Kependudukan), 16 digits |
| name | TEXT | Patient full name |
| birth_date | TEXT | Date of birth (ISO format) |
| gender | TEXT | `M` or `F` |
| address | TEXT | Home address |
| clinic_id | TEXT | Primary facility |
| created_at | TEXT | Record creation timestamp |

## registrations

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT | Registration UUID |
| registration_number | TEXT | Facility-assigned sequential number |
| patient_id | TEXT | References `patients.id` |
| clinic_id | TEXT | Facility where registered |
| registered_at | TEXT | Registration timestamp |
| status | TEXT | Registration status |
| polyclinic | TEXT | Service unit (`umum`, `gigi`, `kia`, `lansia`) |

## visits

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT | Visit UUID |
| registration_id | TEXT | References `registrations.id` |
| patient_id | TEXT | References `patients.id` |
| clinic_id | TEXT | Facility where visit occurred |
| polyclinic | TEXT | Service unit attended |
| visit_date | TEXT | Visit date and time |
| diagnosis | TEXT | Diagnosis text |
| notes | TEXT | Clinical notes (may be null) |

## dispensing

| Column | Type | Description |
|--------|------|-------------|
| id | TEXT | Dispensing UUID |
| visit_id | TEXT | References `visits.id` |
| patient_id | TEXT | References `patients.id` |
| medicine_id | TEXT | Medicine identifier |
| quantity | INTEGER | Quantity dispensed |
| dispensed_at | TEXT | Dispensing timestamp |
