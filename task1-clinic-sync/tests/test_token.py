import threading
import time
import types

from mock_claim_api import tokens as mock_tokens
from tests.test_sync import (
    _bill,
    _get_claims_by_id,
    _patch_httpx_to_mock,
    _post_batch_create_claims,
)


def test_expired_token_401_triggers_refresh_and_resubmission(app_client, sample_batch, mock_api_client, monkeypatch):
    import app as app_module
    _patch_httpx_to_mock(app_module, mock_api_client, monkeypatch)

    token_resp = mock_api_client.post("/token", json={"client_id": "clinic-sync"})
    stale_token = token_resp.json()["access_token"]
    mock_tokens[stale_token] = time.time() - 1.0

    monkeypatch.setattr(app_module, "_token", stale_token)
    monkeypatch.setattr(app_module, "_token_expiry", time.time() + 3600, raising=False)

    new_claims = _post_batch_create_claims(app_client, sample_batch, [_bill(150000.0)])

    resp = app_client.post("/sync")
    assert resp.status_code == 200

    claims_after = _get_claims_by_id(app_client)
    for claim in new_claims:
        current = claims_after[claim["id"]]
        assert current["synced"] == 1
        assert current["status"] == "synced"
        assert current["external_claim_id"]

    external_records = mock_api_client.get("/claims").json()
    for claim in new_claims:
        match = [e for e in external_records if e["original_claim_id"] == claim["id"]]
        assert len(match) == 1
        assert claims_after[claim["id"]]["external_claim_id"] == match[0]["claim_id"]


def test_expired_cached_token_is_refreshed_before_submission(app_client, sample_batch, mock_api_client, monkeypatch):
    import app as app_module
    _patch_httpx_to_mock(app_module, mock_api_client, monkeypatch)

    token_resp = mock_api_client.post("/token", json={"client_id": "clinic-sync"})
    token = token_resp.json()["access_token"]
    tokens_before = len(mock_tokens)

    monkeypatch.setattr(app_module, "_token", token)
    monkeypatch.setattr(app_module, "_token_expiry", time.time() - 1.0, raising=False)

    new_claims = _post_batch_create_claims(app_client, sample_batch, [_bill(160000.0)])

    resp = app_client.post("/sync")
    assert resp.status_code == 200

    claims_after = _get_claims_by_id(app_client)
    for claim in new_claims:
        current = claims_after[claim["id"]]
        assert current["synced"] == 1
        assert current["external_claim_id"]

    assert len(mock_tokens) == tokens_before + 1


def test_concurrent_token_refresh_fetches_once(mock_api_client, monkeypatch):
    import app as app_module
    _patch_httpx_to_mock(app_module, mock_api_client, monkeypatch)

    def slow_token_post(url, headers=None, json=None, timeout=None, **kwargs):
        if url.endswith("/token"):
            time.sleep(0.15)
            return mock_api_client.post("/token", json=json)
        if url.endswith("/claims"):
            return mock_api_client.post("/claims", headers=headers, json=json)
        raise AssertionError(f"Unexpected URL: {url}")

    monkeypatch.setattr(app_module, "httpx", types.SimpleNamespace(post=slow_token_post))

    thread_count = 5
    barrier = threading.Barrier(thread_count)
    results = []
    errors = []

    def worker():
        try:
            barrier.wait(timeout=5)
            results.append(app_module.get_token())
        except Exception as e:
            errors.append(e)

    workers = [threading.Thread(target=worker) for _ in range(thread_count)]
    for w in workers:
        w.start()
    for w in workers:
        w.join(timeout=10)

    assert not errors
    assert len(results) == thread_count
    assert len(set(results)) == 1
    assert len(mock_tokens) == 1
