// Package handler contains HTTP handlers for the app's API endpoints.
// It provides utilities for request binding, validation, and response rendering.
// Route handlers must not contain any business logic; this belongs in the service layer..
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/service"
	"github.com/ognev-dev/goplease/server/response"
	"go.opentelemetry.io/otel/trace"
)

var (
	// ErrIDParamMissingFromRequest is returned when an expected ID URL parameter is absent.
	ErrIDParamMissingFromRequest = app.ErrBadRequest("ID param missing from request")
	// ErrIDParamMustBeInt64 is returned when an ID URL parameter is present but cannot be parsed as a positive int64.
	ErrIDParamMustBeInt64 = app.ErrBadRequest("ID param must be positive int64")
)

// Fn is the function signature for a standard request handler.
type Fn func(w http.ResponseWriter, r *http.Request)

// Handler holds dependencies required by request handlers.
type Handler struct {
	service *service.Service
	tracer  trace.Tracer
}

// New creates and returns a new Handler instance.
func New(service *service.Service, t trace.Tracer) *Handler {
	return &Handler{
		service: service,
		tracer:  t,
	}
}

// ServerStatus is an HTTP handler that returns basic information about the running server.
func (h *Handler) ServerStatus(w http.ResponseWriter, r *http.Request) {
	_, span := h.tracer.Start(r.Context(), "serverStatus")
	defer span.End()

	conf := app.Config()
	jsonOK(w, response.ServerStatus{
		Env:     conf.App.Env,
		Version: conf.App.Version,
		Time:    time.Now(),
	})
}

// Request is a wrapper around the standard http.Request and http.ResponseWriter
// that provides convenience methods for handling the current request/response lifecycle.
type Request struct {
	Request  *http.Request
	Response http.ResponseWriter
	aborted  bool
}

// NewRequest creates and returns a new Request wrapper object.
func NewRequest(r *http.Request, w http.ResponseWriter) *Request {
	return &Request{
		Request:  r,
		Response: w,
	}
}

// handleJSON is a helper function that attempts to parse a JSON request body into the
// provided 'body' struct, performs validation/sanitization, and handles errors.
// It returns a Request object wrapper for subsequent actions.
func handleJSON(w http.ResponseWriter, r *http.Request, body any) *Request {
	h := NewRequest(r, w)

	err := bindJSON(r, body)
	// we'll get EOF error when body is empty,
	// proceed as usual if so
	// (valid requests can be without body)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	if err != nil {
		h.Abort(err)
		return h
	}

	if v, ok := body.(Sanitizer); ok {
		v.Sanitize()
	}

	if v, ok := body.(Validator); ok {
		err := app.NewInputError()
		v.Validate(&err)

		if err.Has() {
			h.Abort(err)
			return h
		}
	}

	return h
}

// handleAuthorizedJSON wraps handleJSON to include user authentication.
// It retrieves the user from the request context and aborts the request
// with a 401 Unauthorized status if no user is found.
func handleAuthorizedJSON(w http.ResponseWriter, r *http.Request, body any) (user *ds.User, req *Request) {
	req = handleJSON(w, r, body)
	if req.Aborted() {
		return
	}

	user = ds.UserFromContext(r.Context())
	if user == nil {
		req.AbortUnauthorized()
		return
	}

	return
}

// MapHeaders parses request headers and maps values to fields in the 'to' struct
// based on the struct's 'h' tags.
func (h *Request) MapHeaders(to any) {
	val := reflect.ValueOf(to).Elem()
	typ := val.Type()

	for i := range typ.NumField() {
		field := typ.Field(i)

		tag := field.Tag.Get("h")
		if tag == "" {
			continue
		}

		headerName, _, _ := strings.Cut(tag, ",")

		if value, ok := h.Request.Header[http.CanonicalHeaderKey(headerName)]; ok && len(value) > 0 {
			fieldVal := val.Field(i)
			switch fieldVal.Kind() {
			// TODO handle other types (when needed)
			case reflect.Int:
				intVal, err := strconv.Atoi(value[0])
				if err == nil {
					fieldVal.SetInt(int64(intVal))
				}
			case reflect.String:
				fieldVal.SetString(value[0])
			}
		}
	}
}

// AbortUnauthorized wraps Abort with 401 Unauthorized.
func (h *Request) AbortUnauthorized() {
	h.Abort(app.ErrUnauthorized())
}

// Abort flags the request as aborted and writes the provided error to the response.
func (h *Request) Abort(err error) {
	h.aborted = true
	Abort(h.Response, h.Request, err)
}

// Aborted returns true if the request lifecycle has been stopped due to an error.
func (h *Request) Aborted() bool {
	return h.aborted
}

