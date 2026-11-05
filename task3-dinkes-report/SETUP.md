# Task 3: Setup

## What You Need

Only the SQLite file and any query tool. No server, no Docker, no dependencies.

### SQLite clients (pick any)

- **Command line:** `sqlite3` (pre-installed on macOS/Linux)
- **GUI:** [DB Browser for SQLite](https://sqlitebrowser.org/) or [DBeaver](https://dbeaver.io/)
- **Python:** `import sqlite3` (built-in, no pip install needed)
- **Any other tool** that reads SQLite

## Quick Start

Verify the file opens:

```bash
sqlite3 clinic_data.sqlite ".tables"
```

Expected output:

```
clinics        dispensing     patients       registrations  visits
```

Check row counts:

```bash
sqlite3 clinic_data.sqlite "SELECT name, (SELECT COUNT(*) FROM clinics) FROM sqlite_master WHERE name='clinics' UNION ALL SELECT 'patients', COUNT(*) FROM patients UNION ALL SELECT 'visits', COUNT(*) FROM visits UNION ALL SELECT 'registrations', COUNT(*) FROM registrations UNION ALL SELECT 'dispensing', COUNT(*) FROM dispensing;"
```

## Notes

- Work on a copy of the file and keep the original untouched.
- You may use SQL, Python/pandas, R, or any tool — as long as the final numbers are reproducible from what you submit.
- If using Python, `sqlite3` is built in — no extra packages needed.
