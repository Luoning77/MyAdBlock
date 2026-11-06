package bridge

import (
	"github.com/Luoning77/MyAdBlock/internal/engine"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testServer() *Server { r := &engine.Runtime{Ports: engine.Ports{UI: 23456}}; return New(r) }
func request(s *Server, method, path, origin, host, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://127.0.0.1:23456"+path, strings.NewReader(body))
	r.Host = host
	r.RemoteAddr = "127.0.0.1:45000"
	r.Header.Set("Origin", origin)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestBoundaries(t *testing.T) {
	s := testServer()
	cases := []struct {
		method, origin, host string
		want                 int
	}{{"GET", "", "127.0.0.1:23456", 401}, {"GET", "http://evil.test", "127.0.0.1:23456", 403}, {"GET", "", "evil.test:23456", 403}, {"POST", "", "127.0.0.1:23456", 403}, {"POST", "http://evil.test", "127.0.0.1:23456", 403}}
	for _, c := range cases {
		w := request(s, c.method, "/api/status", c.origin, c.host, "")
		if w.Code != c.want {
			t.Fatalf("%+v got %d", c, w.Code)
		}
	}
}
func TestKnownPasswordIsNotRootGrant(t *testing.T) {
	s := testServer()
	s.sessions["session"] = Session{Expires: time.Now().Add(time.Minute), CSRF: "csrf"}
	for _, csrf := range []string{"", "csrf"} {
		r := httptest.NewRequest("POST", "http://127.0.0.1:23456/api/pause", strings.NewReader(`{"seconds":300}`))
		r.Host = "127.0.0.1:23456"
		r.RemoteAddr = "127.0.0.1:12345"
		r.Header.Set("Origin", "http://127.0.0.1:23456")
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-CSRF-Token", csrf)
		r.AddCookie(&http.Cookie{Name: "mab_session", Value: "session"})
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal(w.Code)
		}
	}
}
func TestNoArbitraryCommandAPI(t *testing.T) {
	s := testServer()
	s.sessions["session"] = Session{Expires: time.Now().Add(time.Minute), Grant: time.Now().Add(time.Minute), CSRF: "csrf"}
	r := httptest.NewRequest("POST", "http://127.0.0.1:23456/api/shell", strings.NewReader(`{"command":"id"}`))
	r.Host = "127.0.0.1:23456"
	r.RemoteAddr = "127.0.0.1:12345"
	r.Header.Set("Origin", "http://127.0.0.1:23456")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", "csrf")
	r.AddCookie(&http.Cookie{Name: "mab_session", Value: "session"})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
}
