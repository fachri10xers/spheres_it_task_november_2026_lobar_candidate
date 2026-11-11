package main

import (
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T) *store {
	t.Helper()
	st, err := openStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openStore: %v", err)
	}
	t.Cleanup(func() { st.close() })
	return st
}

func ptrString(s string) *string { return &s }

func TestResetSeedsOpeningLedger(t *testing.T) {
	st := openTestStore(t)
	err := st.reset(
		[]Clinic{{ID: "PKM-001", Name: "Puskesmas Sandik Raya", Timezone: "Asia/Makassar"}},
		[]MedicineSeed{
			{ID: "MED-001", Name: "Paracetamol 500mg", Quantity: 5000},
			{ID: "MED-003", Name: "ORS (Oralit)", Quantity: 20},
		},
	)
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	got, ok, err := st.currentStock("MED-001")
	if err != nil || !ok || got != 5000 {
		t.Fatalf("MED-001 stock = %d, ok=%v, err=%v; want 5000, ok", got, ok, err)
	}
	got, ok, err = st.currentStock("MED-003")
	if err != nil || !ok || got != 20 {
		t.Fatalf("MED-003 stock = %d, ok=%v, err=%v; want 20, ok", got, ok, err)
	}
	_, ok, err = st.currentStock("MED-UNKNOWN")
	if err != nil || ok {
		t.Fatalf("unknown medicine ok=%v err=%v; want ok=false, nil error", ok, err)
	}
}

