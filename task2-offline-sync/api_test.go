package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const resetSeedBody = `{"clinics":[
{"id":"PKM-001","name":"Puskesmas Sandik Raya","timezone":"Asia/Makassar"},
{"id":"PKM-002","name":"Puskesmas Gunungsari Timur","timezone":"Asia/Makassar"},
{"id":"PKM-003","name":"Puskesmas Kalimanah Wetan","timezone":"Asia/Jakarta"}],
"stock":[
{"medicine_id":"MED-001","name":"Paracetamol 500mg","quantity":5000},
{"medicine_id":"MED-002","name":"Amoxicillin 500mg","quantity":3000},
{"medicine_id":"MED-003","name":"ORS (Oralit)","quantity":20}]}`

type apiHarness struct {
	t   *testing.T
	st  *Store
	srv *httptest.Server
}

func newAPIHarness(t *testing.T) *apiHarness {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "api-test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	mux := http.NewServeMux()
	mux.HandleFunc("POST /sync", handleSync(st))
	mux.HandleFunc("GET /patients/{id}", handleGetPatient(st))
	mux.HandleFunc("GET /stock/{id}", handleGetStock(st))
	mux.HandleFunc("GET /review-queue", handleReviewQueue(st))
	mux.HandleFunc("POST /reset", handleReset(st))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &apiHarness{t: t, st: st, srv: srv}
}

func (h *apiHarness) do(method, path string, body []byte) (int, []byte) {
	h.t.Helper()
	var req *http.Request
	var err error
	if body == nil {
		req, err = http.NewRequest(method, h.srv.URL+path, nil)
	} else {
		req, err = http.NewRequest(method, h.srv.URL+path, bytes.NewReader(body))
	}
	if err != nil {
		h.t.Fatalf("build request %s %s: %v", method, path, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
}

func (h *apiHarness) get(path string) (int, []byte) { return h.do("GET", path, nil) }

func (h *apiHarness) postJSON(path string, body []byte) (int, []byte) {
	return h.do("POST", path, body)
}

func (h *apiHarness) mustReset() {
	h.t.Helper()
	code, raw := h.postJSON("/reset", []byte(resetSeedBody))
	if code != 200 {
		h.t.Fatalf("/reset: %d %s", code, raw)
	}
}

func (h *apiHarness) postPayload(name string) (int, []byte) {
	h.t.Helper()
	raw, err := os.ReadFile(filepath.Join("sample-payloads", name))
	if err != nil {
		h.t.Fatalf("read payload %s: %v", name, err)
	}
	return h.postJSON("/sync", raw)
}

func mustDecodeSync(t *testing.T, raw []byte) syncResponse {
	t.Helper()
	var out syncResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode syncResponse: %v: %s", err, raw)
	}
	return out
}

func resultsAsMaps(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	var out struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode results: %v: %s", err, raw)
	}
	return out.Results
}

func requireKeyAbsent(t *testing.T, m map[string]any, key string) {
	t.Helper()
	if _, present := m[key]; present {
		t.Fatalf("result %v must not contain %q", m, key)
	}
}

func requireKeyEquals(t *testing.T, m map[string]any, key string, want any) {
	t.Helper()
	got, present := m[key]
	if !present || got != want {
		t.Fatalf("result %v: %q = %v; want %v", m, key, got, want)
	}
}

func stockOf(t *testing.T, h *apiHarness, medicineID string) int64 {
	t.Helper()
	code, raw := h.get("/stock/" + medicineID)
	if code != 200 {
		t.Fatalf("GET /stock/%s: %d %s", medicineID, code, raw)
	}
	var s stockUpdate
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("decode stock: %v: %s", err, raw)
	}
	return s.CurrentStock
}

func TestResetSeedsKnownState(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	if got := stockOf(t, h, "MED-001"); got != 5000 {
		t.Fatalf("MED-001 = %d; want 5000", got)
	}
	if got := stockOf(t, h, "MED-003"); got != 20 {
		t.Fatalf("MED-003 = %d; want 20 (intentional for stock conflict)", got)
	}
	code, raw := h.get("/stock/MED-004")
	if code != 404 || !bytes.Equal(bytes.TrimSpace(raw), []byte(`{"detail":"Medicine not found"}`)) {
		t.Fatalf("unknown stock = %d %s; want 404 Medicine not found", code, raw)
	}
}

