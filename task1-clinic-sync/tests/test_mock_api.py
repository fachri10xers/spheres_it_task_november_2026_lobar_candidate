def test_mock_api_health(mock_api_client):
    resp = mock_api_client.get("/health")
    assert resp.status_code == 200


def test_mock_api_token_flow(mock_api_client):
    resp = mock_api_client.post("/token", json={"client_id": "test"})
    assert resp.status_code == 200
    data = resp.json()
    assert "access_token" in data
    assert "expires_in" in data


def test_mock_api_submit_claim(mock_api_client):
    token_resp = mock_api_client.post("/token", json={"client_id": "test"})
    token = token_resp.json()["access_token"]

    claim_resp = mock_api_client.post(
        "/claims",
        headers={"Authorization": f"Bearer {token}"},
        json={
            "claim_id": "CLM-001",
            "amount": 150000,
            "patient_id": "PAT-001",
        },
    )
    assert claim_resp.status_code == 200
    assert claim_resp.json()["status"] == "accepted"

    list_resp = mock_api_client.get("/claims")
    assert len(list_resp.json()) == 1


def test_mock_api_requires_auth(mock_api_client):
    resp = mock_api_client.post(
        "/claims",
        json={"claim_id": "CLM-001", "amount": 100000, "patient_id": "PAT-001"},
    )
    assert resp.status_code == 401
