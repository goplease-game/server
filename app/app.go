// Package app ...
package app

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	z "github.com/Oudwins/zog"
	"github.com/gosimple/slug"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/microcosm-cc/bluemonday"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

const (
	// DevEnv is used for the local development environment.
	DevEnv = "dev"

	// TestEnv is used for running automated tests and quality assurance (QA) checks.
	TestEnv = "test"

	// StagingEnv is a production-like environment used for final testing before a full public release.
	StagingEnv = "staging"

	// ProductionEnv refers to the final production environment serving live users.
	ProductionEnv = "production"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

var mdRenderer = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
	),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
)

var htmlPolicy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.RequireNoFollowOnLinks(true)
	p.RequireNoReferrerOnLinks(true)

	return p
}()

var (
	// ErrInvalidJWT is returned when an authentication token is malformed,
	// invalidly signed, or contains unexpected claims.
	ErrInvalidJWT = ErrForbidden("invalid token")

	// ErrNotString indicates that a value is not of string type.
	ErrNotString = ErrForbidden("is not string")

	// ErrExpectedStringSliceOrAny indicates that a value is not a slice of strings or a slice of any type.
	ErrExpectedStringSliceOrAny = ErrForbidden("expected []string or []any")

	// ErrUniqueViolation indicates a violation of a uniqueness constraint.
	ErrUniqueViolation = errors.New("UNIQUE VIOLATION")
)

// serverURL holds parsed conf.Server.Addr.
// It is initialized during application startup and must not be modified.
var serverURL *url.URL

// CamelCaseToSnakeCase converts a string from CamelCase to snake_case.
func CamelCaseToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")

	return strings.ToLower(snake)
}

// RelativeFilePath computes the relative path of a full path with respect to a base path,
// and converts the path to use forward slashes.
func RelativeFilePath(basePath, fullPath string) string {
	rel, err := filepath.Rel(basePath, fullPath)
	if err != nil {
		return fullPath
	}

	rel = filepath.ToSlash(rel)

	return rel
}

// Validate executes the validation rules defined in the provided 'schema' against the 'data' struct.
// It converts any validation issues into an InputError type for structured error handling.
func Validate(schema z.Shape, data any) (err error) {
	// Zod panics if struct is missing schema key
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

	issues := z.Struct(schema).Validate(data)
	if len(issues) == 0 {
		return nil
	}

	ie := NewInputError()
	for _, issue := range issues {

		ie.Add(CamelCaseToSnakeCase(strings.Join(issue.Path, ".")), issue.Message)
	}

	return ie
}

// String converts the input value into a string representation.
func String(v any) string {
	if v == nil {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}

	if err, ok := v.(error); ok {
		return err.Error()
	}

	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}

	return fmt.Sprintf("%v", v)
}

// Token creates a cryptographically secure random token.
func Token(lengthOpt ...int) (string, error) {
	length := 32
	if len(lengthOpt) > 0 {
		length = lengthOpt[0]
	}

	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Slug converts any string into a URL-friendly format.
func Slug(s string) string {
	return slug.Make(s)
}

// MarkdownToHTML renders Markdown input into safe HTML.
func MarkdownToHTML(in string) (out string, err error) {
	var buf bytes.Buffer

	err = mdRenderer.Convert([]byte(in), &buf)
	if err != nil {
		return
	}

	out = htmlPolicy.Sanitize(buf.String())
	return
}

// HumanTime formats a timestamp into a human-readable relative time string.
//
// The function compares the given time with the current time and returns
// a concise, user-friendly representation (e.g. "just now", "yesterday, 15:04").
// An optional reference time may be provided for deterministic output.
func HumanTime(t time.Time, relOpt ...time.Time) string {
	now := time.Now()
	if len(relOpt) > 0 {
		now = relOpt[0]
	}
	d := now.Sub(t)

	switch {
	case d < 30*time.Second:
		return "just now"

	case d < 3*time.Minute:
		return "few minutes ago"

	case d < 10*time.Minute:
		return "five minutes ago"
	}

	ty, tm, td := t.Date()
	ny, nm, nd := now.Date()

	if ty == ny && tm == nm && td == nd {
		return t.Format("15:04")
	}

	yesterday := now.AddDate(0, 0, -1)
	yy, ym, yd := yesterday.Date()
	if ty == yy && tm == ym && td == yd {
		return "yesterday, " + t.Format("15:04")
	}

	if ty == ny {
		return t.Format("Jan 2, 15:04")
	}

	return t.Format("Jan 2, 2006; 15:04")
}

// IsNil checks if a value is nil.
func IsNil(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr,
		reflect.Map,
		reflect.Slice,
		reflect.Interface:
		return rv.IsNil()
	}

	return false
}

// ServerURL constructs an absolute URL by joining a relative path with the configured server URL.
func ServerURL(rel string) string {
	abs := *serverURL
	abs.Path = path.Join(abs.Path, rel)

	return abs.String()
}

// StringSliceFromAny converts a dynamically typed value into a []string.
func StringSliceFromAny(v any) (out []string, err error) {
	if v == nil {
		return
	}

	if ss, ok := v.([]string); ok {
		return ss, nil
	}

	raw, ok := v.([]any)
	if !ok {
		err = fmt.Errorf("%w, got %T", ErrExpectedStringSliceOrAny, v)
		return
	}

	out = make([]string, len(raw))
	for i, x := range raw {
		s, ok := x.(string)
		if !ok {
			err = fmt.Errorf("element[%d] %w, but %T", i, ErrNotString, x)
			return
		}
		out[i] = s
	}

	return out, nil
}

// MakePatch generates a patch string representing the differences between text1 and text2.
func MakePatch(text1, text2 string) string {
	dmp := diffmatchpatch.New()
	patches := dmp.PatchMake(text1, text2)

	return dmp.PatchToText(patches)
}

// ApplyPatch applies a patch string to the given text and returns the result.
func ApplyPatch(text, patch string) (result string, err error) {
	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(patch)
	if err != nil {
		err = fmt.Errorf("applying patch: %w", err)
		return
	}

	result, _ = dmp.PatchApply(patches, text)
	return
}

// IsUniqueViolation checks if an error is a PostgreSQL unique constraint violation.
// It returns the column name that caused the violation and a boolean indicating success.
// If the column name is not directly available in the error, it attempts to parse it
// from the error detail message (e.g., "Key (username)=(test) already exists.").
// For composite keys, it returns only the first column name
// (e.g., "public_id" from "Key (public_id, type)=(things-fall-apart, book) already exists.").
func IsUniqueViolation(err error) (column string, ok bool) {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return
	}

	// 23505 = unique_violation
	if pgErr.Code != "23505" {
		return
	}

	if pgErr.ColumnName == "" {
		s := pgErr.Detail // "Key (username)=(test) already exists."
		start := strings.Index(s, "(")
		if start == -1 {
			return
		}

		end := strings.Index(s[start+1:], ")")
		if end == -1 {
			return
		}

		column = s[start+1 : start+1+end] // username or "public_id, type"

		// For composite keys, take only the first column name
		if commaIdx := strings.Index(column, ","); commaIdx != -1 {
			column = column[:commaIdx]
		}

		column = strings.TrimSpace(column)
		return column, true
	}

	return pgErr.ColumnName, true
}

// Value dereferences a pointer and returns the value, or zero value if nil.
func Value[T any](p *T) (v T) {
	if p == nil {
		return
	}

	return *p
}
