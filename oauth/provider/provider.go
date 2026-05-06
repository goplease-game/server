// Package provider manages OAuth identity provider types and their validation.
package provider

import z "github.com/Oudwins/zog"

// Type defines the supported OAuth provider strings.
type Type string

// Supported OAuth providers.
const (
	Google Type = "google"
	GitHub Type = "github"
)

// Types is a collection of all valid OAuth provider types.
var Types = []Type{
	Google,
	GitHub,
}

// InvalidType is the fallback string returned when a provider is unrecognized.
var InvalidType = "invalid_oauth_provider"

var nameByType = map[Type]string{
	Google: "Google",
	GitHub: "GitHub",
}

// TypeInputRules defines the zog validation logic for provider types.
var TypeInputRules = z.CustomFunc(func(val *Type, _ z.Ctx) bool {
	if val == nil || !val.Valid() {
		return false
	}

	return true
}, z.Message("Invalid provider type"))

// String converts the provider type to its raw string representation.
func (p Type) String() string {
	return string(p)
}

// Name returns the capitalized display name of the provider.
func (p Type) Name() string {
	v, ok := nameByType[p]
	if ok {
		return v
	}

	return InvalidType
}

// Valid checks if the type exists within the allowed provider map.
func (p Type) Valid() bool {
	_, ok := nameByType[p]
	return ok
}

// New creates a new Type instance from a string.
func New(s string) Type {
	return Type(s)
}
