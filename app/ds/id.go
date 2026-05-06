package ds

import (
	"database/sql/driver"
	"errors"
	"fmt"

	z "github.com/Oudwins/zog"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// ErrInvalidIDFormat indicates that a provided ID string is not a valid UUID format.
var ErrInvalidIDFormat = errors.New("invalid UUID format")

// ID is a domain-specific UUID (v7).
// Add `json:",omitzero"` to JSON payloads where ID is optional,
// or zero ID will happily serialize itself as
// "00000000-0000-0000-0000-000000000000".
type ID uuid.UUID //nolint:recvcheck

// NilID is an empty UUID, all zeros.
var NilID = ID(uuid.Nil)

// NewID generates a new UUID v7.
// It panics if the system clock is severely misconfigured.
func NewID() ID {
	return ID(uuid.Must(uuid.NewV7()))
}

// ParseID converts a string into a ds.ID.
// It supports standard UUID formats.
func ParseID(s string) (ID, error) {
	uid, err := uuid.Parse(s)
	if err != nil {
		return ID{}, fmt.Errorf("invalid UUID string: %w", err)
	}
	return ID(uid), nil
}

// IsNil ...
func (id ID) IsNil() bool {
	return uuid.UUID(id) == uuid.Nil
}

// String returns the standard UUID string representation.
func (id ID) String() string {
	return uuid.UUID(id).String()
}

// Value implements driver.Valuer to allow ID to be used in SQL queries.
func (id ID) Value() (driver.Value, error) {
	if id.IsNil() {
		return nil, nil
	}

	return uuid.UUID(id).Value()
}

// Scan implements sql.Scanner to allow reading UUID from the database into ID.
func (id *ID) Scan(src any) error {
	var u uuid.UUID
	err := u.Scan(src)
	if err != nil {
		return fmt.Errorf("ds.ID: scan failed: %w", err)
	}
	*id = ID(u)
	return nil
}

// MarshalJSON uses the built-in MarshalText from google/uuid.
func (id ID) MarshalJSON() ([]byte, error) {
	text, err := uuid.UUID(id).MarshalText()
	if err != nil {
		return nil, err
	}

	res := make([]byte, 0, len(text)+2) //nolint:mnd
	res = append(res, '"')
	res = append(res, text...)
	res = append(res, '"')
	return res, nil
}

// UnmarshalJSON decodes ID from JSON.
// It treats null and empty string ("") as zero (empty) ID.
func (id *ID) UnmarshalJSON(data []byte) error {
	// null → zero value
	if string(data) == "null" {
		*id = ID{}
		return nil
	}

	// must be JSON string
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return ErrInvalidIDFormat
	}

	// empty string → zero value
	if len(data) == 2 { //nolint:mnd
		*id = ID{}
		return nil
	}

	var u uuid.UUID
	err := u.UnmarshalText(data[1 : len(data)-1])
	if err != nil {
		return fmt.Errorf("ds.ID: unmarshal failed: %w", err)
	}

	*id = ID(u)
	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (id *ID) UnmarshalYAML(value *yaml.Node) error {
	u, err := uuid.Parse(value.Value)
	if err != nil {
		return fmt.Errorf("ds.ID: unmarshal failed: %w", err)
	}

	*id = ID(u)
	return nil
}

// IDInputRules ...
var IDInputRules = z.CustomFunc(func(val *ID, _ z.Ctx) bool {
	if val == nil || *val == NilID {
		return false
	}

	return true
}, z.Message("Invalid UUID"))
