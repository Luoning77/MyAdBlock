//go:build linux

package network

import (
	"github.com/miekg/dns"
	"net"
	"os"
	"os/exec"
	"syscall"
	"testing"
)

func TestCaptureChild(t *testing.T) {
	if os.Getenv("MYADBLOCK_CAPTURE_CHILD") != "1" {
		t.Skip("namespace child only")
	}
	if e := syscall.Setgroups([]int{3003}); e != nil {
		t.Fatal(e)
	}
	if e := syscall.Setgid(9999); e != nil {
		t.Fatal(e)
	}
	if e := syscall.Setuid(9999); e != nil {
		t.Fatal(e)
	}
	if !Capture("192.0.2.53:53") {
		t.Fatal("NAT canary failed")
	}
}
func TestIsolatedLinuxNATAndConntrack(t *testing.T) {
	if os.Getenv("MYADBLOCK_NETNS_TEST") != "1" {
		t.Skip("requires explicitly isolated Linux namespace")
	}
	a, _ := os.Readlink("/proc/self/ns/net")
	b, _ := os.Readlink("/proc/1/ns/net")
	if a == b || a == "" || os.Geteuid() != 0 {
		t.Fatal("refusing test outside isolated root network namespace")
	}
	run := Exec{}
	for _, args := range [][]string{{"link", "set", "lo", "up"}, {"link", "add", "dummy0", "type", "dummy"}, {"link", "set", "dummy0", "up"}, {"addr", "add", "198.51.100.2/24", "dev", "dummy0"}, {"route", "add", "default", "dev", "dummy0"}} {
		if _, e := run.Run("ip", args...); e != nil {
			t.Fatal(e)
		}
	}
	if e := ConntrackAvailable(); e != nil {
		t.Fatal(e)
	}
	pc, e := net.ListenPacket("udp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	port := pc.LocalAddr().(*net.UDPAddr).Port
	tcp, e := net.Listen("tcp", pc.LocalAddr().String())
	if e != nil {
		t.Fatal(e)
	}
	h := dns.HandlerFunc(func(w dns.ResponseWriter, q *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(q)
		a, _ := dns.NewRR(CanaryName + " 1 IN A " + CanaryAnswer)
		m.Answer = []dns.RR{a}
		_ = w.WriteMsg(m)
	})
	u := &dns.Server{PacketConn: pc, Handler: h}
	v := &dns.Server{Listener: tcp, Handler: h}
	go u.ActivateAndServe()
	go v.ActivateAndServe()
	defer u.Shutdown()
	defer v.Shutdown()
	r := Rules{Run: run}
	defer r.Remove()
	if e = r.Install("iptables", port); e != nil {
		t.Fatal(e)
	}
	child := func() error {
		c := exec.Command(os.Args[0], "-test.run=^TestCaptureChild$")
		c.Env = append(os.Environ(), "MYADBLOCK_CAPTURE_CHILD=1")
		out, e := c.CombinedOutput()
		if e != nil {
			t.Log(string(out))
		}
		return e
	}
	if e = child(); e != nil {
		t.Fatal(e)
	}
	entries, e := ctDump(syscall.AF_INET, port)
	if e != nil || len(entries) == 0 {
		t.Fatal("target DNS NAT tuple missing", e)
	}
	if e = r.Remove(); e != nil {
		t.Fatal(e)
	}
	if e = PurgeDNS(port); e != nil {
		t.Fatal(e)
	}
	entries, e = ctDump(syscall.AF_INET, port)
	if e != nil || len(entries) != 0 {
		t.Fatal("NAT tuple survived purge", e)
	}
	if child() == nil {
		t.Fatal("capture survived removal")
	}
}
