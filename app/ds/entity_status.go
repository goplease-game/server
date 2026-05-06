package ds

import "slices"

// EntityStatus defines the moderation state of an entity.
type EntityStatus string

const (
	// EntityStatusUnderReview means the entity is awaiting moderation.
	EntityStatusUnderReview EntityStatus = "review"

	// EntityStatusApproved means the entity is live and approved.
	EntityStatusApproved EntityStatus = "approved"

	// EntityStatusRejected means the entity failed moderation.
	EntityStatusRejected EntityStatus = "rejected"
)

// EntityStatuses defines the list of valid entity statuses.
var EntityStatuses = []EntityStatus{
	EntityStatusUnderReview,
	EntityStatusApproved,
	EntityStatusRejected,
}

// Valid reports whether the status value is one of the supported entity statuses.
func (v EntityStatus) Valid() bool {
	return slices.Contains(EntityStatuses, v)
}

// Is reports whether the status matches any of the provided values.
func (v EntityStatus) Is(v2 ...EntityStatus) bool {
	return slices.Contains(v2, v)
}

// Not reports whether the status does not match any of the provided values.
func (v EntityStatus) Not(v2 ...EntityStatus) bool {
	return !v.Is(v2...)
}
