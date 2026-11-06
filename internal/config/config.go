// SPDX-License-Identifier: GPL-3.0-or-later
package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

const Schema = 1

var PublicDoH = []string{"https://dns.alidns.com/dns-query", "https://cloudflare-dns.com/dns-query"}
var Bootstrap = []string{"223.5.5.5", "1.1.1.1", "2606:4700:4700::1111"}
var Sources = []string{"https://adguardteam.github.io/HostlistsRegistry/assets/filter_1.txt", "https://raw.githubusercontent.com/TG-Twilight/AWAvenue-Ads-Rule/main/Filters/AWAvenue-Ads-Rule-Adguard-Only.Ads.txt", "https://raw.githubusercontent.com/TG-Twilight/AWAvenue-Ads-Rule/main/AWAvenue-Ads-Rule.txt"}

type Preferences struct {
	Schema    int      `json:"schema"`
	Mode      string   `json:"mode"`
	DoH       []string `json:"doh"`
	Bootstrap []string `json:"bootstrap"`
	IPv6      bool     `json:"ipv6"`
}

func Defaults() Preferences {
	return Preferences{Schema, "daily", append([]string{}, PublicDoH...), append([]string{}, Bootstrap...), true}
}
func Random() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func Atomic(path string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".txn-")
	if e != nil {
		return e
	}
	n := f.Name()
	defer os.Remove(n)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	if e = os.Rename(n, path); e != nil {
		return e
	}
	d, e := os.Open(filepath.Dir(path))
	if e == nil {
		defer d.Close()
		e = d.Sync()
	}
	return e
}
func Save(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return Atomic(path, b)
}
func Read(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}

type YAML = map[string]any

func Load(path string) (YAML, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	return Parse(b)
}
func Parse(b []byte) (YAML, error) {
	if len(b) > 2<<20 {
		return nil, errors.New("配置超过 2 MiB")
	}
	var m YAML
	e := yaml.Unmarshal(b, &m)
	if e != nil {
		return nil, e
	}
	if m == nil {
		return nil, errors.New("空配置")
	}
	return m, nil
}
func Write(path string, m YAML) error {
	b, e := yaml.Marshal(m)
	if e != nil {
		return e
	}
	return Atomic(path, b)
}
func Section(m YAML, k string) map[string]any {
	if a, ok := m[k].(map[string]any); ok {
		return a
	}
	a := map[string]any{}
	m[k] = a
	return a
}
func Hash(s string) string {
	b, e := bcrypt.GenerateFromPassword([]byte(s), 10)
	if e != nil {
		panic(e)
	}
	return string(b)
}