func TestNormalBatchLifecycle(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	code, raw := h.postPayload("01-normal-batch.json")
	if code != 200 {
		t.Fatalf("01: %d %s", code, raw)
	}
	resp := mustDecodeSync(t, raw)
	if len(resp.Results) != 6 {
		t.Fatalf("results = %d; want 6", len(resp.Results))
	}
	for i, r := range resp.Results {
		if r.Status != statusAccepted {
			t.Fatalf("result[%d] = %s (%s); want accepted", i, r.Status, r.Reason)
		}
		if r.ServerID == "" {
			t.Fatalf("result[%d] missing server_id", i)
		}
	}
	if len(resp.StockUpdates) != 2 ||
		resp.StockUpdates[0].MedicineID != "MED-001" || resp.StockUpdates[0].CurrentStock != 4990 ||
		resp.StockUpdates[1].MedicineID != "MED-002" || resp.StockUpdates[1].CurrentStock != 2985 {
		t.Fatalf("stock_updates = %+v", resp.StockUpdates)
	}
	if _, err := time.Parse(time.RFC3339, resp.ServerReceivedAt); err != nil {
		t.Fatalf("server_received_at %q not RFC3339: %v", resp.ServerReceivedAt, err)
	}

	code, raw = h.get("/patients/" + resp.Results[0].ServerID)
	if code != 200 {
		t.Fatalf("GET patient: %d %s", code, raw)
	}
	var p patientResponse
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("decode patient: %v", err)
	}
	if p.ID != resp.Results[0].ServerID || p.ClientUUID != "pat-a001-uuid" ||
		p.NIK != "5201141506900001" || p.Name != "Ahmad Fauzi" || p.BirthDate != "1990-06-15" ||
		p.Gender != "M" || p.ClinicID != "PKM-001" || p.ReceivedAt == "" {
		t.Fatalf("patient body = %+v", p)
	}

	code, raw = h.get("/patients/00000000-0000-0000-0000-000000000000")
	if code != 404 || !bytes.Equal(bytes.TrimSpace(raw), []byte(`{"detail":"Patient not found"}`)) {
		t.Fatalf("unknown patient = %d %s; want 404 Patient not found", code, raw)
	}

	code, raw = h.get("/review-queue")
	if code != 200 || !bytes.Equal(bytes.TrimSpace(raw), []byte(`{"items":[]}`)) {
		t.Fatalf("review queue = %d %s; want {\"items\":[]}", code, raw)
	}
}

func TestDuplicateBatchReplayIsByteIdentical(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	code1, body1 := h.postPayload("01-normal-batch.json")
	code2, body2 := h.postPayload("02-duplicate-batch.json")
	if code1 != 200 || code2 != 200 {
		t.Fatalf("codes = %d, %d", code1, code2)
	}
	if !bytes.Equal(body1, body2) {
		t.Fatalf("replay differs:\n1: %s\n2: %s", body1, body2)
	}
	resp := mustDecodeSync(t, body2)
	if resp.BatchID != "batch-normal-001" {
		t.Fatalf("batch_id = %s", resp.BatchID)
	}
	if got := stockOf(t, h, "MED-001"); got != 4990 {
		t.Fatalf("MED-001 after replay = %d; want 4990 (no double consume)", got)
	}
}

