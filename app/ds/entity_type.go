package ds

import "slices"

// EntityType defines the type of the entity content.
type EntityType string

// Supported entity types.
const (
	EntityTypeBook EntityType = "book"
	EntityTypePage EntityType = "page"
)

// EntityTypes lists all supported entity types.
var EntityTypes = []EntityType{
	EntityTypeBook,
	EntityTypePage,
}

// Valid reports whether the entity type is supported.
func (t EntityType) Valid() bool {
	return slices.Contains(EntityTypes, t)
}
