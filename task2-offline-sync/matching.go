package main

import (
	"strings"
	"unicode"
)

const maxNameDistance = 2

const (
	reasonExactNIK    = "Exact NIK match"
	reasonSimilarName = "Similar name and birth date"
)

func findPossibleDuplicatePatient(st *Store, patient Patient) (*Patient, string, error) {
	if !isNoNIK(patient.NIK) {
		byNIK, err := st.PatientsByNIK(patient.NIK)
		if err != nil {
			return nil, "", err
		}
		if len(byNIK) > 0 {
			return &byNIK[0], reasonExactNIK, nil
		}
	}
	sameBirthDate, err := st.PatientsByBirthDate(patient.BirthDate)
	if err != nil {
		return nil, "", err
	}
	incoming := normalizeName(patient.Name)
	for i := range sameBirthDate {
		if levenshtein(incoming, normalizeName(sameBirthDate[i].Name)) <= maxNameDistance {
			return &sameBirthDate[i], reasonSimilarName, nil
		}
	}
	return nil, "", nil
}

func isNoNIK(nik string) bool {
	if nik == "" {
		return true
	}
	for _, r := range nik {
		if r != '0' {
			return false
		}
	}
	return true
}

func normalizeName(name string) string {
	var b strings.Builder
	lastWasSeparator := true
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastWasSeparator = false
		case unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r):
			if !lastWasSeparator {
				b.WriteRune(' ')
				lastWasSeparator = true
			}
		}
	}
	return strings.TrimRight(b.String(), " ")
}

func levenshtein(a, b string) int {
	ar, br := []rune(a), []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}
