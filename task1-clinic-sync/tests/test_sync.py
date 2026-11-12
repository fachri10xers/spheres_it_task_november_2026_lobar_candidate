import time
import types


def _bill(amount, description="Consultation fee"):
    return {
        "visit_id": "visit-placeholder",
        "patient_nik": "5201141506900001",
        "amount": amount,
        "description": description,
    }


def _post_batch_create_claims(app_client, sample_batch, bills):
    before = {c["id"] for c in app_client.get("/claims").json()}
    sample_batch["bills"] = bills
    resp = app_client.post("/batch", json=sample_batch)
    assert resp.status_code == 200
    assert resp.json()["claims_created"] == len(bills)
    after = app_client.get("/claims").json()
    new_claims = [c for c in after if c["id"] not in before]
    assert len(new_claims) == len(bills)
    new_claims.sort(key=lambda c: c["created_at"])
    return new_claims


def _get_claims_by_id(app_client):
    return {c["id"]: c for c in app_client.get("/claims").json()}


def _patch_httpx_to_mock(app_module, mock_api_client, monkeypatch):
    def fake_post(url, headers=None, json=None, timeout=None, **kwargs):
        if url.endswith("/token"):
            return mock_api_client.post("/token", json=json)
        if url.endswith("/claims"):
            return mock_api_client.post("/claims", headers=headers, json=json)
        raise AssertionError(f"Unexpected URL: {url}")

    monkeypatch.setattr(app_module, "_token", None)
    monkeypatch.setattr(app_module, "httpx", types.SimpleNamespace(post=fake_post))


def _wait_until_synced(client, claim_ids, timeout=5.0):
    deadline = time.time() + timeout
    rows = _get_claims_by_id(client)
    while time.time() < deadline and not all(
        rows[cid]["synced"] == 1 for cid in claim_ids
    ):
        time.sleep(0.05)
        rows = _get_claims_by_id(client)
    return rows


def test_sync_failure_leaves_claim_pending(app_client, sample_batch, monkeypatch):
    import app as app_module
    monkeypatch.setattr(
        app_module,
        "submit_claim_to_api",
        lambda claim_id, claim_amount, patient_id: None,
    )

    new_claims = _post_batch_create_claims(app_client, sample_batch, [_bill(150000.0)])

    resp = app_client.post("/sync")
    assert resp.status_code == 200

    claims_after = _get_claims_by_id(app_client)
    for claim in new_claims:
        current = claims_after[claim["id"]]
        assert current["synced"] == 0
        assert current["status"] == "pending"
        assert current["external_claim_id"] is None
        assert current["submitted_at"] is None


def test_sync_success_saves_external_claim_id(app_client, sample_batch, mock_api_client, monkeypatch):
    import app as app_module
    _patch_httpx_to_mock(app_module, mock_api_client, monkeypatch)

    new_claims = _post_batch_create_claims(app_client, sample_batch, [_bill(150000.0)])

    resp = app_client.post("/sync")
    assert resp.status_code == 200

    claims_after = _get_claims_by_id(app_client)
    for claim in new_claims:
        current = claims_after[claim["id"]]
        assert current["synced"] == 1
        assert current["status"] == "synced"
        assert current["submitted_at"] is not None
        assert current["external_claim_id"]

    external_records = mock_api_client.get("/claims").json()
    for claim in new_claims:
        match = [
            e for e in external_records
            if e["original_claim_id"] == claim["id"]
        ]
        assert len(match) == 1
        assert match[0]["claim_id"] == claims_after[claim["id"]]["external_claim_id"]


def test_sync_only_marks_accepted_claims(app_client, sample_batch, monkeypatch):
    import app as app_module

    new_claims = _post_batch_create_claims(app_client, sample_batch, [_bill(150000.0), _bill(200000.0, "Procedure fee")])
    accepted_claim, rejected_claim = new_claims[0], new_claims[1]

    def fake_submit(claim_id, claim_amount, patient_id):
        if claim_id == accepted_claim["id"]:
            return {"claim_id": f"EXT-{claim_id}", "status": "accepted"}
        return None

    monkeypatch.setattr(app_module, "submit_claim_to_api", fake_submit)

    resp = app_client.post("/sync")
    assert resp.status_code == 200

    claims_after = _get_claims_by_id(app_client)

    synced = claims_after[accepted_claim["id"]]
    assert synced["synced"] == 1
    assert synced["status"] == "synced"
    assert synced["external_claim_id"] == f"EXT-{accepted_claim['id']}"

    pending = claims_after[rejected_claim["id"]]
    assert pending["synced"] == 0
    assert pending["status"] == "pending"
    assert pending["external_claim_id"] is None


def test_background_sync_submits_pending_claims_automatically(sample_batch, mock_api_client, monkeypatch):
    import app as app_module
    from fastapi.testclient import TestClient

    monkeypatch.setattr(app_module, "SYNC_INTERVAL_SECONDS", 0.2, raising=False)
    _patch_httpx_to_mock(app_module, mock_api_client, monkeypatch)

    with TestClient(app_module.app) as client:
        first_batch = _post_batch_create_claims(client, sample_batch, [_bill(150000.0)])
        rows = _wait_until_synced(client, [c["id"] for c in first_batch])
        for claim in first_batch:
            current = rows[claim["id"]]
            assert current["synced"] == 1
            assert current["status"] == "synced"

        external_records = mock_api_client.get("/claims").json()
        for claim in first_batch:
            match = [e for e in external_records if e["original_claim_id"] == claim["id"]]
            assert len(match) == 1

        second_batch = _post_batch_create_claims(client, sample_batch, [_bill(210000.0)])
        rows = _wait_until_synced(client, [c["id"] for c in second_batch])
        for claim in second_batch:
            current = rows[claim["id"]]
            assert current["synced"] == 1
            assert current["status"] == "synced"


def test_startup_syncs_pending_backlog(app_client, sample_batch, mock_api_client, monkeypatch):
    import app as app_module
    from fastapi.testclient import TestClient

    monkeypatch.setattr(app_module, "SYNC_INTERVAL_SECONDS", 0.2, raising=False)
    _patch_httpx_to_mock(app_module, mock_api_client, monkeypatch)

    new_claims = _post_batch_create_claims(app_client, sample_batch, [_bill(180000.0)])

    with TestClient(app_module.app) as client:
        rows = _wait_until_synced(client, [c["id"] for c in new_claims])
        for claim in new_claims:
            current = rows[claim["id"]]
            assert current["synced"] == 1
            assert current["status"] == "synced"
            assert current["external_claim_id"]
