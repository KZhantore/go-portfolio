package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func request(t *testing.T, h http.Handler, method, path, body string, code int) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	if w.Code != code {
		t.Fatalf("%s %s: got %d, want %d (%s)", method, path, w.Code, code, w.Body.String())
	}
	return w
}
func TestLifecycle(t *testing.T) {
	file := filepath.Join(t.TempDir(), "tasks.json")
	s, err := openStore(file)
	if err != nil {
		t.Fatal(err)
	}
	h := routes(s)
	request(t, h, "GET", "/tasks", "", 200)
	created := request(t, h, "POST", "/tasks", `{"title":"  Learn Go  "}`, 201)
	if created.Header().Get("Location") != "/tasks/1" || !strings.Contains(created.Body.String(), `"title":"Learn Go"`) {
		t.Fatal(created)
	}
	request(t, h, "PUT", "/tasks/1", `{"title":"Learn Go","done":true}`, 200)
	s, err = openStore(file)
	if err != nil {
		t.Fatal(err)
	}
	h = routes(s)
	got := request(t, h, "GET", "/tasks/1", "", 200)
	if !strings.Contains(got.Body.String(), `"done":true`) {
		t.Fatal(got.Body.String())
	}
	request(t, h, "DELETE", "/tasks/1", "", 204)
	request(t, h, "GET", "/tasks/1", "", 404)
	next := request(t, h, "POST", "/tasks", `{"title":"Next"}`, 201)
	if next.Header().Get("Location") != "/tasks/2" {
		t.Fatal("deleted ID was reused")
	}
}
func TestInvalidRequests(t *testing.T) {
	s, err := openStore(filepath.Join(t.TempDir(), "tasks.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := routes(s)
	for _, body := range []string{`{}`, `{"title":"  "}`, `{"title":"x","extra":1}`, `{"title":"x"} {}`, `null`} {
		request(t, h, "POST", "/tasks", body, 400)
	}
	request(t, h, "POST", "/tasks", `{"title":"`+strings.Repeat("x", 9000)+`"}`, 413)
	request(t, h, "GET", "/tasks/0", "", 400)
	request(t, h, "PATCH", "/tasks/1", "", 405)
}
func TestStorageFailureDoesNotMutateMemory(t *testing.T) {
	s, err := openStore(filepath.Join(t.TempDir(), "missing", "tasks.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := routes(s)
	request(t, h, "POST", "/tasks", `{"title":"Not saved"}`, 500)
	if got := request(t, h, "GET", "/tasks", "", 200).Body.String(); got != "[]\n" {
		t.Fatal(got)
	}
}
