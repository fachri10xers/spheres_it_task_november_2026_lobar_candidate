package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type patientResponse struct {
	ID         string `json:"id"`
	ClientUUID string `json:"client_uuid"`
	NIK        string `json:"nik"`
	Name       string `json:"name"`
	BirthDate  string `json:"birth_date"`
	Gender     string `json:"gender"`
	Address    string `json:"address"`
	ClinicID   string `json:"clinic_id"`
	ReceivedAt string `json:"received_at"`
}

type patientDupItem struct {
	Type              string `json:"type"`
	NewPatientID      string `json:"new_patient_id"`
	ExistingPatientID string `json:"existing_patient_id"`
	Reason            string `json:"reason"`
	CreatedAt         string `json:"created_at"`
}

type negativeStockItem struct {
	Type         string `json:"type"`
	MedicineID   string `json:"medicine_id"`
	CurrentStock int64  `json:"current_stock"`
	CreatedAt    string `json:"created_at"`
}

type reviewQueueResponse struct {
	Items []any `json:"items"`
}

type resetResponse struct {
	Status    string `json:"status"`
	Clinics   int    `json:"clinics"`
	Medicines int    `json:"medicines"`
}

func handleGetPatient(st *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, found, err := st.PatientByServerID(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !found {
			writeError(w, http.StatusNotFound, "Patient not found")
			return
		}
		writeJSON(w, http.StatusOK, patientResponse{
			ID:         p.ServerID,
			ClientUUID: p.ClientUUID,
			NIK:        p.NIK,
			Name:       p.Name,
			BirthDate:  p.BirthDate,
			Gender:     p.Gender,
			Address:    p.Address,
			ClinicID:   p.ClinicID,
			ReceivedAt: p.ReceivedAt,
		})
	}
}

func handleGetStock(st *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		medicineID := r.PathValue("id")
		currentStock, exists, err := st.CurrentStock(medicineID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !exists {
			writeError(w, http.StatusNotFound, "Medicine not found")
			return
		}
		writeJSON(w, http.StatusOK, stockUpdate{MedicineID: medicineID, CurrentStock: currentStock})
	}
}

func handleReviewQueue(st *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := st.ListReviewItems()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out := make([]any, 0, len(items))
		for _, it := range items {
			switch it.Type {
			case "possible_duplicate_patient":
				out = append(out, patientDupItem{
					Type:              it.Type,
					NewPatientID:      it.NewPatientID,
					ExistingPatientID: it.ExistingPatientID,
					Reason:            it.Reason,
					CreatedAt:         it.CreatedAt,
				})
			case "negative_stock":
				out = append(out, negativeStockItem{
					Type:         it.Type,
					MedicineID:   it.MedicineID,
					CurrentStock: it.CurrentStock,
					CreatedAt:    it.CreatedAt,
				})
			}
		}
		writeJSON(w, http.StatusOK, reviewQueueResponse{Items: out})
	}
}

func handleReset(st *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Clinics []Clinic       `json:"clinics"`
			Stock   []MedicineSeed `json:"stock"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		st.WithWriteLock(func() {
			if err := st.Reset(body.Clinics, body.Stock); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			log.Printf("reset: wiped prior state, seeded %d clinics, %d medicines", len(body.Clinics), len(body.Stock))
			writeJSON(w, http.StatusOK, resetResponse{
				Status:    "reset",
				Clinics:   len(body.Clinics),
				Medicines: len(body.Stock),
			})
		})
	}
}