// jsonOK writes a standard HTTP 200 OK JSON response with the provided body.
func (h *Request) jsonOK(body any) {
	jsonOK(h.Response, body)
}

// jsonCreated writes a standard HTTP 201 Created JSON response with the provided body.
func (h *Request) jsonCreated(body any) {
	jsonCreated(h.Response, body)
}

// jsonSuccess writes a standard HTTP 200 OK JSON response using the predefined
// success struct.
func (h *Request) jsonSuccess() {
	jsonSuccess(h.Response)
}

// bindJSON reads and decodes the request body as JSON into the provided object.
func bindJSON(r *http.Request, obj any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()

	err := decoder.Decode(obj)
	if err != nil {
		err = fmt.Errorf("decode JSON: %w", err)
	}

	return err
}

// bindJSON is a convenience method to bind a JSON body and handle errors within the Request lifecycle.
func (h *Request) bindJSON(body any) *Request {
	return handleJSON(h.Response, h.Request, body)
}

// writeJSON serializes the provided body into a JSON response, sets the Content-Type,
// and writes the given HTTP status code.
func writeJSON(w http.ResponseWriter, body any, status int) (err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err = json.NewEncoder(w).Encode(body)
	if err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	return nil
}

// jsonOK writes a standard HTTP 200 OK JSON response with the provided body.
func jsonOK(w http.ResponseWriter, body any) {
	err := writeJSON(w, body, http.StatusOK)
	if err != nil {
		log.Println(err)
	}
}

// jsonSuccess writes a standard HTTP 200 OK JSON response with the provided body.
func jsonSuccess(w http.ResponseWriter) {
	jsonOK(w, response.Success)
}

// jsonCreated writes a standard HTTP 201 Created JSON response with the provided body.
func jsonCreated(w http.ResponseWriter, body any) {
	err := writeJSON(w, body, http.StatusCreated)
	if err != nil {
		log.Println(err)
	}
}

// Validator is an interface for structs that can perform custom, multi-field validation.
type Validator interface {
	Validate(err *app.InputError)
}

// Sanitizer is an interface for structs that can clean or normalize their input data.
type Sanitizer interface {
	Sanitize()
}

func idFromPath(r *http.Request, paramNameOpt ...string) (ds.ID, error) { //nolint:unparam
	name := "id"
	if len(paramNameOpt) == 1 {
		name = paramNameOpt[0]
	}

	return ds.ParseID(r.PathValue(name))
}

// Abort serializes and writes an application error (app.Error or app.InputError)
// to the client, handling appropriate HTTP status codes and logging internal errors.
func Abort(w http.ResponseWriter, r *http.Request, err error) {
	resp := Error{
		Code: app.CodeInternal,
	}

	if appErr, ok := errors.AsType[app.Error](err); ok {
		resp.Code = appErr.Code
		resp.Error = appErr.Error()
	}
	if inputErr, ok := errors.AsType[app.InputError](err); ok {
		resp.Code = app.CodeUnprocessable
		resp.InputErrors = inputErr
	} else {
		resp.Error = err.Error()
	}

	log.Println(resp.Error)

	if resp.Code >= app.CodeInternal {
		log.Println(string(debug.Stack()))
	}

	err = writeJSON(w, resp, resp.Code)
	if err != nil {
		log.Println(err)
	}

	return
}

// Error is the structure used for serializing and returning structured JSON error responses
// to the client, categorized by code.
type Error struct {
	Code        int               `json:"code"`
	Error       string            `json:"error,omitempty"`
	Errors      []string          `json:"errors,omitempty"`
	InputErrors map[string]string `json:"input_errors,omitempty"`
}

// bindQuery binds URL query parameters to fields in the 'to' struct based on the struct's 'url' tags.
// 'to' must be a non-nil pointer to a struct.
func bindQuery(r *http.Request, to any) {
	if to == nil {
		return
	}

	v := reflect.ValueOf(to)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		panic("bindQuery: 'to' must be a non-nil pointer to a struct")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		panic("bindQuery: 'to' must point to a struct")
	}

	q := r.URL.Query()
	bindQueryToStruct(q, v)
}

// bindQueryToStruct recursively binds URL query parameters to the fields of v.
func bindQueryToStruct(q url.Values, v reflect.Value) {
	t := v.Type()

	for i := range t.NumField() {
		sf := t.Field(i)
		fv := v.Field(i)

		if sf.Anonymous && tryBindAnonymous(q, fv) {
			continue
		}

		tag := cleanURLTag(sf.Tag.Get("url"))
		if tag == "" {
			continue
		}

		vals := q[tag]
		if len(vals) == 0 || !fv.CanSet() {
			continue
		}

		setFieldFromStrings(fv, vals)
	}
}

