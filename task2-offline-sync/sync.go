package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
)

const (
	statusAccepted      = "accepted"
	statusDuplicate     = "duplicate"
	statusRejected      = "rejected"
	statusFlaggedDupPat = "flagged_possible_duplicate_patient"

	typePatient    = "patient_registration"
	typeVisit      = "visit"
	typeDispensing = "dispensing"
)

type syncRecordIn struct {
	Type             string          `json:"type"`
	ClientUUID       string          `json:"client_uuid"`
	DeviceRecordedAt string          `json:"device_recorded_at"`
	SchemaVersion    json.RawMessage `json:"schema_version"`
	Data             map[string]any  `json:"data"`
}

type syncRequest struct {
	DeviceID string         `json:"device_id"`
	BatchID  string         `json:"batch_id"`
	Records  []syncRecordIn `json:"records"`
}

type recordResult struct {
	ClientUUID        string `json:"client_uuid"`
	Status            string `json:"status"`
	ServerID          string `json:"server_id,omitempty"`
	Reason            string `json:"reason,omitempty"`
	ExistingPatientID string `json:"existing_patient_id,omitempty"`
}

type stockUpdate struct {
	MedicineID   string `json:"medicine_id"`
	CurrentStock int64  `json:"current_stock"`
}

type syncResponse struct {
	BatchID          string         `json:"batch_id"`
	ServerReceivedAt string         `json:"server_received_at"`
	Results          []recordResult `json:"results"`
	StockUpdates     []stockUpdate  `json:"stock_updates"`
}

var requiredFieldsByType = map[string][]string{
	typePatient:    {"nik", "name", "birth_date", "gender", "clinic_id"},
	typeVisit:      {"patient_client_uuid", "clinic_id", "polyclinic", "visit_date"},
	typeDispensing: {"visit_client_uuid", "medicine_id", "quantity"},
}

func handleSync(st *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := readSyncRequest(w, r)
		if !ok {
			return
		}
		st.mu.Lock()
		defer st.mu.Unlock()

		replay, found, err := st.lookupBatchResponse(req.BatchID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if found {
			writeJSON(w, http.StatusOK, json.RawMessage(replay))
			return
		}

		receivedAt := nowRFC3339()
		resp := syncResponse{
			BatchID:          req.BatchID,
			ServerReceivedAt: receivedAt,
			Results:          []recordResult{},
			StockUpdates:     []stockUpdate{},
		}
		stockIndex := map[string]int{}
		for _, rec := range req.Records {
			result, upd := processRecord(st, rec, receivedAt)
			resp.Results = append(resp.Results, result)
			if upd != nil {
				if i, seen := stockIndex[upd.MedicineID]; seen {
					resp.StockUpdates[i] = *upd
				} else {
					stockIndex[upd.MedicineID] = len(resp.StockUpdates)
					resp.StockUpdates = append(resp.StockUpdates, *upd)
				}
			}
		}

		body, err := json.Marshal(resp)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := st.saveBatchResponse(req.BatchID, req.DeviceID, receivedAt, string(body)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, json.RawMessage(body))
	}
}

func readSyncRequest(w http.ResponseWriter, r *http.Request) (syncRequest, bool) {
	var req syncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return req, false
	}
	switch {
	case req.DeviceID == "":
		writeError(w, http.StatusBadRequest, "missing field: device_id")
	case req.BatchID == "":
		writeError(w, http.StatusBadRequest, "missing field: batch_id")
	case req.Records == nil:
		writeError(w, http.StatusBadRequest, "missing field: records")
	default:
		return req, true
	}
	return req, false
}

