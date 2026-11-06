// SPDX-License-Identifier: GPL-3.0-or-later
package network

import (
	"context"
	"errors"
	"github.com/miekg/dns"
	"net"
	"strings"
	"sync"
	"time"
)

type cachedIPs struct {
	ips   []net.IP
	until time.Time
}

var hostCache = struct {
	sync.Mutex
	entries map[string]cachedIPs
}{entries: map[string]cachedIPs{}}

func MergeBootstrap(lists ...[]string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, list := range lists {
		for _, s := range list {
			ip := net.ParseIP(s)
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() {
				continue
			}
			s = ip.String()
			if !seen[s] && len(out) < 12 {
				out = append(out, s)
				seen[s] = true
			}
		}
	}
	return out
}

type bootstrapExchange func(context.Context, *dns.Msg, string) (*dns.Msg, error)

func exchangeBootstrap(ctx context.Context, q *dns.Msg, address string) (*dns.Msg, error) {
	c := &dns.Client{Net: "udp", Timeout: time.Second}
	m, _, e := c.ExchangeContext(ctx, q, address)
	if e == nil && m.Truncated {
		c.Net = "tcp"
		m, _, e = c.ExchangeContext(ctx, q, address)
	}
	return m, e
}

// Query every configured bootstrap, not just the first successful UDP connect (which proves no DNS health).
func resolveUncached(ctx context.Context, host string, bootstrap []string, exchange bootstrapExchange) ([]net.IP, time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	type result struct {
		q, m *dns.Msg
		e    error
	}
	ch := make(chan result, len(bootstrap)*2)
	var wg sync.WaitGroup
	for _, ip := range bootstrap {
		for _, typ := range []uint16{dns.TypeA, dns.TypeAAAA} {
			q := Query()
			q.Question[0].Name = dns.Fqdn(host)
			q.Question[0].Qtype = typ
			wg.Add(1)
			go func(q *dns.Msg, address string) {
				defer wg.Done()
				m, e := exchange(ctx, q, address)
				ch <- result{q, m, e}
			}(q, net.JoinHostPort(ip, "53"))
		}
	}
	go func() { wg.Wait(); close(ch) }()
	ips := []net.IP{}
	seen := map[string]bool{}
	ttl := 60 * time.Second
	finish := func() ([]net.IP, time.Duration, error) {
		if len(ips) == 0 {
			return nil, 0, errors.New("所有 bootstrap 均未返回可用地址")
		}
		return ips, ttl, nil
	}
	for {
		select {
		case <-ctx.Done():
			return finish()
		case v, ok := <-ch:
			if !ok {
				return finish()
			}
			if v.e != nil || !ValidReply(v.q, v.m) {
				continue
			}
			for _, rr := range v.m.Answer {
				var ip net.IP
				switch a := rr.(type) {
				case *dns.A:
					ip = a.A
				case *dns.AAAA:
					ip = a.AAAA
				}
				if ip == nil || seen[ip.String()] || len(ips) >= 32 {
					continue
				}
				seen[ip.String()] = true
				ips = append(ips, append(net.IP{}, ip...))
				if t := time.Duration(rr.Header().Ttl) * time.Second; t < ttl {
					ttl = t
				}
			}
		}
	}
}
func resolveIPs(ctx context.Context, host string, bootstrap []string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	key := host + "|" + strings.Join(bootstrap, ",")
	hostCache.Lock()
	saved, ok := hostCache.entries[key]
	hostCache.Unlock()
	if ok && time.Now().Before(saved.until) {
		return saved.ips, nil
	}
	ips, ttl, e := resolveUncached(ctx, host, bootstrap, exchangeBootstrap)
	if e == nil {
		hostCache.Lock()
		if len(hostCache.entries) >= 32 {
			hostCache.entries = map[string]cachedIPs{}
		}
		hostCache.entries[key] = cachedIPs{ips, time.Now().Add(ttl)}
		hostCache.Unlock()
	}
	return ips, e
}
func forgetIPs(host string, bootstrap []string) {
	hostCache.Lock()
	delete(hostCache.entries, host+"|"+strings.Join(bootstrap, ","))
	hostCache.Unlock()
}
func dialIPs(ctx context.Context, network, port string, ips []net.IP) (net.Conn, error) {
	if len(ips) == 0 {
		return nil, errors.New("无可用地址")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	winner := make(chan net.Conn)
	failures := make(chan error, len(ips))
	for _, ip := range ips {
		go func(ip net.IP) {
			c, e := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if e != nil {
				failures <- e
				return
			}
			select {
			case winner <- c:
			case <-ctx.Done():
				c.Close()
			}
		}(ip)
	}
	var last error
	for range ips {
		select {
		case c := <-winner:
			return c, nil
		case last = <-failures:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, last
}
