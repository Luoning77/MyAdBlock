// SPDX-License-Identifier: GPL-3.0-or-later
package network

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"github.com/miekg/dns"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const ProbeName = "example.com."

func Query() *dns.Msg {
	m := new(dns.Msg)
	m.SetQuestion(ProbeName, dns.TypeA)
	var b [2]byte
	if _, e := rand.Read(b[:]); e == nil {
		m.Id = binary.BigEndian.Uint16(b[:])
	}
	return m
}
func ValidReply(q, r *dns.Msg) bool {
	return r != nil && r.Response && r.Id == q.Id && len(r.Question) == 1 && strings.EqualFold(r.Question[0].Name, q.Question[0].Name) && r.Question[0].Qtype == q.Question[0].Qtype && (r.Rcode == dns.RcodeSuccess || r.Rcode == dns.RcodeNameError)
}
func DNS(addr string, tcp bool) bool {
	q := Query()
	proto := "udp"
	if tcp {
		proto = "tcp"
	}
	c := &dns.Client{Net: proto, Timeout: 2 * time.Second}
	r, _, e := c.Exchange(q, addr)
	return e == nil && ValidReply(q, r)
}

// Probe resolves DoH endpoints with an explicit bootstrap resolver, independent of AGH and Android DNS capture.
func DoH(ctx context.Context, endpoint string, bootstrap []string) bool {
	u, e := url.Parse(endpoint)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || len(bootstrap) == 0 {
		return false
	}
	tr := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, n, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil {
			return nil, e
		}
		ips, e := resolveIPs(ctx, host, bootstrap)
		if e != nil {
			return nil, e
		}
		return dialIPs(ctx, n, port, ips)
	}, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, DisableKeepAlives: true}
	defer tr.CloseIdleConnections()
	q := Query()
	b, e := q.Pack()
	if e != nil {
		return false
	}
	req, e := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(b))
	if e != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")
	resp, e := (&http.Client{Transport: tr, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("DoH redirect denied") }}).Do(req)
	if e != nil {
		forgetIPs(u.Hostname(), bootstrap)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	b, e = io.ReadAll(io.LimitReader(resp.Body, 65536))
	if e != nil {
		return false
	}
	r := new(dns.Msg)
	return r.Unpack(b) == nil && ValidReply(q, r)
}
func SafeHTTPS(s string) error {
	u, e := url.Parse(s)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") || len(s) > 2048 {
		return errors.New("仅允许无凭据 HTTPS URL（443）")
	}
	return nil
}

// SnapshotHTTP rejects redirects and private/link-local IPs at actual dial time to prevent DNS rebinding / SSRF.
func SnapshotHTTP(bootstrap []string) *http.Client {
	return &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("订阅重定向被拒绝") }, Transport: &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, n, a string) (net.Conn, error) {
		h, p, e := net.SplitHostPort(a)
		if e != nil {
			return nil, e
		}
		ips, e := resolveIPs(ctx, h, bootstrap)
		if e != nil {
			return nil, e
		}
		for _, ip := range ips {
			if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() {
				continue
			}
			c, e := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, n, net.JoinHostPort(ip.String(), p))
			if e == nil {
				return c, nil
			}
		}
		return nil, errors.New("无可用公网订阅地址")
	}}}
}

var plainAdPattern = regexp.MustCompile(`^[-A-Za-z0-9_*]+$`)

func ValidateRules(b []byte) error {
	if len(b) < 100 || len(b) > 20<<20 || bytes.Contains(b, []byte{0}) || bytes.Contains(bytes.ToLower(b), []byte("<html")) {
		return errors.New("订阅为空、超限或 HTML")
	}
	rules := 0
	for _, s := range strings.Split(string(b), "\n") {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "!") || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "[") {
			continue
		}
		if strings.Contains(s, "||") || strings.HasPrefix(s, "0.0.0.0 ") || strings.HasPrefix(s, "127.0.0.1 ") || (strings.HasPrefix(s, "-") && strings.HasSuffix(s, "-") && strings.Trim(s, "-*_ ") != "" && plainAdPattern.MatchString(s)) || (!strings.ContainsAny(s, " <>\t") && strings.Contains(s, ".")) {
			rules++
		} else {
			return errors.New("包含无法识别的 DNS 规则")
		}
	}
	if rules < 10 {
		return errors.New("有效规则过少")
	}
	return nil
}

const CanaryName = "myadblock-health.invalid."
const CanaryAnswer = "192.0.2.173"

func Capture(addr string) bool {
	for _, proto := range []string{"udp", "tcp"} {
		q := new(dns.Msg)
		q.SetQuestion(CanaryName, dns.TypeA)
		m, _, e := (&dns.Client{Net: proto, Timeout: 2 * time.Second}).Exchange(q, addr)
		if e != nil {
			return false
		}
		found := false
		for _, rr := range m.Answer {
			if a, ok := rr.(*dns.A); ok && a.A.String() == CanaryAnswer {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
