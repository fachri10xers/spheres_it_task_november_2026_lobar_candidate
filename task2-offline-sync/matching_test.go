package main

import (
	"testing"
)

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"St. Aminah":          "st aminah",
		"SITI AMINAH":         "siti aminah",
		"st.  aminah ":        "st aminah",
		"  Dewi   Lestari  ":  "dewi lestari",
		"Ahmad Fauzi, S.Pd":   "ahmad fauzi s pd",
		"Rahmat-Hidayat (II)": "rahmat hidayat ii",
		"":                    "",
	}
	for in, want := range cases {
		if got := normalizeName(in); got != want {
			t.Errorf("normalizeName(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"st aminah", "siti aminah", 2},
		{"siti aminah", "st aminah", 2},
		{"kitten", "sitting", 3},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"flaw", "lawn", 2},
		{"ahmad fauzi", "st aminah", 9},
	}
	for _, c := range cases {
		if got := levenshtein(c.a, c.b); got != c.want {
			t.Errorf("levenshtein(%q, %q) = %d; want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIsNoNIK(t *testing.T) {
	cases := map[string]bool{
		"":                 true,
		"0000000000000000": true,
		"0":                true,
		"5201141506900001": false,
		"0000000000000001": false,
	}
	for in, want := range cases {
		if got := isNoNIK(in); got != want {
			t.Errorf("isNoNIK(%q) = %v; want %v", in, got, want)
		}
	}
}

func seedPatient(t *testing.T, st *Store, p Patient) {
	t.Helper()
	tx, err := st.BeginTx()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := InsertPatientTx(tx, p); err != nil {
		t.Fatalf("seed patient: %v", err)
	}
	if err := InsertRecordTx(tx, RecordEntry{ClientUUID: p.ClientUUID, Type: typePatient, ServerID: p.ServerID, ReceivedAt: p.ReceivedAt}); err != nil {
		t.Fatalf("seed record: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestFindPossibleDuplicateExactNIKWins(t *testing.T) {
	st := openTestStore(t)
	seedPatient(t, st, Patient{ServerID: "srv-ahmad", ClientUUID: "pat-a001", NIK: "5201141506900001", Name: "Ahmad Fauzi", BirthDate: "1990-06-15", Gender: "M", ClinicID: "PKM-001", ReceivedAt: "2026-11-11T00:00:00Z"})

	incoming := Patient{ClientUUID: "pat-new", NIK: "5201141506900001", Name: "Budi Santoso", BirthDate: "1990-06-15", Gender: "M", ClinicID: "PKM-002"}
	got, reason, err := findPossibleDuplicatePatient(st, incoming)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil || reason != reasonExactNIK || got.ServerID != "srv-ahmad" {
		t.Fatalf("got=%+v reason=%q; want srv-ahmad with %q", got, reason, reasonExactNIK)
	}
}

func TestFindPossibleDuplicateAllZeroNIKNeverExact(t *testing.T) {
	st := openTestStore(t)
	seedPatient(t, st, Patient{ServerID: "srv-st", ClientUUID: "pat-b001", NIK: "0000000000000000", Name: "St. Aminah", BirthDate: "1990-06-15", Gender: "F", ClinicID: "PKM-001", ReceivedAt: "2026-11-11T00:00:00Z"})

	incoming := Patient{ClientUUID: "pat-b002", NIK: "0000000000000000", Name: "Rahmat Hidayat", BirthDate: "1975-11-03", Gender: "M", ClinicID: "PKM-001"}
	got, reason, err := findPossibleDuplicatePatient(st, incoming)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got != nil || reason != "" {
		t.Fatalf("no-NIK patient with different birth date must not match; got=%+v reason=%q", got, reason)
	}
}

func TestFindPossibleDuplicateFuzzyAcrossClinics(t *testing.T) {
	st := openTestStore(t)
	seedPatient(t, st, Patient{ServerID: "srv-ahmad", ClientUUID: "pat-a001", NIK: "5201141506900001", Name: "Ahmad Fauzi", BirthDate: "1990-06-15", Gender: "M", ClinicID: "PKM-001", ReceivedAt: "2026-11-11T00:00:00Z"})
	seedPatient(t, st, Patient{ServerID: "srv-st", ClientUUID: "pat-b001", NIK: "0000000000000000", Name: "St. Aminah", BirthDate: "1990-06-15", Gender: "F", ClinicID: "PKM-001", ReceivedAt: "2026-11-11T00:00:01Z"})

	incoming := Patient{ClientUUID: "pat-b002", NIK: "5201146106900005", Name: "SITI AMINAH", BirthDate: "1990-06-15", Gender: "F", ClinicID: "PKM-002"}
	got, reason, err := findPossibleDuplicatePatient(st, incoming)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil || reason != reasonSimilarName || got.ServerID != "srv-st" {
		t.Fatalf("got=%+v reason=%q; want srv-st with %q", got, reason, reasonSimilarName)
	}
}

func TestFindPossibleDuplicateDistanceBoundary(t *testing.T) {
	st := openTestStore(t)
	seedPatient(t, st, Patient{ServerID: "srv-1", ClientUUID: "pat-1", NIK: "0000000000000000", Name: "Dewi Lestari", BirthDate: "1992-05-25", Gender: "F", ClinicID: "PKM-003", ReceivedAt: "2026-11-11T00:00:00Z"})

	withinThreshold := Patient{ClientUUID: "pat-in", NIK: "0000000000000000", Name: "Dewi Lestarina", BirthDate: "1992-05-25", Gender: "F", ClinicID: "PKM-003"}
	got, reason, err := findPossibleDuplicatePatient(st, withinThreshold)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil || reason != reasonSimilarName {
		t.Fatalf("distance-2 case must flag; got=%+v reason=%q", got, reason)
	}

	beyondThreshold := Patient{ClientUUID: "pat-out", NIK: "0000000000000000", Name: "Dewi Lestarinah", BirthDate: "1992-05-25", Gender: "F", ClinicID: "PKM-003"}
	got, reason, err = findPossibleDuplicatePatient(st, beyondThreshold)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got != nil || reason != "" {
		t.Fatalf("distance-3 case must not flag; got=%+v reason=%q", got, reason)
	}
}