// cleanURLTag normalizes a `url` struct tag by removing supported modifiers
// (currently only ",omitempty") and returning the pure query key.
func cleanURLTag(tag string) string {
	return strings.Replace(tag, ",omitempty", "", 1)
}

// tryBindAnonymous attempts to process an anonymous (embedded) field.
func tryBindAnonymous(q url.Values, fv reflect.Value) bool {
	switch fv.Kind() {
	case reflect.Struct:
		bindQueryToStruct(q, fv)
		return true

	case reflect.Ptr:
		if fv.Type().Elem().Kind() != reflect.Struct {
			return false
		}

		if fv.IsNil() {
			if !fv.CanSet() {
				return true // handled: cannot allocate, so skip
			}
			fv.Set(reflect.New(fv.Type().Elem()))
		}

		bindQueryToStruct(q, fv.Elem())
		return true

	default:
		return false
	}
}

// setFieldFromStrings assigns query values to a struct field.
func setFieldFromStrings(fv reflect.Value, vals []string) {
	switch fv.Kind() {
	case reflect.Slice:
		setSliceFromStrings(fv, vals)
	case reflect.Ptr:
		setPtrFromString(fv, vals[0])
	default:
		setScalarFromString(fv, vals[0])
	}
}

// setScalarFromString parses a single string value and assigns it to a
// non-pointer, non-slice field of a supported scalar kind
// (string, bool, int*, uint*, float*).
func setScalarFromString(fv reflect.Value, s string) {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(s)

	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err == nil {
			fv.SetBool(b)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			fv.SetInt(n)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err == nil {
			fv.SetUint(n)
		}

	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, fv.Type().Bits())
		if err == nil {
			fv.SetFloat(n)
		}
	}
}

// setPtrFromString parses a single string value and assigns it to a pointer field.
func setPtrFromString(fv reflect.Value, s string) {
	elemType := fv.Type().Elem()

	// Special-case *string to avoid reflect.New + SetString.
	if elemType.Kind() == reflect.String {
		fv.Set(reflect.ValueOf(&s))
		return
	}

	if !shouldSetPtr(elemType.Kind(), s, elemType) {
		return
	}

	ptr := reflect.New(elemType)
	setScalarFromString(ptr.Elem(), s)
	fv.Set(ptr)
}

// shouldSetPtr reports whether a pointer value of the given kind should be set
// based on whether the string can be successfully parsed into that type.
func shouldSetPtr(kind reflect.Kind, s string, t reflect.Type) bool {
	switch kind {
	case reflect.Bool:
		_, err := strconv.ParseBool(s)
		return err == nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		_, err := strconv.ParseInt(s, 10, 64)
		return err == nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		_, err := strconv.ParseUint(s, 10, 64)
		return err == nil

	case reflect.Float32, reflect.Float64:
		_, err := strconv.ParseFloat(s, t.Bits())
		return err == nil

	default:
		return false
	}
}

// setSliceFromStrings binds multiple query values to a slice field.
func setSliceFromStrings(fv reflect.Value, vals []string) {
	elemType := fv.Type().Elem()
	slice := reflect.MakeSlice(fv.Type(), len(vals), len(vals))

	for i, s := range vals {
		elem := slice.Index(i)

		switch elemType.Kind() {
		case reflect.String:
			elem.SetString(s)
		case reflect.Bool:
			b, err := strconv.ParseBool(s)
			if err == nil {
				elem.SetBool(b)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(s, 10, 64)
			if err == nil {
				elem.SetInt(n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := strconv.ParseUint(s, 10, 64)
			if err == nil {
				elem.SetUint(n)
			}
		case reflect.Float32, reflect.Float64:
			n, err := strconv.ParseFloat(s, elemType.Bits())
			if err == nil {
				elem.SetFloat(n)
			}
		}
	}

	fv.Set(slice)
}

const sessionCookieName = "session"

// setSessionCookie sets the user authentication token as an HTTP-only, secure, same-site cookie.
func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, NewSessionCookie(token))
}

// NewSessionCookie creates and returns a new session cookie.
func NewSessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   60 * 60 * app.Config().Session.DurationHours,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

// clearSessionCookie sets an expired cookie with the session name to effectively remove it from the client.
func clearSessionCookie(w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &cookie)
}

// GetSessionFromCookie retrieves the authentication token string from the request cookies.
func GetSessionFromCookie(r *http.Request) string {
	cookie, _ := r.Cookie(sessionCookieName)
	if cookie != nil {
		return cookie.Value
	}

	return ""
}

type ctxKey int

const ctxServeJSON ctxKey = iota

// SetServerJSON marks request context to indicate the response must be JSON.
func SetServerJSON(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxServeJSON, true))
}