func processRecord(st *store, rec syncRecordIn, receivedAt string) (recordResult, *stockUpdate) {
	if rec.ClientUUID == "" {
		return rejectedResult(rec.ClientUUID, "missing required field: client_uuid"), nil
	}
	if _, found, err := st.lookupRecord(rec.ClientUUID); err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	} else if found {
		return recordResult{ClientUUID: rec.ClientUUID, Status: statusDuplicate}, nil
	}
	if schemaVersionUnsupported(rec.SchemaVersion) {
		return rejectedResult(rec.ClientUUID, "unsupported schema_version"), nil
	}
	switch rec.Type {
	case typePatient:
		return processPatientRecord(st, rec, receivedAt)
	case typeVisit:
		return processVisitRecord(st, rec, receivedAt)
	case typeDispensing:
		return processDispensingRecord(st, rec, receivedAt)
	default:
		return rejectedResult(rec.ClientUUID, fmt.Sprintf("unknown type: %s", rec.Type)), nil
	}
}

func processPatientRecord(st *store, rec syncRecordIn, receivedAt string) (recordResult, *stockUpdate) {
	data := rec.Data
	for _, f := range requiredFieldsByType[typePatient] {
		if _, ok := dataString(data, f); !ok {
			return rejectedResult(rec.ClientUUID, "missing required field: "+f), nil
		}
	}
	p := Patient{
		ServerID:         newUUIDv4(),
		ClientUUID:       rec.ClientUUID,
		NIK:              str(data["nik"]),
		Name:             str(data["name"]),
		BirthDate:        str(data["birth_date"]),
		Gender:           str(data["gender"]),
		Address:          optionalString(data, "address"),
		ClinicID:         str(data["clinic_id"]),
		DeviceRecordedAt: rec.DeviceRecordedAt,
		ReceivedAt:       receivedAt,
	}
	existing, reason, err := findPossibleDuplicatePatient(st, p)
	if err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	result := recordResult{ClientUUID: rec.ClientUUID, Status: statusAccepted, ServerID: p.ServerID}
	tx, err := beginTx(st)
	if err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	defer tx.Rollback()
	if err := insertPatientTx(tx, p); err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	if err := insertRecordTx(tx, recordEntry{ClientUUID: rec.ClientUUID, Type: typePatient, ServerID: p.ServerID, ReceivedAt: receivedAt}); err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	if existing != nil {
		result.Status = statusFlaggedDupPat
		result.ExistingPatientID = existing.ServerID
		if err := addReviewItemTx(tx, ReviewItem{
			Type:              "possible_duplicate_patient",
			NewPatientID:      p.ServerID,
			ExistingPatientID: existing.ServerID,
			Reason:            reason,
			CreatedAt:         receivedAt,
		}); err != nil {
			return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
		}
	}
	if err := tx.Commit(); err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	return result, nil
}

func processVisitRecord(st *store, rec syncRecordIn, receivedAt string) (recordResult, *stockUpdate) {
	data := rec.Data
	for _, f := range requiredFieldsByType[typeVisit] {
		if _, ok := dataString(data, f); !ok {
			return rejectedResult(rec.ClientUUID, "missing required field: "+f), nil
		}
	}
	patientRef := str(data["patient_client_uuid"])
	patient, err := resolveRecordOf(st, patientRef, typePatient)
	if err != nil {
		return rejectedResult(rec.ClientUUID, err.Error()), nil
	}
	if patient == nil {
		return rejectedResult(rec.ClientUUID, "unknown patient_client_uuid: "+patientRef), nil
	}
	v := Visit{
		ServerID:         newUUIDv4(),
		ClientUUID:       rec.ClientUUID,
		PatientServerID:  patient.ServerID,
		ClinicID:         str(data["clinic_id"]),
		Polyclinic:       str(data["polyclinic"]),
		VisitDate:        str(data["visit_date"]),
		Diagnosis:        optionalString(data, "diagnosis"),
		Notes:            optionalString(data, "notes"),
		DeviceRecordedAt: rec.DeviceRecordedAt,
		ReceivedAt:       receivedAt,
	}
	tx, err := beginTx(st)
	if err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	defer tx.Rollback()
	if err := insertVisitTx(tx, v); err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	if err := insertRecordTx(tx, recordEntry{ClientUUID: rec.ClientUUID, Type: typeVisit, ServerID: v.ServerID, ReceivedAt: receivedAt}); err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	if err := tx.Commit(); err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	return recordResult{ClientUUID: rec.ClientUUID, Status: statusAccepted, ServerID: v.ServerID}, nil
}