// Normalize is used ONLY with a stopped AGH. All user fields outside the safety boundary survive.
func Normalize(m YAML, data string, httpPort, dnsPort int, ipv6 bool, secret string) error {
	if v, ok := m["schema_version"]; ok && fmt.Sprint(v) != "34" {
		return errors.New("仅接受 AGH schema 34；迁移失败，保留原配置")
	}
	m["schema_version"] = 34
	Section(m, "filtering")["safe_fs_patterns"] = []string{filepath.Join(data, "subscriptions", "*.txt")}
	filtering := Section(m, "filtering")
	rewrites, _ := filtering["rewrites"].([]any)
	keptRewrites := []any{}
	for _, v := range rewrites {
		entry, ok := v.(map[string]any)
		if ok && entry["domain"] != "myadblock-health.invalid" {
			keptRewrites = append(keptRewrites, v)
		}
	}
	filtering["rewrites"] = append(keptRewrites, map[string]any{"domain": "myadblock-health.invalid", "answer": "192.0.2.173", "enabled": true})
	filtering["rewrites_enabled"] = true

	h := Section(m, "http")
	h["address"] = fmt.Sprintf("127.0.0.1:%d", httpPort)
	h["pprof"] = map[string]any{"enabled": false}
	h["doh"] = map[string]any{"insecure_enabled": false}
	d := Section(m, "dns")
	hosts := []string{"127.0.0.1"}
	if ipv6 {
		hosts = append(hosts, "::1")
	}
	d["bind_hosts"] = hosts
	d["port"] = dnsPort
	d["upstream_dns_file"] = ""
	d["serve_plain_dns"] = true
	d["fallback_dns"] = []string{}
	d["ipset"] = []string{}
	d["ipset_file"] = ""
	d["use_private_ptr_resolvers"] = false
	d["local_ptr_upstreams"] = []string{}
	d["hostsfile_enabled"] = false
	d["use_dns64"] = false
	m["tls"] = map[string]any{"enabled": false}
	m["dhcp"] = map[string]any{"enabled": false}
	m["http_proxy"] = ""
	m["os"] = map[string]any{"user": "", "group": "", "rlimit_nofile": 0}
	Section(m, "querylog")["dir_path"] = filepath.Join(data, "agh-data")
	Section(m, "statistics")["dir_path"] = filepath.Join(data, "agh-data")
	// The private random management account survives root password changes, and is never exposed to the browser.
	users, _ := m["users"].([]any)
	kept := []any{}
	for _, u := range users {
		v, ok := u.(map[string]any)
		if ok && v["name"] != "myadblock-controller" {
			kept = append(kept, v)
		}
	}
	if len(kept) == 0 {
		kept = append(kept, map[string]any{"name": "root", "password": Hash("root")})
	}
	kept = append(kept, map[string]any{"name": "myadblock-controller", "password": Hash(secret)})
	m["users"] = kept
	return nil
}
func Fresh(data string) YAML {
	filters := []any{}
	for i := 1; i <= 3; i++ {
		filters = append(filters, map[string]any{"enabled": i != 3, "url": filepath.Join(data, "subscriptions", fmt.Sprintf("%d.txt", i)), "name": []string{"AdGuard DNS filter", "AWAvenue · 日常", "AWAvenue · 严格"}[i-1], "id": i})
	}
	return YAML{"schema_version": 34, "users": []any{map[string]any{"name": "root", "password": Hash("root")}}, "auth_attempts": 5, "block_auth_min": 15, "language": "zh-cn", "theme": "auto", "dns": map[string]any{"upstream_dns": PublicDoH, "bootstrap_dns": Bootstrap, "upstream_mode": "load_balance", "upstream_timeout": "2s", "cache_enabled": true, "cache_size": 33554432, "ratelimit": 0, "enable_dnssec": true}, "filtering": map[string]any{"protection_enabled": true, "filtering_enabled": true, "filters_update_interval": 0}, "filters": filters, "whitelist_filters": []any{}, "user_rules": []string{}, "querylog": map[string]any{"enabled": true, "file_enabled": false, "size_memory": 1000, "interval": "24h"}, "statistics": map[string]any{"enabled": true, "interval": "168h"}, "log": map[string]any{"enabled": false, "file": "", "verbose": false}}
}

// Import supports a deliberately bounded portable subset. Paths, listeners, users and execution settings are never imported.
func Import(b []byte, base YAML) (YAML, error) {
	m, e := Parse(b)
	if e != nil {
		return nil, e
	}
	allowed := map[string]bool{"schema_version": true, "dns": true, "user_rules": true, "querylog": true, "statistics": true}
	for k := range m {
		if !allowed[k] {
			return nil, fmt.Errorf("不允许导入字段 %s；请导出可移植配置", k)
		}
	}
	if fmt.Sprint(m["schema_version"]) != "34" {
		return nil, errors.New("schema_version 必须为 34")
	}
	if v, ok := m["dns"]; ok {
		d, ok := v.(map[string]any)
		if !ok {
			return nil, errors.New("dns 格式无效")
		}
		for k, v := range d {
			switch k {
			case "cache_enabled", "cache_size", "cache_ttl_min", "cache_ttl_max", "enable_dnssec":
				Section(base, "dns")[k] = v
			default:
				return nil, fmt.Errorf("dns.%s 不在可移植白名单", k)
			}
		}
	}
	if v, ok := m["user_rules"]; ok {
		a, ok := v.([]any)
		if !ok || len(a) > 10000 {
			return nil, errors.New("规则格式无效")
		}
		for _, x := range a {
			s, ok := x.(string)
			if !ok || len(s) > 4096 || strings.ContainsAny(s, "\x00\r\n") {
				return nil, errors.New("规则无效")
			}
		}
		base["user_rules"] = v
	}
	for _, k := range []string{"querylog", "statistics"} {
		if v, ok := m[k]; ok {
			a, ok := v.(map[string]any)
			if !ok {
				return nil, errors.New("设置格式无效")
			}
			for key, v := range a {
				if key != "enabled" && key != "file_enabled" && key != "interval" {
					return nil, fmt.Errorf("禁止路径或未知字段 %s.%s", k, key)
				}
				Section(base, k)[key] = v
			}
		}
	}
	return base, nil
}
