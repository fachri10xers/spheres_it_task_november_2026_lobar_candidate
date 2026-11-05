from typing import Optional

from pydantic import BaseModel


class PatientRecord(BaseModel):
    nik: str
    name: str
    birth_date: str
    gender: str
    address: Optional[str] = None
    clinic_id: str


class VisitRecord(BaseModel):
    patient_nik: str
    clinic_id: str
    polyclinic: str
    visit_date: str
    diagnosis: Optional[str] = None
    notes: Optional[str] = None


class DispensingRecord(BaseModel):
    visit_id: str
    patient_nik: str
    medicine_id: str
    quantity: int


class BillRecord(BaseModel):
    visit_id: str
    patient_nik: str
    amount: float
    description: Optional[str] = None


class BatchPayload(BaseModel):
    clinic_id: str
    batch_id: str
    timestamp: str
    patients: list[PatientRecord] = []
    visits: list[VisitRecord] = []
    dispensing: list[DispensingRecord] = []
    bills: list[BillRecord] = []


class BatchResponse(BaseModel):
    batch_id: str
    patients_created: int
    patients_existing: int
    visits_recorded: int
    dispensing_recorded: int
    bills_created: int
    claims_created: int
    errors: list[str] = []


class HealthResponse(BaseModel):
    status: str
    version: str
