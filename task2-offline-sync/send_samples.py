"""Send sample payloads to the sync endpoint and print responses."""

import json
import os
import sys
import urllib.request

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8000"
PAYLOADS_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "sample-payloads")


def post_json(url, data):
    body = json.dumps(data).encode("utf-8")
    req = urllib.request.Request(url, data=body, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read())


def get_json(url):
    with urllib.request.urlopen(url) as resp:
        return json.loads(resp.read())


def main():
    print(f"Sending sample payloads to {BASE_URL}")
    print("---")

    print("Resetting server...")
    result = post_json(f"{BASE_URL}/reset", {
        "clinics": [
            {"id": "PKM-001", "name": "Puskesmas Sandik Raya", "timezone": "Asia/Makassar"},
            {"id": "PKM-002", "name": "Puskesmas Gunungsari Timur", "timezone": "Asia/Makassar"},
            {"id": "PKM-003", "name": "Puskesmas Kalimanah Wetan", "timezone": "Asia/Jakarta"},
        ],
        "stock": [
            {"medicine_id": "MED-001", "name": "Paracetamol 500mg", "quantity": 5000},
            {"medicine_id": "MED-002", "name": "Amoxicillin 500mg", "quantity": 3000},
            {"medicine_id": "MED-003", "name": "ORS (Oralit)", "quantity": 20},
        ],
    })
    print(json.dumps(result, indent=2))
    print()

    for filename in sorted(os.listdir(PAYLOADS_DIR)):
        if not filename.endswith(".json"):
            continue
        filepath = os.path.join(PAYLOADS_DIR, filename)
        with open(filepath) as f:
            payload = json.load(f)

        print(f">>> {filename}")
        result = post_json(f"{BASE_URL}/sync", payload)
        print(json.dumps(result, indent=2))
        print()

    print("--- Review queue ---")
    print(json.dumps(get_json(f"{BASE_URL}/review-queue"), indent=2))

    print()
    print("--- Stock ---")
    for med in ["MED-001", "MED-002", "MED-003"]:
        print(json.dumps(get_json(f"{BASE_URL}/stock/{med}"), indent=2))


if __name__ == "__main__":
    main()
