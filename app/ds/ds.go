// Package ds (Data Structure)
// All data models belonging to the app are stored here.
// Global functions should not be created here.
// Methods on types are welcome.
package ds

import (
	"time"
)

type ctxKey string

const (
	// PerPageNoLimit disables pagination limits and returns all records.
	PerPageNoLimit = -1

	// PerPageDefault defines the default number of records per page
	// when no explicit per-page value is provided.
	PerPageDefault = 25

	// PerPageMax defines the maximum allowed number of records per page
	// to prevent excessive result sizes.
	PerPageMax = 100
)

// FilterDT ...
type FilterDT struct {
	DT   *time.Time
	From *time.Time
	To   *time.Time
}

// DtAt ...
func DtAt(t time.Time) *FilterDT {
	return &FilterDT{
		DT: &t,
	}
}

// DtBefore ...
func DtBefore(t time.Time) *FilterDT {
	return &FilterDT{
		To: &t,
	}
}

// DtAfter ...
func DtAfter(t time.Time) *FilterDT {
	return &FilterDT{
		From: &t,
	}
}

// DtBetween ...
func DtBetween(from, to time.Time) *FilterDT {
	return &FilterDT{
		From: &from,
		To:   &to,
	}
}

// FilterString defines options for filtering string-type data.
type FilterString struct {
	Null          *bool
	Empty         *bool
	Equal         *string
	Contains      *string
	StartsWith    *string
	EndsWith      *string
	Not           bool
	CaseSensitive bool
}
