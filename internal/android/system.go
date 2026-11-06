// SPDX-License-Identifier: GPL-3.0-or-later
package android

import (
	"encoding/json"
	"errors"
	"github.com/Luoning77/MyAdBlock/internal/config"
	"github.com/Luoning77/MyAdBlock/internal/network"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Link struct {
	Reliable  bool     `json:"reliable"`
	VPN       bool     `json:"vpn"`
	Interface string   `json:"interface"`
	DNS       []string `json:"dns"`
	Proxy     []string `json:"proxy"`
}

var ifaceRE = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,32}$`)

func Discover(run network.Runner, module string) Link {
	var l Link
	// CLASSPATH is inherited by app_process without a shell or interpolated input.
	os.Setenv("CLASSPATH", filepath.Join(module, "lib/network-probe.jar"))
	s, e := run.Run("/system/bin/app_process", "/system/bin", "io.myadblock.NetworkProbe")
	if e != nil || json.Unmarshal([]byte(s), &l) != nil {
		return Link{}
	}
	if !ifaceRE.MatchString(l.Interface) || strings.HasPrefix(l.Interface, "tun") || strings.HasPrefix(l.Interface, "wg") {
		l.Reliable = false
	}
	out := []string{}
	seen := map[string]bool{}
	for _, s := range l.DNS {
		ip := net.ParseIP(s)
		if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.Equal(net.ParseIP("198.18.0.2")) || ip.Equal(net.ParseIP("10.0.0.2")) {
			continue
		}
		s = ip.String()
		if !seen[s] {
			out = append(out, s)
			seen[s] = true
		}
	}
	if len(out) > 4 {
		out = out[:4]
	}
	l.DNS = out
	if len(out) == 0 {
		l.Reliable = false
	}
	ps, _ := run.Run("/system/bin/ps", "-A", "-o", "NAME")
	low := strings.ToLower(ps)
	for _, p := range []string{"flclash", "mihomo", "clash", "sing-box", "box for root"} {
		if strings.Contains(low, p) {
			l.Proxy = append(l.Proxy, p)
		}
	}
	// Any TUN/WG interface is ambiguous: keep the VPN's route untouched.
	if entries, e := os.ReadDir("/sys/class/net"); e == nil {
		for _, en := range entries {
			if strings.HasPrefix(en.Name(), "tun") || strings.HasPrefix(en.Name(), "wg") {
				l.VPN = true
			}
		}
	}
	return l
}

type Field struct {
	Before   string `json:"before"`
	Written  string `json:"written"`
	Modified bool   `json:"modified"`
	Applied  bool   `json:"applied"`
}
type PrivateDNS struct {
	Fields      map[string]Field `json:"fields"`
	Initialized bool             `json:"initialized"`
}

func (p *PrivateDNS) Disable(run network.Runner, path string) error {
	if e := config.Read(path, p); e != nil && !os.IsNotExist(e) {
		return e
	}
	if p.Initialized {
		return nil
	}
	p.Fields = map[string]Field{}
	// Back up BOTH real fields. Only mode needs changing; server name stays untouched.
	for _, k := range []string{"private_dns_mode", "private_dns_specifier"} {
		s, e := run.Run("/system/bin/settings", "get", "global", k)
		if e != nil {
			return errors.New("无法备份私人 DNS")
		}
		p.Fields[k] = Field{Before: strings.TrimSpace(s)}
	}
	f := p.Fields["private_dns_mode"]
	f.Written = "off"
	f.Modified = f.Before != "off"
	p.Fields["private_dns_mode"] = f
	p.Initialized = true
	// Journal before mutation; a crash is recoverable by compare-and-restore.
	if e := config.Save(path, p); e != nil {
		return e
	}
	if f.Modified {
		if _, e := run.Run("/system/bin/settings", "put", "global", "private_dns_mode", "off"); e != nil {
			return e
		}
	}
	s, e := run.Run("/system/bin/settings", "get", "global", "private_dns_mode")
	if e != nil || strings.TrimSpace(s) != "off" {
		return errors.New("私人 DNS 关闭失败")
	}
	if f.Modified {
		f.Applied = true
		p.Fields["private_dns_mode"] = f
		return config.Save(path, p)
	}
	return nil
}
func (p *PrivateDNS) Conflict(run network.Runner) bool {
	if !p.Initialized {
		return false
	}
	s, e := run.Run("/system/bin/settings", "get", "global", "private_dns_mode")
	return e != nil || strings.TrimSpace(s) != "off"
}
func Restore(run network.Runner, path string) error {
	var p PrivateDNS
	if e := config.Read(path, &p); e != nil {
		if os.IsNotExist(e) {
			return nil
		}
		return e
	}
	for k, f := range p.Fields {
		if !f.Modified || !f.Applied {
			continue
		}
		s, e := run.Run("/system/bin/settings", "get", "global", k)
		if e != nil {
			return e
		}
		if strings.TrimSpace(s) != f.Written {
			continue
		}
		a := []string{"put", "global", k, f.Before}
		if f.Before == "null" {
			a = []string{"delete", "global", k}
		}
		if _, e = run.Run("/system/bin/settings", a...); e != nil {
			return e
		}
	}
	return nil
}
