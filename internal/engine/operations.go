// SPDX-License-Identifier: GPL-3.0-or-later
package engine

import (
	"errors"
	"fmt"
	"github.com/Luoning77/MyAdBlock/internal/config"
	"github.com/Luoning77/MyAdBlock/internal/network"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Filter struct {
	ID      int64  `json:"id"`
	URL     string `json:"url"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Rules   int    `json:"rules_count"`
	Updated string `json:"last_updated"`
}
type Filters struct {
	Filters []Filter `json:"filters"`
	Rules   []string `json:"user_rules"`
}

func (r *Runtime) Filters() (Filters, error) {
	var f Filters
	e := r.AGH.Call("GET", "/filtering/status", nil, &f)
	return f, e
}
func (r *Runtime) Mode(mode string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if mode != "daily" && mode != "strict" {
		return errors.New("无效模式")
	}
	f, e := r.Filters()
	if e != nil {
		return e
	}
	old := r.Prefs.Mode
	// Disable the other AWAvenue first; retain all user subscriptions and user_rules.
	change := func(id int64, on bool) error {
		for _, v := range f.Filters {
			if v.ID == id {
				return r.AGH.Call("POST", "/filtering/set_url", map[string]any{"url": v.URL, "whitelist": false, "data": map[string]any{"name": v.Name, "url": v.URL, "enabled": on}}, nil)
			}
		}
		return errors.New("内置订阅缺失，请恢复过滤预设")
	}
	rollback := func() error {
		var errs []error
		for _, id := range []int64{2, 3} {
			if e := change(id, false); e != nil {
				errs = append(errs, e)
			}
		}
		for _, v := range f.Filters {
			if (v.ID == 2 || v.ID == 3) && v.Enabled {
				if e := change(v.ID, true); e != nil {
					errs = append(errs, e)
				}
			}
		}
		return errors.Join(errs...)
	}
	off, on := int64(3), int64(2)
	if mode == "strict" {
		off, on = 2, 3
	}
	if e = change(off, false); e != nil {
		return e
	}
	if e = change(on, true); e != nil {
		return errors.Join(e, rollback())
	}
	r.Prefs.Mode = mode
	if e = config.Save(filepath.Join(r.Data, "preferences.json"), r.Prefs); e != nil {
		r.Prefs.Mode = old
		return errors.Join(e, rollback())
	}
	r.activeUpstream = ""
	return nil
}
func (r *Runtime) SavePrefs(p config.Preferences) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	p.Schema = config.Schema
	p.Mode = r.Prefs.Mode
	if e := ValidatePrefs(p); e != nil {
		return e
	}
	if e := config.Save(filepath.Join(r.Data, "preferences.json"), p); e != nil {
		return e
	}
	r.Prefs = p
	r.activeUpstream = ""
	r.M.Fail = 0
	r.M.Recover = 0
	_ = r.remove()
	if r.M.State != Paused {
		r.M.State = Bypass
	}
	return nil
}
func (r *Runtime) UserRules(rules []string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if len(rules) > 10000 {
		return errors.New("最多 10000 条规则")
	}
	for _, s := range rules {
		if len(s) > 4096 || strings.ContainsAny(s, "\x00\r\n") {
			return errors.New("无效规则")
		}
	}
	return r.AGH.Call("POST", "/filtering/set_rules", map[string]any{"rules": rules}, nil)
}
func (r *Runtime) DomainRule(domain string, allow bool) error {
	domain = strings.TrimSuffix(strings.TrimSpace(domain), ".")
	if len(domain) > 253 || strings.ContainsAny(domain, "/|^$@#* \r\n\x00:") || !strings.Contains(domain, ".") {
		return errors.New("无效域名")
	}
	f, e := r.Filters()
	if e != nil {
		return e
	}
	s := "||" + strings.ToLower(domain) + "^"
	if allow {
		s = "@@" + s
	}
	for _, v := range f.Rules {
		if v == s {
			return nil
		}
	}
	return r.UserRules(append(f.Rules, s))
}
func fetchRules(s string, bootstrap []string) ([]byte, error) {
	if e := network.SafeHTTPS(s); e != nil {
		return nil, e
	}
	req, e := http.NewRequest("GET", s, nil)
	if e != nil {
		return nil, e
	}
	resp, e := network.SnapshotHTTP(bootstrap).Do(req)
	if e != nil {
		return nil, errors.New("订阅下载失败，旧快照保留")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("订阅 HTTP %d；旧快照保留", resp.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, (20<<20)+1))
	if e != nil {
		return nil, e
	}
	return b, network.ValidateRules(b)
}
func (r *Runtime) UpdateRules() error {
	if !r.updateMu.TryLock() {
		return errors.New("订阅更新正在进行")
	}
	defer r.updateMu.Unlock()
	r.Mu.Lock()
	var custom map[string]string
	_ = config.Read(filepath.Join(r.Data, "sources.json"), &custom)
	sources := map[string]string{}
	for i, u := range config.Sources {
		sources[fmt.Sprint(i+1)] = u
	}
	for k, v := range custom {
		sources[k] = v
	}
	bootstrap := network.MergeBootstrap(r.Prefs.Bootstrap, r.Status.Link.DNS)
	r.Mu.Unlock()
	updates := map[string][]byte{}
	var errs []error
	total := 0
	for id, u := range sources {
		b, e := fetchRules(u, bootstrap)
		if e != nil {
			errs = append(errs, e)
			continue
		}
		total += len(b)
		if total > 64<<20 {
			errs = append(errs, errors.New("批量快照超过 64 MiB，保留旧快照"))
			break
		}
		updates[id] = b
	}
	if len(updates) > 0 {
		if e := r.applyRuleUpdates(updates); e != nil {
			errs = append(errs, e)
		}
	}
	if len(errs) > 0 {
		r.Mu.Lock()
		r.Status.RulesUpdated = "更新部分或全部失败；失败订阅保留旧快照"
		r.Mu.Unlock()
		return errors.Join(errs...)
	}
	return nil
}

// All validated sources are committed together, followed by ONE AGH refresh to avoid its rate limiter.
func (r *Runtime) applyRuleUpdates(updates map[string][]byte) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	f, e := r.Filters()
	if e != nil {
		return e
	}
	old := map[string][]byte{}
	paths := map[string]string{}
	for id := range updates {
		if id == "" || strings.Trim(id, "0123456789") != "" {
			return errors.New("非法订阅标识")
		}
		p := filepath.Join(r.Data, "subscriptions", id+".txt")
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		old[id] = b
		paths[id] = p
	}
	restoreSources := func() {
		for id, b := range old {
			_ = config.Atomic(paths[id], b)
		}
	}
	for id, b := range updates {
		if e = config.Atomic(paths[id], b); e != nil {
			restoreSources()
			return e
		}
	}
	var result struct {
		Updated int `json:"updated"`
	}
	if e = r.AGH.Call("POST", "/filtering/refresh", map[string]any{"whitelist": false}, &result); e != nil {
		// An HTTP failure may still have loaded a partial new cache. Reconcile while stopped.
		_ = r.remove()
		r.StopCore()
		restoreSources()
		for _, v := range f.Filters {
			for id, p := range paths {
				if v.URL == p {
					_ = config.Atomic(filepath.Join(r.Data, "agh-data", "data", "filters", fmt.Sprint(v.ID)+".txt"), old[id])
				}
			}
		}
		se := r.startControlled()
		r.activeUpstream = ""
		if r.M.State != Paused {
			r.M.State = Bypass
			r.M.Recover = 0
			r.M.Fail = 0
		}
		return errors.Join(e, se)
	}
	if result.Updated == 0 {
		r.Status.RulesUpdated = "快照已验证；AGH 未报告规则变化（未变或限流），更新时间以订阅状态为准"
	} else {
		r.Status.RulesUpdated = time.Now().Format(time.RFC3339)
	}
	return nil
}

func (r *Runtime) AddFilter(name, s string) error {
	if len(name) < 1 || len(name) > 80 {
		return errors.New("名称长度 1–80")
	}
	r.Mu.Lock()
	bootstrap := network.MergeBootstrap(r.Prefs.Bootstrap, r.Status.Link.DNS)
	r.Mu.Unlock()
	b, e := fetchRules(s, bootstrap)
	if e != nil {
		return e
	}
	r.Mu.Lock()
	defer r.Mu.Unlock()
	id := fmt.Sprint(time.Now().UnixNano())
	p := filepath.Join(r.Data, "subscriptions", id+".txt")
	if e = config.Atomic(p, b); e != nil {
		return e
	}
	var sources map[string]string
	_ = config.Read(filepath.Join(r.Data, "sources.json"), &sources)
	if sources == nil {
		sources = map[string]string{}
	}
	sources[id] = s
	if e = config.Save(filepath.Join(r.Data, "sources.json"), sources); e != nil {
		return e
	}
	if e = r.AGH.Call("POST", "/filtering/add_url", map[string]any{"name": name, "url": p, "whitelist": false}, nil); e != nil {
		delete(sources, id)
		_ = config.Save(filepath.Join(r.Data, "sources.json"), sources)
		os.Remove(p)
	}
	return e
}
func (r *Runtime) RemoveFilter(id int64) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if id <= 3 {
		return errors.New("内置订阅不能删除；可切换预设")
	}
	f, e := r.Filters()
	if e != nil {
		return e
	}
	for _, v := range f.Filters {
		if v.ID == id {
			if filepath.Dir(v.URL) != filepath.Join(r.Data, "subscriptions") {
				return errors.New("只管理通过此界面添加的订阅")
			}
			if e = r.AGH.Call("POST", "/filtering/remove_url", map[string]any{"url": v.URL, "whitelist": false}, nil); e != nil {
				return e
			}
			key := strings.TrimSuffix(filepath.Base(v.URL), ".txt")
			var sources map[string]string
			_ = config.Read(filepath.Join(r.Data, "sources.json"), &sources)
			delete(sources, key)
			_ = config.Save(filepath.Join(r.Data, "sources.json"), sources)
			return os.Remove(v.URL)
		}
	}
	return errors.New("订阅不存在")
}
func (r *Runtime) Export() ([]byte, error) {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	m, e := config.Load(filepath.Join(r.Data, "AdGuardHome.yaml"))
	if e != nil {
		return nil, e
	}
	portable := config.YAML{"schema_version": 34, "user_rules": m["user_rules"]}
	d := config.Section(m, "dns")
	safe := map[string]any{}
	for _, k := range []string{"cache_enabled", "cache_size", "cache_ttl_min", "cache_ttl_max", "enable_dnssec"} {
		if v, ok := d[k]; ok {
			safe[k] = v
		}
	}
	portable["dns"] = safe
	return marshalYAML(portable)
}
func (r *Runtime) Import(b []byte) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	initial, e := config.Load(filepath.Join(r.Data, "AdGuardHome.yaml"))
	if e != nil {
		return e
	}
	if _, e = config.Import(b, initial); e != nil {
		return e
	}

	_ = r.remove()
	r.StopCore()
	p := filepath.Join(r.Data, "AdGuardHome.yaml")
	base, e := config.Load(p)
	if e != nil {
		return e
	}
	m, e := config.Import(b, base)
	if e == nil {
		e = r.CommitConfig(m)
	}
	startErr := r.startControlled()
	r.activeUpstream = ""
	if r.M.State != Paused {
		r.M.State = Bypass
	}
	return errors.Join(e, startErr)
}
func (r *Runtime) Restore(scope string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if scope != "filters" && scope != "dns" && scope != "all" {
		return errors.New("无效恢复范围")
	}
	_ = r.remove()
	r.StopCore()
	p := filepath.Join(r.Data, "AdGuardHome.yaml")
	m, e := config.Load(p)
	if e != nil {
		return e
	}
	fresh := config.Fresh(r.Data)
	if scope == "filters" || scope == "all" {
		m["filters"] = fresh["filters"]
		m["whitelist_filters"] = []any{}
		m["user_rules"] = []string{}
	}
	if scope == "dns" || scope == "all" {
		m["dns"] = fresh["dns"]
		r.Prefs.DoH = append([]string{}, config.PublicDoH...)
		r.Prefs.Bootstrap = append([]string{}, config.Bootstrap...)
	}
	if scope == "all" {
		m["querylog"] = fresh["querylog"]
		m["statistics"] = fresh["statistics"]
	}
	// Credentials and data are preserved, including for full reset.
	if e = config.Normalize(m, r.Data, r.Ports.AGH, r.Ports.DNS, r.Status.IPv6Available, r.AGH.Password); e == nil {
		e = r.CommitConfig(m)
	}
	if e == nil {
		if scope != "dns" {
			r.Prefs.Mode = "daily"
		}
		e = config.Save(filepath.Join(r.Data, "preferences.json"), r.Prefs)
	}
	se := r.startControlled()
	r.activeUpstream = ""
	if r.M.State != Paused {
		r.M.State = Bypass
	}
	return errors.Join(e, se)
}
func (r *Runtime) QueryLog(search, filter, older string) (any, error) {
	if len(search) > 253 || len(older) > 100 {
		return nil, errors.New("查询参数过长")
	}
	allowed := map[string]bool{"all": true, "filtered": true, "blocked": true, "whitelisted": true, "processed": true}
	if !allowed[filter] {
		filter = "all"
	}
	v := url.Values{"search": {search}, "response_status": {filter}, "limit": {"50"}}
	if older != "" {
		v.Set("older_than", older)
	}
	var data any
	e := r.AGH.Call("GET", "/querylog?"+v.Encode(), nil, &data)
	return data, e
}

// Planned user operations do not consume the automatic crash restart budget.
func (r *Runtime) startControlled() error {
	saved := r.attempts
	r.attempts = nil
	e := r.StartCore()
	r.attempts = saved
	return e
}
func (r *Runtime) ChangePassword(password string) error {
	if len(password) < 1 || len(password) > 72 {
		return errors.New("密码长度 1–72 字节")
	}
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if e := r.remove(); e != nil {
		return errors.New("无法安全撤销接管，密码尚未修改")
	}
	r.StopCore()
	p := filepath.Join(r.Data, "AdGuardHome.yaml")
	m, e := config.Load(p)
	if e == nil {
		found := false
		users, _ := m["users"].([]any)
		for _, u := range users {
			entry, ok := u.(map[string]any)
			if ok && entry["name"] == "root" {
				entry["password"] = config.Hash(password)
				found = true
			}
		}
		if !found {
			e = errors.New("root 用户缺失，拒绝修改其他账号")
		}
		if e == nil {
			e = r.CommitConfig(m)
		}
	}
	se := r.startControlled()
	r.activeUpstream = ""
	if r.M.State != Paused {
		r.M.State = Bypass
	}
	if se != nil && e == nil {
		return errors.New("密码已保存，但核心重启失败；请用新密码重试并检查服务")
	}
	return errors.Join(e, se)
}