func TestFreshBatchResendReturnsDuplicates(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	code, raw := h.postPayload("01-normal-batch.json")
	if code != 200 {
		t.Fatalf("01: %d %s", code, raw)
	}
	original, err := os.ReadFile(filepath.Join("sample-payloads", "01-normal-batch.json"))
	if err != nil {
		t.Fatalf("read payload: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(original, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	payload["batch_id"] = "batch-resend-NEW-id"
	body, _ := json.Marshal(payload)

	code, raw = h.postJSON("/sync", body)
	if code != 200 {
		t.Fatalf("resend: %d %s", code, raw)
	}
	maps := resultsAsMaps(t, raw)
	if len(maps) != 6 {
		t.Fatalf("results = %d; want 6", len(maps))
	}
	for _, m := range maps {
		requireKeyEquals(t, m, "status", statusDuplicate)
		requireKeyAbsent(t, m, "server_id")
		requireKeyAbsent(t, m, "reason")
	}
	if got := stockOf(t, h, "MED-001"); got != 4990 {
		t.Fatalf("MED-001 after resend = %d; want 4990", got)
	}
}

func TestPartiallyInvalidBatch(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	code01, raw01 := h.postPayload("01-normal-batch.json")
	if code01 != 200 {
		t.Fatalf("01: %d %s", code01, raw01)
	}
	code, raw := h.postPayload("03-partial-invalid.json")
	if code != 200 {
		t.Fatalf("03: %d %s (contract: 200 even with rejected records)", code, raw)
	}
	maps := resultsAsMaps(t, raw)
	if len(maps) != 5 {
		t.Fatalf("results = %d; want 5", len(maps))
	}
	requireKeyEquals(t, maps[0], "status", statusAccepted)
	requireKeyEquals(t, maps[1], "status", statusRejected)
	requireKeyEquals(t, maps[1], "reason", "missing required field: name")
	requireKeyEquals(t, maps[2], "status", statusAccepted)
	requireKeyEquals(t, maps[3], "status", statusRejected)
	requireKeyEquals(t, maps[3], "reason", "unknown medicine_id: MED-UNKNOWN")
	requireKeyEquals(t, maps[4], "status", statusAccepted)
	requireKeyAbsent(t, maps[0], "reason")
	if _, present := maps[1]["server_id"]; present {
		t.Fatalf("rejected record must not have server_id: %v", maps[1])
	}
	if got := stockOf(t, h, "MED-001"); got != 4985 {
		t.Fatalf("MED-001 = %d; want 4985 (5000-10 from 01, -5 from valid dispensing in 03)", got)
	}
}

func TestPossibleDuplicatePatientFlagged(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	_, _ = h.postPayload("01-normal-batch.json")
	code, raw := h.postPayload("04-duplicate-patients.json")
	if code != 200 {
		t.Fatalf("04: %d %s", code, raw)
	}
	maps := resultsAsMaps(t, raw)
	if len(maps) != 4 {
		t.Fatalf("results = %d; want 4", len(maps))
	}
	requireKeyEquals(t, maps[0], "status", statusAccepted)
	requireKeyEquals(t, maps[1], "status", statusFlaggedDupPat)
	requireKeyEquals(t, maps[2], "status", statusAccepted)
	requireKeyEquals(t, maps[3], "status", statusAccepted)

	flaggedID, _ := maps[1]["server_id"].(string)
	existingID, _ := maps[1]["existing_patient_id"].(string)
	if flaggedID == "" || existingID == "" {
		t.Fatalf("flagged result missing ids: %v", maps[1])
	}
	if existingID != maps[0]["server_id"] {
		t.Fatalf("existing_patient_id = %s; want pat-b001 server_id %v", existingID, maps[0]["server_id"])
	}
	if _, present := maps[1]["reason"]; present {
		t.Fatalf("flagged result must not carry reason (reason lives in review queue): %v", maps[1])
	}

	code, raw = h.get("/patients/" + flaggedID)
	if code != 200 {
		t.Fatalf("flagged patient stored: %d %s", code, raw)
	}
	var p patientResponse
	json.Unmarshal(raw, &p)
	if p.Name != "SITI AMINAH" || p.ClinicID != "PKM-002" {
		t.Fatalf("flagged patient body = %+v", p)
	}

	code, raw = h.get("/review-queue")
	if code != 200 {
		t.Fatalf("review queue: %d %s", code, raw)
	}
	var rq reviewQueueResponse
	if err := json.Unmarshal(raw, &rq); err != nil {
		t.Fatalf("decode review queue: %v: %s", err, raw)
	}
	if len(rq.Items) != 1 {
		t.Fatalf("review items = %d; want 1", len(rq.Items))
	}
	item, ok := rq.Items[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected item type: %T", rq.Items[0])
	}
	if item["type"] != "possible_duplicate_patient" {
		t.Fatalf("item type = %v", item["type"])
	}
	if item["reason"] != reasonSimilarName {
		t.Fatalf("item reason = %v; want %q", item["reason"], reasonSimilarName)
	}
	if item["new_patient_id"] != flaggedID || item["existing_patient_id"] != existingID {
		t.Fatalf("item ids = %v + %v; want %s + %s", item["new_patient_id"], item["existing_patient_id"], flaggedID, existingID)
	}
	if _, err := time.Parse(time.RFC3339, item["created_at"].(string)); err != nil {
		t.Fatalf("item created_at not RFC3339: %v", err)
	}
	if _, present := item["medicine_id"]; present {
		t.Fatalf("patient-dup item must not carry medicine fields: %v", item)
	}
}

func TestStockConflictLastUnits(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	code, raw := h.postPayload("05-stock-conflict.json")
	if code != 200 {
		t.Fatalf("05: %d %s", code, raw)
	}
	maps := resultsAsMaps(t, raw)
	if len(maps) != 4 {
		t.Fatalf("results = %d; want 4", len(maps))
	}
	for _, m := range maps {
		requireKeyEquals(t, m, "status", statusAccepted)
	}
	resp := mustDecodeSync(t, raw)
	if got := stockOf(t, h, "MED-003"); got != -5 {
		t.Fatalf("MED-003 = %d; want -5 (20-15=5, then 5-10)", got)
	}
	if len(resp.StockUpdates) != 1 || resp.StockUpdates[0].MedicineID != "MED-003" || resp.StockUpdates[0].CurrentStock != -5 {
		t.Fatalf("stock_updates = %+v; want single MED-003 at -5", resp.StockUpdates)
	}
	code, raw = h.get("/review-queue")
	var rq reviewQueueResponse
	json.Unmarshal(raw, &rq)
	if code != 200 || len(rq.Items) != 1 {
		t.Fatalf("review queue = %d %s; want 1 item", code, raw)
	}
	item, ok := rq.Items[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected item type: %T", rq.Items[0])
	}
	if item["type"] != "negative_stock" || item["medicine_id"] != "MED-003" || item["current_stock"] != float64(-5) {
		t.Fatalf("negative_stock item = %v", item)
	}
	if _, err := time.Parse(time.RFC3339, item["created_at"].(string)); err != nil {
		t.Fatalf("item created_at not RFC3339: %v", err)
	}
	if _, present := item["new_patient_id"]; present {
		t.Fatalf("negative_stock item must not carry patient fields: %v", item)
	}
}

func TestUnsupportedSchemaVersion(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	code, raw := h.postPayload("07-old-schema-version.json")
	if code != 200 {
		t.Fatalf("07: %d %s", code, raw)
	}
	maps := resultsAsMaps(t, raw)
	if len(maps) != 2 {
		t.Fatalf("results = %d; want 2", len(maps))
	}
	for _, m := range maps {
		requireKeyEquals(t, m, "status", statusRejected)
		requireKeyEquals(t, m, "reason", "unsupported schema_version")
	}
	if got := stockOf(t, h, "MED-001"); got != 5000 {
		t.Fatalf("MED-001 = %d; want 5000 (rejected records must not touch stock)", got)
	}
}

func TestWrongClockAccepted(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	code, raw := h.postPayload("06-wrong-clock.json")
	if code != 200 {
		t.Fatalf("06: %d %s", code, raw)
	}
	resp := mustDecodeSync(t, raw)
	for _, r := range resp.Results {
		if r.Status != statusAccepted {
			t.Fatalf("result = %s (%s); device clock set to 2019 must not reject", r.Status, r.Reason)
		}
	}
	ts, err := time.Parse(time.RFC3339, resp.ServerReceivedAt)
	if err != nil {
		t.Fatalf("server_received_at unparsable: %v", err)
	}
	if ts.Year() == 2019 {
		t.Fatalf("server_received_at must use server clock, not device clock: %s", resp.ServerReceivedAt)
	}
}

func TestMalformedTopLevelRejected(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	cases := []struct {
		name string
		body string
	}{
		{"missing device_id", `{"batch_id":"b-1","records":[]}`},
		{"missing batch_id", `{"device_id":"d-1","records":[]}`},
		{"missing records", `{"device_id":"d-1","batch_id":"b-1"}`},
		{"records not an array", `{"device_id":"d-1","batch_id":"b-1","records":"nope"}`},
		{"invalid json", `{not json at all`},
	}
	for _, tc := range cases {
		code, raw := h.postJSON("/sync", []byte(tc.body))
		if code != 400 {
			t.Fatalf("%s: status = %d; want 400", tc.name, code)
		}
		trimmed := bytes.TrimSpace(raw)
		if !bytes.HasPrefix(trimmed, []byte(`{"detail":`)) {
			t.Fatalf("%s: body = %s; want {\"detail\":...}", tc.name, trimmed)
		}
	}
}

func TestReferenceValidation(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()

	buildBatch := func(batchID string, records ...string) []byte {
		var b bytes.Buffer
		b.WriteString(`{"device_id":"d","batch_id":"`)
		b.WriteString(batchID)
		b.WriteString(`","records":[`)
		b.WriteString(strings.Join(records, ","))
		b.WriteString(`]}`)
		return b.Bytes()
	}

	code, raw := h.postJSON("/sync", buildBatch("b-ref-1",
		`{"type":"visit","client_uuid":"vis-ghost-1","device_recorded_at":"2026-10-15T09:35:00","schema_version":1,
		  "data":{"patient_client_uuid":"pat-ghost","clinic_id":"PKM-001","polyclinic":"umum","visit_date":"2026-10-15T09:30:00"}}`))
	if code != 200 {
		t.Fatalf("ghost patient ref: %d %s", code, raw)
	}
	maps := resultsAsMaps(t, raw)
	requireKeyEquals(t, maps[0], "status", statusRejected)
	requireKeyEquals(t, maps[0], "reason", "unknown patient_client_uuid: pat-ghost")

	code, raw = h.postJSON("/sync", buildBatch("b-ref-2",
		`{"type":"dispensing","client_uuid":"disp-ghost-1","device_recorded_at":"2026-10-15T09:40:00","schema_version":1,
		  "data":{"visit_client_uuid":"vis-ghost","medicine_id":"MED-001","quantity":5}}`))
	if code != 200 {
		t.Fatalf("ghost visit ref: %d %s", code, raw)
	}
	maps = resultsAsMaps(t, raw)
	requireKeyEquals(t, maps[0], "status", statusRejected)
	requireKeyEquals(t, maps[0], "reason", "unknown visit_client_uuid: vis-ghost")

	code, raw = h.postJSON("/sync", buildBatch("b-ref-3",
		`{"type":"visit","client_uuid":"vis-real-1","device_recorded_at":"2026-10-15T09:35:00","schema_version":1,
		  "data":{"patient_client_uuid":"pat-zzz","clinic_id":"PKM-001","polyclinic":"umum","visit_date":"2026-10-15T09:30:00"}}`,
		`{"type":"visit","client_uuid":"vis-points-at-visit","device_recorded_at":"2026-10-15T09:36:00","schema_version":1,
		  "data":{"patient_client_uuid":"vis-real-1","clinic_id":"PKM-001","polyclinic":"umum","visit_date":"2026-10-15T09:31:00"}}`))
	if code != 200 {
		t.Fatalf("type mismatch ref: %d %s", code, raw)
	}
	maps = resultsAsMaps(t, raw)
	requireKeyEquals(t, maps[0], "status", statusRejected)
	requireKeyEquals(t, maps[0], "reason", "unknown patient_client_uuid: pat-zzz")
	requireKeyEquals(t, maps[1], "status", statusRejected)
	requireKeyEquals(t, maps[1], "reason", "unknown patient_client_uuid: vis-real-1")
	if got := stockOf(t, h, "MED-001"); got != 5000 {
		t.Fatalf("MED-001 = %d; want 5000 (no rejected record may touch stock)", got)
	}
}

func TestInBatchReferenceChain(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	code, raw := h.postJSON("/sync", []byte(`{"device_id":"d","batch_id":"b-chain","records":[
{"type":"patient_registration","client_uuid":"pat-c-1","device_recorded_at":"2026-10-15T08:00:00","schema_version":1,
 "data":{"nik":"5201141506900001","name":"Chain Patient","birth_date":"1990-06-15","gender":"M","clinic_id":"PKM-001"}},
{"type":"visit","client_uuid":"vis-c-1","device_recorded_at":"2026-10-15T08:10:00","schema_version":1,
 "data":{"patient_client_uuid":"pat-c-1","clinic_id":"PKM-001","polyclinic":"umum","visit_date":"2026-10-15T08:05:00"}},
{"type":"dispensing","client_uuid":"disp-c-1","device_recorded_at":"2026-10-15T08:20:00","schema_version":1,
 "data":{"visit_client_uuid":"vis-c-1","medicine_id":"MED-001","quantity":7}}]}`))
	if code != 200 {
		t.Fatalf("chain: %d %s", code, raw)
	}
	resp := mustDecodeSync(t, raw)
	for i, r := range resp.Results {
		if r.Status != statusAccepted {
			t.Fatalf("result[%d] = %s (%s); same-batch references must resolve", i, r.Status, r.Reason)
		}
	}
	if got := stockOf(t, h, "MED-001"); got != 4993 {
		t.Fatalf("MED-001 = %d; want 4993", got)
	}
	if len(resp.StockUpdates) != 1 || resp.StockUpdates[0].CurrentStock != 4993 {
		t.Fatalf("stock_updates = %+v", resp.StockUpdates)
	}
}

func TestEmptyRecordsBatchAcceptedAndReplayable(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	body := []byte(`{"device_id":"d","batch_id":"b-empty","records":[]}`)
	code1, raw1 := h.postJSON("/sync", body)
	code2, raw2 := h.postJSON("/sync", body)
	if code1 != 200 || code2 != 200 {
		t.Fatalf("codes = %d, %d; empty records array is present, not missing", code1, code2)
	}
	resp1 := mustDecodeSync(t, raw1)
	if len(resp1.Results) != 0 || len(resp1.StockUpdates) != 0 {
		t.Fatalf("empty batch response = %s", raw1)
	}
	if !bytes.Equal(raw1, raw2) {
		t.Fatalf("replay differs:\n1: %s\n2: %s", raw1, raw2)
	}
}

func TestResetWipesIdempotencyMemory(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	code, raw := h.postPayload("01-normal-batch.json")
	if code != 200 {
		t.Fatalf("01: %d %s", code, raw)
	}
	h.mustReset()
	if got := stockOf(t, h, "MED-001"); got != 5000 {
		t.Fatalf("MED-001 after re-reset = %d; want fresh 5000", got)
	}
	code, raw = h.postPayload("02-duplicate-batch.json")
	if code != 200 {
		t.Fatalf("replay after reset: %d %s", code, raw)
	}
	resp := mustDecodeSync(t, raw)
	for i, r := range resp.Results {
		if r.Status != statusAccepted {
			t.Fatalf("result[%d] = %s; batch replay memory must be wiped by /reset", i, r.Status)
		}
	}
	if got := stockOf(t, h, "MED-001"); got != 4990 {
		t.Fatalf("MED-001 = %d; want 4990 (fresh processing after reset)", got)
	}
}

func TestDuplicateBatchStatusShape(t *testing.T) {
	h := newAPIHarness(t)
	h.mustReset()
	record := `{"type":"patient_registration","client_uuid":"pat-shape-1","device_recorded_at":"2026-10-15T08:00:00","schema_version":1,
 "data":{"nik":"5201141506900001","name":"Shape Patient","birth_date":"1990-06-15","gender":"M","clinic_id":"PKM-001"}}`
	first := []byte(`{"device_id":"d","batch_id":"b-shape","records":[` + record + `]}`)
	resend := []byte(`{"device_id":"d","batch_id":"b-shape-2","records":[` + record + `]}`)

	code, raw := h.postJSON("/sync", first)
	if code != 200 {
		t.Fatalf("first send: %d %s", code, raw)
	}
	code, raw = h.postJSON("/sync", resend)
	if code != 200 {
		t.Fatalf("resend: %d %s", code, raw)
	}
	maps := resultsAsMaps(t, raw)
	if len(maps) != 1 {
		t.Fatalf("results = %d; want 1", len(maps))
	}
	requireKeyEquals(t, maps[0], "client_uuid", "pat-shape-1")
	requireKeyEquals(t, maps[0], "status", statusDuplicate)
	requireKeyAbsent(t, maps[0], "server_id")
	requireKeyAbsent(t, maps[0], "reason")
	requireKeyAbsent(t, maps[0], "existing_patient_id")
}
