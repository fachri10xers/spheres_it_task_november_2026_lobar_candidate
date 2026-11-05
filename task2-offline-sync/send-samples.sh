#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${1:-http://localhost:8000}"
DIR="$(cd "$(dirname "$0")/sample-payloads" && pwd)"

echo "Sending sample payloads to $BASE_URL"
echo "---"

# Reset server state first
echo "Resetting server..."
curl -s -X POST "$BASE_URL/reset" \
  -H "Content-Type: application/json" \
  -d '{
    "clinics": [
      {"id": "PKM-001", "name": "Puskesmas Sandik Raya", "timezone": "Asia/Makassar"},
      {"id": "PKM-002", "name": "Puskesmas Gunungsari Timur", "timezone": "Asia/Makassar"},
      {"id": "PKM-003", "name": "Puskesmas Kalimanah Wetan", "timezone": "Asia/Jakarta"}
    ],
    "stock": [
      {"medicine_id": "MED-001", "name": "Paracetamol 500mg", "quantity": 5000},
      {"medicine_id": "MED-002", "name": "Amoxicillin 500mg", "quantity": 3000},
      {"medicine_id": "MED-003", "name": "ORS (Oralit)", "quantity": 20}
    ]
  }' | python3 -m json.tool
echo ""

for f in "$DIR"/*.json; do
  name=$(basename "$f")
  echo ">>> $name"
  curl -s -X POST "$BASE_URL/sync" \
    -H "Content-Type: application/json" \
    -d @"$f" | python3 -m json.tool
  echo ""
done

echo "--- Review queue ---"
curl -s "$BASE_URL/review-queue" | python3 -m json.tool

echo ""
echo "--- Stock ---"
for med in MED-001 MED-002 MED-003; do
  curl -s "$BASE_URL/stock/$med" | python3 -m json.tool
done