func processDispensingRecord(st *store, rec syncRecordIn, receivedAt string) (recordResult, *stockUpdate) {
	data := rec.Data
	for _, f := range []string{"visit_client_uuid", "medicine_id"} {
		if _, ok := dataString(data, f); !ok {
			return rejectedResult(rec.ClientUUID, "missing required field: "+f), nil
		}
	}
	quantity, qtyErr := quantityValue(data)
	if qtyErr != "" {
		return rejectedResult(rec.ClientUUID, qtyErr), nil
	}
	visitRef := str(data["visit_client_uuid"])
	visit, err := resolveRecordOf(st, visitRef, typeVisit)
	if err != nil {
		return rejectedResult(rec.ClientUUID, err.Error()), nil
	}
	if visit == nil {
		return rejectedResult(rec.ClientUUID, "unknown visit_client_uuid: "+visitRef), nil
	}
	medicineID := str(data["medicine_id"])
	prevStock, exists, err := st.currentStock(medicineID)
	if err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	if !exists {
		return rejectedResult(rec.ClientUUID, "unknown medicine_id: "+medicineID), nil
	}
	newStock := prevStock - quantity
	d := Dispensing{
		ServerID:         newUUIDv4(),
		ClientUUID:       rec.ClientUUID,
		VisitServerID:    visit.ServerID,
		MedicineID:       medicineID,
		Quantity:         quantity,
		DeviceRecordedAt: rec.DeviceRecordedAt,
		ReceivedAt:       receivedAt,
	}
	tx, err := beginTx(st)
	if err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	defer tx.Rollback()
	if err := insertDispensingTx(tx, d); err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	if err := insertRecordTx(tx, recordEntry{ClientUUID: rec.ClientUUID, Type: typeDispensing, ServerID: d.ServerID, ReceivedAt: receivedAt}); err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	if err := addLedgerTx(tx, medicineID, -quantity, &rec.ClientUUID, receivedAt); err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	if newStock < 0 {
		if err := addReviewItemTx(tx, ReviewItem{
			Type:         "negative_stock",
			MedicineID:   medicineID,
			CurrentStock: newStock,
			CreatedAt:    receivedAt,
		}); err != nil {
			return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
		}
	}
	if err := tx.Commit(); err != nil {
		return rejectedResult(rec.ClientUUID, "storage error: "+err.Error()), nil
	}
	return recordResult{ClientUUID: rec.ClientUUID, Status: statusAccepted, ServerID: d.ServerID},
		&stockUpdate{MedicineID: medicineID, CurrentStock: newStock}
}

func resolveRecordOf(st *store, clientUUID, wantType string) (*recordEntry, error) {
	entry, found, err := st.lookupRecord(clientUUID)
	if err != nil {
		return nil, fmt.Errorf("storage error: %s", err.Error())
	}
	if !found || entry.Type != wantType {
		return nil, nil
	}
	return &entry, nil
}

func rejectedResult(clientUUID, reason string) recordResult {
	return recordResult{ClientUUID: clientUUID, Status: statusRejected, Reason: reason}
}

func schemaVersionUnsupported(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		return true
	}
	return v != 1
}

func dataString(data map[string]any, key string) (string, bool) {
	v, present := data[key]
	if !present {
		return "", false
	}
	s, isString := v.(string)
	if !isString || s == "" {
		return "", false
	}
	return s, true
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func optionalString(data map[string]any, key string) string {
	s, _ := dataString(data, key)
	return s
}

func quantityValue(data map[string]any) (int64, string) {
	v, present := data["quantity"]
	if !present {
		return 0, "missing required field: quantity"
	}
	f, isNumber := v.(float64)
	if !isNumber {
		return 0, "invalid quantity"
	}
	if f != math.Trunc(f) {
		return 0, "invalid quantity"
	}
	return int64(f), ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"detail": message})
}
