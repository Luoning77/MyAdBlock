package engine

import (
	"bytes"
	"encoding/json"
	"github.com/Luoning77/MyAdBlock/internal/agh"
	"github.com/Luoning77/MyAdBlock/internal/config"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(code int, v any) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{StatusCode: code, Body: io.NopCloser(bytes.NewReader(b)), Header: make(http.Header)}
}
func TestModeCompensationRestoresActualPriorFlags(t *testing.T) {
	for _, requested := range []string{"daily", "strict"} {
		flags := map[string]bool{"daily": true, "strict": false}
		failed := false
		r := &Runtime{Sim: true, Data: t.TempDir(), Prefs: config.Defaults()}
		r.AGH = agh.New(23456, "private", "private")
		r.AGH.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
			if strings.HasSuffix(req.URL.Path, "/status") {
				return response(200, Filters{Filters: []Filter{{ID: 2, URL: "daily", Name: "daily", Enabled: flags["daily"]}, {ID: 3, URL: "strict", Name: "strict", Enabled: flags["strict"]}}, Rules: []string{"keep"}}), nil
			}
			var body struct {
				URL  string
				Data struct{ Enabled bool }
			}
			_ = json.NewDecoder(req.Body).Decode(&body)
			if body.Data.Enabled && !failed {
				failed = true
				return response(500, map[string]bool{}), nil
			}
			flags[body.URL] = body.Data.Enabled
			return response(200, map[string]bool{}), nil
		})}
		if r.Mode(requested) == nil {
			t.Fatal("failure ignored")
		}
		if !flags["daily"] || flags["strict"] || r.Prefs.Mode != "daily" {
			t.Fatal("rollback enabled wrong AWAvenue", flags)
		}
	}
}
func TestFailedRefreshRestoresSourcesAndCacheWithoutEndingPause(t *testing.T) {
	data := t.TempDir()
	p := filepath.Join(data, "subscriptions", "1.txt")
	cache := filepath.Join(data, "agh-data", "data", "filters", "1.txt")
	old := []byte("old validated snapshot")
	_ = config.Atomic(p, old)
	_ = config.Atomic(cache, old)
	r := &Runtime{Data: data, Sim: true, M: Machine{State: Paused}}
	r.AGH = agh.New(23456, "private", "private")
	r.AGH.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/status") {
			return response(200, Filters{Filters: []Filter{{ID: 1, URL: p, Enabled: true}}}), nil
		}
		return response(500, map[string]bool{}), nil
	})}
	if r.applyRuleUpdates(map[string][]byte{"1": []byte("new snapshot")}) == nil {
		t.Fatal("refresh failure ignored")
	}
	for _, f := range []string{p, cache} {
		b, e := os.ReadFile(f)
		if e != nil || !bytes.Equal(b, old) {
			t.Fatal("snapshot not rolled back", f, e)
		}
	}
	if r.M.State != Paused {
		t.Fatal("refresh failure cancelled user pause")
	}
}

func TestPausedReconciliationAvoidsRedundantProtectionWrites(t *testing.T) {
	calls := 0
	r := &Runtime{Data: t.TempDir(), Sim: true, cmd: &exec.Cmd{}, done: make(chan error, 1)}
	r.M.Pause(time.Now(), 0)
	r.AGH = agh.New(23456, "private", "private")
	r.AGH.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/protection") {
			calls++
		}
		return response(200, map[string]any{"protection_enabled": false}), nil
	})}
	r.Tick()
	r.Tick()
	if calls != 0 {
		t.Fatal("already-paused core rewritten every tick", calls)
	}
}
