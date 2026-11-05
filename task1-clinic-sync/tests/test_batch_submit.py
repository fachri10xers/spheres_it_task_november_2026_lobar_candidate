def test_submit_valid_batch(app_client, sample_batch):
    resp = app_client.post("/batch", json=sample_batch)
    assert resp.status_code == 200
    data = resp.json()
    assert data["batch_id"] == "BATCH-001"
    assert data["patients_created"] == 1
    assert data["visits_recorded"] == 1
    assert data["errors"] == []


def test_submit_batch_unknown_clinic(app_client, sample_batch):
    sample_batch["clinic_id"] = "PKM-999"
    resp = app_client.post("/batch", json=sample_batch)
    assert resp.status_code == 400


def test_submit_batch_with_bills(app_client, sample_batch):
    sample_batch["bills"] = [
        {
            "visit_id": "visit-placeholder",
            "patient_nik": "5201141506900001",
            "amount": 150000.0,
            "description": "Consultation fee",
        }
    ]
    resp = app_client.post("/batch", json=sample_batch)
    assert resp.status_code == 200
    data = resp.json()
    assert data["bills_created"] == 1
    assert data["claims_created"] == 1
