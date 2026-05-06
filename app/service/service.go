// Package service ...
package service

import (
	"fmt"
	"strings"

	z "github.com/Oudwins/zog"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/repo"
	"go.opentelemetry.io/otel/trace"
)

const (
	// UsernameMinLen defines the minimum allowed length, in characters, for a user's username.
	UsernameMinLen = 2

	// UsernameMaxLen defines the maximum allowed length, in characters, for a user's username.
	UsernameMaxLen = 30

	// UserPasswordMinLen defines the minimum allowed length, in characters, for a user's password.
	UserPasswordMinLen = 6
)

var (
	newPasswordInputRules = z.String().Min(UserPasswordMinLen,
		z.Message("Password must be at least 6 characters")).
		Required(z.Message("Password is required"))
	emailInputRules = z.String().Email().Required(z.Message("Email is required"))
)

// Service holds dependencies required for the application's business logic layer.
type Service struct {
	db     *repo.Repo
	tracer trace.Tracer
}

// New is a factory function that creates and returns a new Service instance.
func New(db *app.DB, t trace.Tracer) *Service {
	return &Service{
		db:     repo.New(db, t),
		tracer: t,
	}
}

// Validatable indicates that the struct can be validated.
type Validatable interface {
	Sanitize()
	Validate() error
}

// CreateRuler defines an interface for entities that provide
// validation rules for creation operations.
type CreateRuler interface {
	CreateRules() z.Shape
}

// UpdateRuler defines an interface for entities that provide
// validation rules for update operations.
type UpdateRuler interface {
	UpdateRules() z.Shape
}

// ValidateCreate validates the given entity using its specific
// creation rules.
func ValidateCreate(v CreateRuler) error {
	return validateInput(v.CreateRules(), v)
}

// ValidateUpdate validates the given entity using its specific
// update rules.
func ValidateUpdate(v UpdateRuler) error {
	return validateInput(v.UpdateRules(), v)
}

// Normalize prepares the input by sanitizing and validating it.
func Normalize(v Validatable) error {
	v.Sanitize()
	return v.Validate()
}

func validateInput(rules z.Shape, data any) (err error) {
	// Zod panics if struct is missing rules key
	// we don't want that
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
				return
			}

			err = fmt.Errorf("%v", r) //nolint:err113
			return
		}
	}()

	issues := z.Struct(rules).Validate(data)
	if len(issues) == 0 {
		return nil
	}

	ie := app.NewInputError()
	for _, issue := range issues {
		key := app.CamelCaseToSnakeCase(strings.Join(issue.Path, "."))
		ie.Add(key, issue.Message)
	}

	return ie
}
