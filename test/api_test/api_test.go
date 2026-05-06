package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/google/go-querystring/query"
	aur "github.com/logrusorgru/aurora"
	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/ds"
	"github.com/ognev-dev/goplease/app/session"
	"github.com/ognev-dev/goplease/server"
	"github.com/ognev-dev/goplease/server/handler"
	"github.com/ognev-dev/goplease/test"
)

type Headers map[string]string

type RequestArgs struct {
	method       string
	path         string
	body         any
	bodyReader   io.Reader
	authToken    string
	headers      Headers
	bindResponse any
	assertStatus int
}

var (
	authUser  *ds.User
	authToken string
	router    http.Handler
	tt        *test.App
)

const ContentTypeJSON = "application/json"

func TestMain(m *testing.M) {
	tt = test.NewApp()

	router = server.New(tt.Service, tt.Tracer).Handler
	code := m.Run()

	tt.Shutdown()
	os.Exit(code)
}

func makeRequest(t *testing.T, r RequestArgs) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequestWithContext(
		context.Background(),
		r.method,
		r.path,
		r.bodyReader,
	)
	test.CheckErr(t, err)

	req.Header.Set("Content-Type", ContentTypeJSON)
	req.Header.Set("Accept", ContentTypeJSON)

	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	token := authToken
	if r.authToken != "" {
		token = r.authToken
	}

	if token != "" {
		req.AddCookie(handler.NewSessionCookie(token))
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

// Request sends given request,
// asserts that status is correct
// binds to response and returns httptest.ResponseRecorder (if you need more asserts).
func Request(t *testing.T, req RequestArgs) *httptest.ResponseRecorder {
	t.Helper()

	var (
		body []byte
		err  error
	)

	req.path = path.Join("/", tt.Conf.Server.APIBasePath, req.path)

	if !strings.HasSuffix(req.path, "/") && !strings.Contains(req.path, "/?") {
		req.path += "/"
	}

	if len(req.headers) == 0 {
		req.headers = Headers{}
	}

	if req.body != nil {
		body, err = json.MarshalIndent(req.body, "", "    ")
		test.CheckErr(t, err)
	}

	req.bodyReader = bytes.NewReader(body)

	resp := makeRequest(t, req)

	return handleResponse(t, req, resp, body)
}

func responseContentType(resp *httptest.ResponseRecorder) string {
	ct, ok := resp.Header()["Content-Type"]
	if !ok || len(ct) == 0 {
		return ""
	}

	frags := strings.Split(ct[0], ";")
	return frags[0]
}

// CREATE makes "create" POST request that expects 201.
func CREATE(t *testing.T, path string, body, response any) *httptest.ResponseRecorder {
	t.Helper()

	req := RequestArgs{
		path:         path,
		body:         body,
		bindResponse: response,
		assertStatus: http.StatusCreated,
		method:       http.MethodPost,
	}

	return Request(t, req)
}

// POST makes POST request that expects 200 by default.
func POST(t *testing.T, path string, body, response any, assertStatusOpt ...int) *httptest.ResponseRecorder {
	t.Helper()

	assertStatus := http.StatusOK
	if len(assertStatusOpt) == 1 {
		assertStatus = assertStatusOpt[0]
	}

	req := RequestArgs{
		path:         path,
		body:         body,
		bindResponse: response,
		assertStatus: assertStatus,
		method:       http.MethodPost,
	}

	return Request(t, req)
}

// UPDATE makes "PUT" request.
func UPDATE(t *testing.T, path string, body, response any) *httptest.ResponseRecorder {
	t.Helper()

	req := RequestArgs{
		path:         path,
		body:         body,
		bindResponse: response,
		assertStatus: http.StatusOK,
		method:       http.MethodPut,
	}

	return Request(t, req)
}

// DELETE makes "delete" request with expected defaults.
func DELETE(t *testing.T, path string, response any) *httptest.ResponseRecorder {
	t.Helper()
	req := RequestArgs{
		path:         path,
		bindResponse: response,
		assertStatus: http.StatusOK,
		method:       http.MethodDelete,
	}

	return Request(t, req)
}

// GET makes simple "get" request that should return "response" and 200.
// path can be string or Query.
func GET(t *testing.T, path any, response any) *httptest.ResponseRecorder {
	t.Helper()

	var url string
	switch v := path.(type) {
	case string:
		url = v
	case Query:
		url = v.String(t)
	default:
		t.Fatalf("unknown path type: %T", v)
	}

	req := RequestArgs{
		path:         url,
		bindResponse: response,
		assertStatus: http.StatusOK,
		method:       http.MethodGet,
	}

	return Request(t, req)
}

func login(t *testing.T) *ds.User {
	t.Helper()

	app.Config().Admins = []string{}

	if authUser != nil && authToken != "" {
		return authUser
	}

	user := create(t, ds.User{
		EmailConfirmed: true,
	})
	loginAs(t, user)

	return authUser
}

func loginAs(t *testing.T, u *ds.User) (token string) {
	t.Helper()

	s, err := tt.Service.CreateUserSession(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}

	token, err = session.NewSignedJWT(s.ID, u.ID)
	if err != nil {
		t.Fatal(err)
	}

	authToken = token
	authUser = u

	return token
}

func loginAsAdmin(t *testing.T) (user *ds.User) {
	t.Helper()

	user = create(t, ds.User{
		EmailConfirmed: true,
	})
	loginAs(t, user)

	app.Config().Admins = []string{user.ID.String()}
	return
}

type fileForm struct {
	authToken    string
	fields       map[string]string
	fileField    string
	purpose      ds.FilePurpose
	filename     string
	contentType  string
	file         io.Reader
	assertStatus int
}

// UploadFile sends a multipart/form-data request containing a file.
// Since we currently have only one file upload endpoint,
// most required params are hardcoded.
func UploadFile(t *testing.T, form fileForm) *ds.File {
	t.Helper()

	var fileResponse ds.File
	r := RequestArgs{
		authToken:    form.authToken,
		method:       http.MethodPost,
		path:         "/api/files/",
		bindResponse: &fileResponse,
		assertStatus: form.assertStatus,
	}

	if r.assertStatus == 0 {
		r.assertStatus = http.StatusCreated
	}

	if form.fileField == "" {
		form.fileField = "file"
	}
	if form.filename == "" {
		form.filename = "lets-test-this-out.dev"
	}
	if form.contentType == "" {
		form.contentType = "application/octet-stream"
	}

	if !strings.HasSuffix(r.path, "/") {
		r.path += "/"
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if form.purpose != "" {
		if form.fields == nil {
			form.fields = map[string]string{}
		}
		form.fields["purpose"] = string(form.purpose)
	}

	for k, v := range form.fields {
		err := mw.WriteField(k, v)
		test.CheckErr(t, err)
	}

	if form.file != nil {
		fw, err := mw.CreateFormFile(form.fileField, form.filename)
		test.CheckErr(t, err)

		_, err = io.Copy(fw, form.file)
		test.CheckErr(t, err)
	}

	err := mw.Close()
	test.CheckErr(t, err)

	req, err := http.NewRequestWithContext(context.Background(), r.method, r.path, &buf)
	test.CheckErr(t, err)

	boundaryCT := "multipart/form-data; boundary=" + mw.Boundary()
	req.Header.Set("Content-Type", boundaryCT)
	req.Header.Set("Accept", ContentTypeJSON)

	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	token := authToken
	if r.authToken != "" {
		token = r.authToken
	}
	if token != "" {
		req.AddCookie(handler.NewSessionCookie(token))
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	_ = handleResponse(t, r, w, nil)

	return &fileResponse
}

func handleResponse(t *testing.T, req RequestArgs, w *httptest.ResponseRecorder, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	if w.Code != req.assertStatus {
		t.Errorf(aur.Red("(%d) %s %s").Bold().String(), w.Code, req.method, req.path)
		t.Errorf(aur.Red("Expecting %d got %d").String(), req.assertStatus, w.Code)

		if len(body) > 0 {
			println(aur.Bold("RequestArgs:").String())
			println(string(body))
		}

		if responseContentType(w) == ContentTypeJSON {
			var respBody map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &respBody)
			if err == nil {
				body2, err2 := json.MarshalIndent(respBody, "", "  ")
				if err2 == nil {
					t.Errorf("Response:\n%s", string(body2))
					t.FailNow()
					return w
				}
			}
		}

		t.Errorf("Response:\n%s", w.Body.String())
		t.FailNow()
		return w
	}

	t.Logf(aur.Green("(%d) %s %s").Bold().String(), w.Code, req.method, req.path)

	if req.bindResponse != nil && responseContentType(w) == ContentTypeJSON {
		err := json.Unmarshal(w.Body.Bytes(), req.bindResponse)
		if err != nil {
			t.Log(w.Body.String())
			t.Fatal(err)
		}
	}

	return w
}

func create[T any](t *testing.T, override ...T) *T {
	t.Helper()

	return test.Create[T](t, tt.Factory, override...)
}

type Query struct {
	Path   string
	Params any // struct only
}

func (q Query) String(t *testing.T) string {
	t.Helper()

	v, err := query.Values(q.Params)
	test.CheckErr(t, err)

	res := q.Path
	params := v.Encode()

	if !strings.HasSuffix(res, "/") {
		res += "/"
	}

	if params != "" {
		res += "?" + params
	}

	return res
}

// forgive me, mother.
func pf(path string, args ...any) string {
	return fmt.Sprintf(path, args...)
}
