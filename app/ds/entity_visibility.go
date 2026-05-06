package ds

import "slices"

// EntityVisibility controls how the entity is accessed in listings and search results.
type EntityVisibility string

const (

	// EntityVisibilityPublic means the entity is visible to everyone and indexed.
	EntityVisibilityPublic EntityVisibility = "public"

	// EntityVisibilityPrivate means the entity is restricted to the owner and collaborators.
	EntityVisibilityPrivate EntityVisibility = "private"

	// EntityVisibilityUnlisted means the entity is accessible only via direct link.
	EntityVisibilityUnlisted EntityVisibility = "unlisted"
)

// EntityVisibilities defines the list of all supported entity visibility values.
var EntityVisibilities = []EntityVisibility{
	EntityVisibilityPublic,
	EntityVisibilityPrivate,
	EntityVisibilityUnlisted,
}

// Valid reports whether the visibility value is one of the supported visibilities.
func (v EntityVisibility) Valid() bool {
	return slices.Contains(EntityVisibilities, v)
}

// Is reports whether the visibility matches any of the provided values.
func (v EntityVisibility) Is(v2 ...EntityVisibility) bool {
	return slices.Contains(v2, v)
}

// Not reports whether the visibility does not match any of the provided values.
func (v EntityVisibility) Not(v2 ...EntityVisibility) bool {
	return !v.Is(v2...)
}