func TestLedgerAppendOnlyDerivation(t *testing.T) {
	st := openTestStore(t)
	if err := st.reset(nil, []MedicineSeed{{ID: "MED-003", Name: "ORS (Oralit)", Quantity: 20}}); err != nil {
		t.Fatalf("reset: %v", err)
	}

	tx, err := beginTx(st)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := addLedgerTx(tx, "MED-003", -15, ptrString("disp-c001"), nowRFC3339()); err != nil {
		t.Fatalf("ledger -15: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	got, _, _ := st.currentStock("MED-003")
	if got != 5 {
		t.Fatalf("stock after -15 = %d; want 5", got)
	}

	tx, err = beginTx(st)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := addLedgerTx(tx, "MED-003", -10, ptrString("disp-c002"), nowRFC3339()); err != nil {
		t.Fatalf("ledger -10: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	got, _, _ = st.currentStock("MED-003")
	if got != -5 {
		t.Fatalf("stock after -10 = %d; want -5", got)
	}
}

func TestLedgerRefUniqueGuardsDoubleApply(t *testing.T) {
	st := openTestStore(t)
	if err := st.reset(nil, []MedicineSeed{{ID: "MED-001", Name: "Paracetamol 500mg", Quantity: 5000}}); err != nil {
		t.Fatalf("reset: %v", err)
	}
	tx, err := beginTx(st)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := addLedgerTx(tx, "MED-001", -10, ptrString("disp-a001"), nowRFC3339()); err != nil {
		t.Fatalf("first ledger row: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	tx, err = beginTx(st)
	if err != nil {
		t.Fatalf("begin 2: %v", err)
	}
	if err := addLedgerTx(tx, "MED-001", -10, ptrString("disp-a001"), nowRFC3339()); err == nil {
		t.Fatalf("second ledger row with same ref must fail, got nil")
	} else {
		tx.Rollback()
	}
	got, _, _ := st.currentStock("MED-001")
	if got != 4990 {
		t.Fatalf("stock = %d; want 4990 (no double apply)", got)
	}
}

func TestRecordRegistryRoundTrip(t *testing.T) {
	st := openTestStore(t)
	_, found, err := st.lookupRecord("pat-a001")
	if err != nil || found {
		t.Fatalf("empty registry lookup found=%v err=%v; want false", found, err)
	}
	tx, err := beginTx(st)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	entry := recordEntry{ClientUUID: "pat-a001", Type: "patient_registration", ServerID: "srv-111", ReceivedAt: "2026-11-11T00:00:00Z"}
	if err := insertRecordTx(tx, entry); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	got, found, err := st.lookupRecord("pat-a001")
	if err != nil || !found {
		t.Fatalf("lookup found=%v err=%v; want true", found, err)
	}
	if got.Type != "patient_registration" || got.ServerID != "srv-111" || got.ReceivedAt != entry.ReceivedAt {
		t.Fatalf("entry = %+v", got)
	}
}

func TestBatchResponseReplayStorage(t *testing.T) {
	st := openTestStore(t)
	_, found, err := st.lookupBatchResponse("batch-x")
	if err != nil || found {
		t.Fatalf("missing batch found=%v err=%v; want false", found, err)
	}
	original := `{"batch_id":"batch-x","results":[]}`
	if err := st.saveBatchResponse("batch-x", "device-a", nowRFC3339(), original); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, found, err := st.lookupBatchResponse("batch-x")
	if err != nil || !found {
		t.Fatalf("replay lookup found=%v err=%v; want true", found, err)
	}
	if got != original {
		t.Fatalf("replay mismatch: got %s", got)
	}
}

func TestReviewItemRoundTrip(t *testing.T) {
	st := openTestStore(t)
	tx, err := beginTx(st)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	pat := ReviewItem{Type: "possible_duplicate_patient", NewPatientID: "srv-new", ExistingPatientID: "srv-old", Reason: "Similar name and birth date", CreatedAt: nowRFC3339()}
	stock := ReviewItem{Type: "negative_stock", MedicineID: "MED-003", CurrentStock: -5, CreatedAt: nowRFC3339()}
	if err := addReviewItemTx(tx, pat); err != nil {
		t.Fatalf("add pat item: %v", err)
	}
	if err := addReviewItemTx(tx, stock); err != nil {
		t.Fatalf("add stock item: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	items, err := st.listReviewItems()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d; want 2", len(items))
	}
	if items[0].Type != "possible_duplicate_patient" || items[0].NewPatientID != "srv-new" || items[0].ExistingPatientID != "srv-old" {
		t.Fatalf("item[0] = %+v", items[0])
	}
	if items[1].Type != "negative_stock" || items[1].MedicineID != "MED-003" || items[1].CurrentStock != -5 {
		t.Fatalf("item[1] = %+v", items[1])
	}
}

func TestPatientQueryHelpers(t *testing.T) {
	st := openTestStore(t)
	tx, err := beginTx(st)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	p1 := Patient{ServerID: "srv-1", ClientUUID: "pat-1", NIK: "0000000000000000", Name: "St. Aminah", BirthDate: "1990-06-15", Gender: "F", Address: "Dusun", ClinicID: "PKM-001", ReceivedAt: "2026-11-11T00:00:00Z"}
	p2 := Patient{ServerID: "srv-2", ClientUUID: "pat-2", NIK: "5201146106900005", Name: "SITI AMINAH", BirthDate: "1990-06-15", Gender: "F", Address: "Dusun", ClinicID: "PKM-002", ReceivedAt: "2026-11-11T00:00:01Z"}
	if err := insertPatientTx(tx, p1); err != nil {
		t.Fatalf("insert p1: %v", err)
	}
	if err := insertPatientTx(tx, p2); err != nil {
		t.Fatalf("insert p2: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	got, found, err := st.patientByServerID("srv-1")
	if err != nil || !found || got.Name != "St. Aminah" {
		t.Fatalf("patientByServerID: %+v found=%v err=%v", got, found, err)
	}
	_, found, err = st.patientByServerID("srv-none")
	if err != nil || found {
		t.Fatalf("missing patient found=%v err=%v; want false", found, err)
	}

	byNIK, err := st.patientsByNIK("5201146106900005")
	if err != nil || len(byNIK) != 1 || byNIK[0].ServerID != "srv-2" {
		t.Fatalf("patientsByNIK: %+v err=%v", byNIK, err)
	}
	byBirth, err := st.patientsByBirthDate("1990-06-15")
	if err != nil || len(byBirth) != 2 {
		t.Fatalf("patientsByBirthDate: %+v err=%v", byBirth, err)
	}
	if byBirth[0].ServerID != "srv-1" || byBirth[1].ServerID != "srv-2" {
		t.Fatalf("patientsByBirthDate not in arrival order: %+v", byBirth)
	}
}

func TestResetClearsAllState(t *testing.T) {
	st := openTestStore(t)
	if err := st.reset(nil, []MedicineSeed{{ID: "MED-001", Name: "Paracetamol 500mg", Quantity: 5000}}); err != nil {
		t.Fatalf("reset 1: %v", err)
	}
	if err := st.saveBatchResponse("batch-x", "device-a", nowRFC3339(), `{"batch_id":"batch-x"}`); err != nil {
		t.Fatalf("save: %v", err)
	}
	tx, err := beginTx(st)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := insertRecordTx(tx, recordEntry{ClientUUID: "pat-1", Type: "patient_registration", ServerID: "srv-1", ReceivedAt: nowRFC3339()}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := addReviewItemTx(tx, ReviewItem{Type: "negative_stock", MedicineID: "MED-001", CurrentStock: -1, CreatedAt: nowRFC3339()}); err != nil {
		t.Fatalf("review: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if err := st.reset(nil, []MedicineSeed{{ID: "MED-001", Name: "Paracetamol 500mg", Quantity: 5000}}); err != nil {
		t.Fatalf("reset 2: %v", err)
	}
	if _, found, _ := st.lookupRecord("pat-1"); found {
		t.Fatalf("registry survived reset")
	}
	if _, found, _ := st.lookupBatchResponse("batch-x"); found {
		t.Fatalf("batch response survived reset")
	}
	items, err := st.listReviewItems()
	if err != nil || len(items) != 0 {
		t.Fatalf("review items after reset: %d err=%v; want 0, nil", len(items), err)
	}
	got, ok, _ := st.currentStock("MED-001")
	if !ok || got != 5000 {
		t.Fatalf("stock after re-reset = %d ok=%v; want 5000 ok", got, ok)
	}
}

func TestNewUUIDv4Format(t *testing.T) {
	id := newUUIDv4()
	if len(id) != 36 {
		t.Fatalf("uuid length = %d; want 36", len(id))
	}
	for i, r := range []int{8, 13, 18, 23} {
		if id[r] != '-' {
			t.Fatalf("uuid %q missing dash at %d", id, i)
		}
	}
	if id[14] != '4' {
		t.Fatalf("uuid %q is not version 4", id)
	}
	if id[19] != '8' && id[19] != '9' && id[19] != 'a' && id[19] != 'b' {
		t.Fatalf("uuid %q has bad variant nibble at 19: %c", id, id[19])
	}
}

func TestNowRFC3339Format(t *testing.T) {
	ts := nowRFC3339()
	if len(ts) != 20 || ts[19] != 'Z' {
		t.Fatalf("timestamp %q is not RFC3339 Z-terminated", ts)
	}
}
