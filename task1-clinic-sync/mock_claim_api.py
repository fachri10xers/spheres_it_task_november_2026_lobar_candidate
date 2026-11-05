import os
import random
import time
import uuid

from fastapi import FastAPI, Header, HTTPException
from pydantic import BaseModel

app = FastAPI(title="Mock Claim API", version="1.0.0")

TOKEN_EXPIRY = int(os.getenv("MOCK_TOKEN_EXPIRY", "3600"))
TIMEOUT_RATE = float(os.getenv("MOCK_TIMEOUT_RATE", "0.05"))
RATE_LIMIT_WINDOW = float(os.getenv("MOCK_RATE_LIMIT_WINDOW", "1.0"))
RATE_LIMIT_MAX = int(os.getenv("MOCK_RATE_LIMIT_MAX", "10"))

tokens: dict[str, float] = {}
claims: list[dict] = []
request_timestamps: list[float] = []


class TokenRequest(BaseModel):
    client_id: str


class ClaimRequest(BaseModel):
    claim_reference: str | None = None
    claim_id: str
    amount: int | float
    patient_id: str


@app.get("/health")
def health():
    return {"status": "ok", "version": "1.0.0"}


@app.post("/token")
def issue_token(req: TokenRequest):
    token = str(uuid.uuid4())
    tokens[token] = time.time() + TOKEN_EXPIRY
    return {"access_token": token, "expires_in": TOKEN_EXPIRY}


def validate_token(authorization: str | None):
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Missing or invalid Authorization header")
    token = authorization.removeprefix("Bearer ")
    if token not in tokens:
        raise HTTPException(status_code=401, detail="Invalid token")
    if time.time() > tokens[token]:
        del tokens[token]
        raise HTTPException(status_code=401, detail="Token expired")


def check_rate_limit():
    now = time.time()
    request_timestamps.append(now)
    cutoff = now - RATE_LIMIT_WINDOW
    while request_timestamps and request_timestamps[0] < cutoff:
        request_timestamps.pop(0)
    if len(request_timestamps) > RATE_LIMIT_MAX:
        raise HTTPException(
            status_code=429,
            detail="Too many requests",
            headers={"Retry-After": str(int(RATE_LIMIT_WINDOW))},
        )


@app.post("/claims")
def submit_claim(claim: ClaimRequest, authorization: str = Header(None)):
    validate_token(authorization)
    check_rate_limit()

    if random.random() < TIMEOUT_RATE:
        time.sleep(15)

    claim_record = {
        "claim_id": str(uuid.uuid4()),
        "claim_reference": claim.claim_reference,
        "original_claim_id": claim.claim_id,
        "amount": claim.amount,
        "patient_id": claim.patient_id,
        "status": "accepted",
        "received_at": time.time(),
    }
    claims.append(claim_record)
    return {"claim_id": claim_record["claim_id"], "status": "accepted"}


@app.get("/claims")
def list_claims():
    return claims


@app.post("/reset")
def reset():
    tokens.clear()
    claims.clear()
    request_timestamps.clear()
    return {"status": "reset"}
