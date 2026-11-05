def test_patient_created_and_listed(app_client, sample_batch):
    app_client.post("/batch", json=sample_batch)

    resp = app_client.get("/patients", params={"clinic_id": "PKM-001"})
    assert resp.status_code == 200
    patients = resp.json()
    assert len(patients) >= 1
    assert patients[0]["name"] == "Ahmad Fauzi"


def test_existing_patient_reused(app_client, sample_batch):
    app_client.post("/batch", json=sample_batch)
    resp2 = app_client.post("/batch", json=sample_batch)
    data = resp2.json()
    assert data["patients_existing"] == 1
    assert data["patients_created"] == 0
