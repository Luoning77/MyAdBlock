package network

import (
	"context"
	"github.com/miekg/dns"
	"net"
	"testing"
	"time"
)

func TestBlockedFirstBootstrapDoesNotHideWorkingSecond(t *testing.T) {
	ctx, c := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer c()
	ips, _, e := resolveUncached(ctx, "doh.example", []string{"192.0.2.1", "192.0.2.2"}, func(ctx context.Context, q *dns.Msg, address string) (*dns.Msg, error) {
		if address == "192.0.2.1:53" {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		m := new(dns.Msg)
		m.SetReply(q)
		if q.Question[0].Qtype == dns.TypeA {
			rr, _ := dns.NewRR("doh.example. 30 IN A 203.0.113.7")
			m.Answer = []dns.RR{rr}
		}
		return m, nil
	})
	if e != nil || len(ips) != 1 || !ips[0].Equal(net.ParseIP("203.0.113.7")) {
		t.Fatal(ips, e)
	}
}
func TestBootstrapMergeExcludesRecursiveAddresses(t *testing.T) {
	v := MergeBootstrap([]string{"127.0.0.1", "::1", "223.5.5.5"}, []string{"192.168.1.1", "223.5.5.5"})
	if len(v) != 2 || v[1] != "192.168.1.1" {
		t.Fatal(v)
	}
}
