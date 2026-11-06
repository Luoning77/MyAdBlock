package engine

import (
	"fmt"
	"github.com/Luoning77/MyAdBlock/internal/agh"
	"github.com/Luoning77/MyAdBlock/internal/config"
	"github.com/Luoning77/MyAdBlock/internal/network"
	"github.com/miekg/dns"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestOfficialAGHIntegration(t *testing.T) {
	bin := os.Getenv("AGH_TEST_BINARY")
	if bin == "" {
		t.Skip("set AGH_TEST_BINARY for real pinned-core integration")
	}
	module := t.TempDir()
	data := t.TempDir()
	for i := 1; i <= 3; i++ {
		p := filepath.Join(module, "presets/snapshots", fmt.Sprintf("%d.txt", i))
		if e := config.Atomic(p, []byte("! local integration fixture\n||ads.myadblock.test^\n")); e != nil {
			t.Fatal(e)
		}
	}
	r, e := New(module, data, bin, true)
	if e != nil {
		m := config.Fresh(data)
		_ = config.Normalize(m, data, 23456, 34567, false, "test-secret")
		_ = config.Write(filepath.Join(data, "debug.yaml"), m)
		out, _ := exec.Command(bin, "--check-config", "-c", filepath.Join(data, "debug.yaml"), "-w", filepath.Join(data, "agh-data")).CombinedOutput()
		t.Log(string(out))
		t.Fatal(e)
	}
	defer r.Shutdown()
	if e = r.StartCore(); e != nil {
		t.Fatal(e)
	}
	ready := false
	for i := 0; i < 100; i++ {
		if r.AGH.Call("GET", "/status", nil, &map[string]any{}) == nil {
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatal("AGH not ready")
	}
	if !network.Capture(fmt.Sprintf("127.0.0.1:%d", r.Ports.DNS)) {
		t.Fatal("managed UDP/TCP canary rewrite not ready")
	}
	// Local authoritative test upstream: no external network involved.
	pc, e := net.ListenPacket("udp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, q *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(q)
		rr, _ := dns.NewRR(q.Question[0].Name + " 1 IN A 192.0.2.7")
		m.Answer = []dns.RR{rr}
		_ = w.WriteMsg(m)
	})
	srv := &dns.Server{PacketConn: pc, Handler: handler}
	go srv.ActivateAndServe()
	defer srv.Shutdown()
	if e = r.AGH.Upstreams([]string{pc.LocalAddr().String()}, []string{"1.1.1.1"}, "load_balance"); e != nil {
		t.Fatal(e)
	}
	query := func() bool {
		q := new(dns.Msg)
		q.SetQuestion("ads.myadblock.test.", dns.TypeA)
		m, _, e := (&dns.Client{Timeout: 2 * time.Second}).Exchange(q, fmt.Sprintf("127.0.0.1:%d", r.Ports.DNS))
		if e != nil {
			t.Fatal(e)
		}
		for _, rr := range m.Answer {
			if a, ok := rr.(*dns.A); ok && a.A.String() == "192.0.2.7" {
				return false
			}
		}
		return true
	}
	if !query() {
		f, _ := r.Filters()
		t.Logf("filters: %+v", f)
		var d map[string]any
		_ = r.AGH.Call("GET", "/dns_info", nil, &d)
		t.Logf("dns protection %v", d["protection_enabled"])
		t.Fatal("offline filter not loaded")
	}
	if e = r.Pause(300); e != nil {
		t.Fatal(e)
	}
	if query() {
		t.Fatal("protection still on during pause")
	}
	if r.M.State != Paused {
		t.Fatal(r.M)
	}
	if e = r.Resume(); e != nil {
		t.Fatal(e)
	}
	if e = r.AGH.Protection(true); e != nil {
		t.Fatal(e)
	}
	if !query() {
		t.Fatal("filter did not resume")
	}
	if e = r.UserRules([]string{"@@||keep.example^"}); e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"strict", "daily"} {
		if e = r.Mode(mode); e != nil {
			t.Fatal(e)
		}
		f, e := r.Filters()
		if e != nil {
			t.Fatal(e)
		}
		if len(f.Rules) != 1 || f.Rules[0] != "@@||keep.example^" {
			t.Fatal("user rules lost")
		}
		on := 0
		for _, f := range f.Filters {
			if (f.ID == 2 || f.ID == 3) && f.Enabled {
				on++
			}
		}
		if on != 1 {
			t.Fatal("AWAvenue mutual exclusion", f)
		}
	}
	root := agh.New(r.Ports.AGH, "root", "root")
	if e = root.Call("GET", "/profile", nil, &map[string]any{}); e != nil {
		t.Fatal("root/root login", e)
	}
	if e = r.ChangePassword("custom-test-password"); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 100; i++ {
		if r.AGH.Call("GET", "/profile", nil, &map[string]any{}) == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if e = r.AGH.Protection(false); e != nil {
		t.Fatal("management failed after password change", e)
	}
	if e = r.Import([]byte("schema_version: 34\nhttp:\n address: 0.0.0.0:80")); e == nil {
		t.Fatal("unsafe YAML accepted")
	}
	custom := agh.New(r.Ports.AGH, "root", "custom-test-password")
	for i := 0; i < 50; i++ {
		if r.AGH.Call("GET", "/profile", nil, &map[string]any{}) == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if e = custom.Call("GET", "/profile", nil, &map[string]any{}); e != nil {
		t.Fatal("original credentials not preserved after rejected import", e)
	}
	if root.Call("GET", "/profile", nil, &map[string]any{}) == nil {
		t.Fatal("old password still accepted")
	}
	t.Log("Official AGH: initialization bypass, offline filtering, protection API, preset exclusivity, user-rule preservation, credential isolation, import rejection passed")
}
