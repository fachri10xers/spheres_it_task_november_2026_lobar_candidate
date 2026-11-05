def test_dispensing_updates_stock(app_client, sample_batch):
    resp = app_client.get("/stock")
    initial_stock = {m["id"]: m["stock"] for m in resp.json()}

    sample_batch["dispensing"] = [
        {
            "visit_id": "visit-placeholder",
            "patient_nik": "5201141506900001",
            "medicine_id": "MED-001",
            "quantity": 10,
        }
    ]
    app_client.post("/batch", json=sample_batch)

    resp = app_client.get("/stock")
    updated_stock = {m["id"]: m["stock"] for m in resp.json()}
    assert updated_stock["MED-001"] == initial_stock["MED-001"] - 10
