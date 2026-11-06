// SPDX-License-Identifier: GPL-3.0-or-later
package bridge

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Luoning77/MyAdBlock/internal/agh"
	"github.com/Luoning77/MyAdBlock/internal/config"
	"github.com/Luoning77/MyAdBlock/internal/engine"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Session struct {
	Expires time.Time
	Grant   time.Time
	CSRF    string
}
type Server struct {
	slots    chan struct{}
	R        *engine.Runtime
	mu       sync.Mutex
	sessions map[string]Session
	attempts int
	window   time.Time
}

func New(r *engine.Runtime) *Server {
	return &Server{R: r, sessions: map[string]Session{}, slots: make(chan struct{}, 8)}
}
func equal(a, b string) bool {
	return len(a) > 0 && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func (s *Server) origin() string { return fmt.Sprintf("http://127.0.0.1:%d", s.R.Ports.UI) }
func (s *Server) boundary(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; worker-src 'self' blob:; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	host := fmt.Sprintf("127.0.0.1:%d", s.R.Ports.UI)
	if r.Host != host {
		http.Error(w, "Host denied", 403)
		return false
	}
	h, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil || !net.ParseIP(h).IsLoopback() {
		http.Error(w, "loopback only", 403)
		return false
	}
	if o := r.Header.Get("Origin"); o != "" && o != s.origin() {
		http.Error(w, "Origin denied", 403)
		return false
	}
	if f := r.Header.Get("Sec-Fetch-Site"); f != "" && f != "same-origin" && f != "none" {
		http.Error(w, "cross-site denied", 403)
		return false
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if r.Header.Get("Origin") != s.origin() || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "same-origin JSON required", 403)
			return false
		}
	}
	return true
}
func (s *Server) session(r *http.Request) (string, Session, bool) {
	c, e := r.Cookie("mab_session")
	if e != nil {
		return "", Session{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.sessions[c.Value]
	if !ok || time.Now().After(v.Expires) {
		delete(s.sessions, c.Value)
		return "", v, false
	}
	return c.Value, v, true
}
func reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, e error) {
	w.WriteHeader(400)
	reply(w, map[string]any{"error": e.Error()})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return errors.New("请求 JSON 格式无效或过大")
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return errors.New("请求包含多余或无效 JSON")
	}
	return nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		http.Error(w, "本地服务繁忙，请稍后重试", 503)
		return
	}
	if !s.boundary(w, r) {
		return
	}
	if r.URL.Path == "/api/login" && r.Method == "POST" {
		s.login(w, r)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "method denied", 405)
			return
		}
		if strings.Contains(r.URL.Path, "..") || strings.ContainsAny(r.URL.Path, "\\\x00") {
			http.Error(w, "path denied", 400)
			return
		}
		http.FileServer(http.Dir(filepath.Join(s.R.Module, "webroot"))).ServeHTTP(w, r)
		return
	}
	id, sess, ok := s.session(r)
	if !ok {
		http.Error(w, "请登录", 401)
		return
	}
	if r.Method == "POST" {
		if !equal(r.Header.Get("X-CSRF-Token"), sess.CSRF) {
			http.Error(w, "CSRF denied", 403)
			return
		}
		if r.URL.Path != "/api/authorize" && r.URL.Path != "/api/logout" && time.Now().After(sess.Grant) {
			http.Error(w, "请从 Root 管理器操作按钮重新授权（授权有效期 5 分钟）", 403)
			return
		}
	}
	var e error
	switch r.Method + " " + r.URL.Path {
	case "GET /api/session":
		reply(w, map[string]any{"csrf": sess.CSRF, "authorized": time.Now().Before(sess.Grant), "grantUntil": sess.Grant.Unix()})
		return
	case "POST /api/logout":
		s.mu.Lock()
		delete(s.sessions, id)
		s.mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "mab_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
		reply(w, map[string]bool{"ok": true})
		return
	case "POST /api/authorize":
		var in struct {
			Token string `json:"token"`
		}
		if e = decode(w, r, &in); e == nil {
			var grant struct {
				Token   string
				Expires int64
			}
			p := filepath.Join(s.R.Data, "grant.json")
			s.mu.Lock()
			if config.Read(p, &grant) != nil || time.Now().Unix() > grant.Expires || !equal(in.Token, grant.Token) {
				e = errors.New("授权无效或已过期，请重新点击操作按钮")
			} else {
				sess.Grant = time.Now().Add(5 * time.Minute)
				s.sessions[id] = sess
				_ = os.Remove(p)
			}
			s.mu.Unlock()
		}
	case "GET /api/status":
		reply(w, s.R.Snapshot())
		return
	case "GET /api/preferences":
		s.R.Mu.Lock()
		p := s.R.Prefs
		s.R.Mu.Unlock()
		reply(w, p)
		return
	case "GET /api/stats":
		var data any
		recent, _ := strconv.Atoi(r.URL.Query().Get("recent"))
		if recent < 3600000 || recent > 86400000 || recent%3600000 != 0 {
			recent = 86400000
		}
		e = s.R.AGH.Call("GET", "/stats?recent="+strconv.Itoa(recent), nil, &data)
		if e == nil {
			reply(w, data)
			return
		}
	case "GET /api/querylog":
		var data any
		data, e = s.R.QueryLog(r.URL.Query().Get("search"), r.URL.Query().Get("filter"), r.URL.Query().Get("older"))
		if e == nil {
			reply(w, data)
			return
		}
	case "GET /api/filters":
		var data any
		data, e = s.R.Filters()
		if e == nil {
			reply(w, data)
			return
		}
	case "GET /api/backup":
		if time.Now().After(sess.Grant) {
			http.Error(w, "请重新授权", 403)
			return
		}
		var b []byte
		b, e = s.R.Export()
		if e == nil {
			w.Header().Set("Content-Type", "application/yaml")
			w.Header().Set("Content-Disposition", "attachment; filename=myadblock-portable.yaml")
			_, _ = w.Write(b)
			return
		}
	case "POST /api/pause":
		var in struct {
			Seconds int `json:"seconds"`
		}
		if e = decode(w, r, &in); e == nil {
			e = s.R.Pause(in.Seconds)
		}
	case "POST /api/resume":
		e = s.R.Resume()
	case "POST /api/mode":
		var in struct {
			Mode string `json:"mode"`
		}
		if e = decode(w, r, &in); e == nil {
			e = s.R.Mode(in.Mode)
		}
	case "POST /api/preferences":
		var in config.Preferences
		if e = decode(w, r, &in); e == nil {
			e = s.R.SavePrefs(in)
		}
	case "POST /api/rules":
		var in struct {
			Rules []string `json:"rules"`
		}
		if e = decode(w, r, &in); e == nil {
			e = s.R.UserRules(in.Rules)
		}
	case "POST /api/domain":
		var in struct {
			Domain string `json:"domain"`
			Allow  bool   `json:"allow"`
		}
		if e = decode(w, r, &in); e == nil {
			e = s.R.DomainRule(in.Domain, in.Allow)
		}
	case "POST /api/filter/add":
		var in struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}
		if e = decode(w, r, &in); e == nil {
			e = s.R.AddFilter(in.Name, in.URL)
		}
	case "POST /api/filter/remove":
		var in struct {
			ID int64 `json:"id"`
		}
		if e = decode(w, r, &in); e == nil {
			e = s.R.RemoveFilter(in.ID)
		}
	case "POST /api/filter/update":
		e = s.R.UpdateRules()
	case "POST /api/import":
		var in struct {
			YAML    string `json:"yaml"`
			Confirm bool   `json:"confirm"`
		}
		if e = decode(w, r, &in); e == nil {
			if !in.Confirm {
				e = errors.New("需要确认导入")
			} else {
				e = s.R.Import([]byte(in.YAML))
			}
		}
	case "POST /api/restore":
		var in struct {
			Scope   string `json:"scope"`
			Confirm bool   `json:"confirm"`
		}
		if e = decode(w, r, &in); e == nil {
			if !in.Confirm {
				e = errors.New("需要确认恢复及清除自定义规则")
			} else {
				e = s.R.Restore(in.Scope)
			}
		}
	case "POST /api/password":
		var in struct {
			Password string `json:"password"`
		}
		if e = decode(w, r, &in); e == nil {
			if len(in.Password) < 1 || len(in.Password) > 72 {
				e = errors.New("密码长度 1–72 字节")
			} else {
				e = s.R.ChangePassword(in.Password)
				if e == nil {
					s.mu.Lock()
					s.sessions = map[string]Session{}
					s.mu.Unlock()
				}
			}
		}
	default:
		http.Error(w, "API 不存在或方法不允许", 404)
		return
	}
	if e != nil {
		fail(w, e)
		return
	}
	reply(w, map[string]bool{"ok": true})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	now := time.Now()
	if now.Sub(s.window) > time.Minute {
		s.window = now
		s.attempts = 0
	}
	s.attempts++
	blocked := s.attempts > 10
	s.mu.Unlock()
	if blocked {
		http.Error(w, "登录过于频繁，请稍后重试", 429)
		return
	}
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if e := decode(w, r, &in); e != nil {
		fail(w, e)
		return
	}
	if in.Username != "root" || len(in.Password) > 72 {
		http.Error(w, "凭据无效", 401)
		return
	}
	c := agh.New(s.R.Ports.AGH, in.Username, in.Password)
	var profile map[string]any
	if c.Call("GET", "/profile", nil, &profile) != nil {
		http.Error(w, "凭据无效或 AGH 尚未就绪", 401)
		return
	}
	id := config.Random()
	sess := Session{Expires: now.Add(30 * time.Minute), CSRF: config.Random()}
	s.mu.Lock()
	for k, v := range s.sessions {
		if now.After(v.Expires) {
			delete(s.sessions, k)
		}
	}
	if len(s.sessions) >= 32 {
		s.mu.Unlock()
		http.Error(w, "会话已满", 429)
		return
	}
	s.sessions[id] = sess
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "mab_session", Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 1800})
	reply(w, map[string]any{"csrf": sess.CSRF, "authorized": false})
}
func (s *Server) HTTP() *http.Server {
	return &http.Server{Handler: s, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
}
func PortString(p int) string { return strconv.Itoa(p) }
