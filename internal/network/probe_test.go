package network

import (
	"github.com/miekg/dns"
	"os"
	"testing"
)

func TestNXDOMAINIsHealthy(t *testing.T) {
	q := Query()
	r := new(dns.Msg)
	r.SetRcode(q, dns.RcodeNameError)
	if !ValidReply(q, r) {
		t.Fatal("NXDOMAIN is valid DNS")
	}
	r.Rcode = dns.RcodeServerFailure
	if ValidReply(q, r) {
		t.Fatal("SERVFAIL")
	}
	r.SetRcode(q, dns.RcodeSuccess)
	r.Id++
	if ValidReply(q, r) {
		t.Fatal("mismatched ID")
	}
}
func TestRejectBadSnapshots(t *testing.T) {
	for _, b := range []string{"", "<html>login</html>", "! only comments\n"} {
		if ValidateRules([]byte(b)) == nil {
			t.Fatal(b)
		}
	}
	good := ""
	for i := 0; i < 20; i++ {
		good += "||ads.example.com^\n"
	}
	if e := ValidateRules([]byte(good)); e != nil {
		t.Fatal(e)
	}
}
func TestSafeURLs(t *testing.T) {
	for _, s := range []string{"file:///etc/passwd", "http://example.com", "https://root:root@example.com/", "https://example.com:853/"} {
		if SafeHTTPS(s) == nil {
			t.Fatal(s)
		}
	}
}

func TestBundledSnapshotsValid(t *testing.T) {
	for _, n := range []string{"1", "2", "3"} {
		b, e := os.ReadFile("../../presets/snapshots/" + n + ".txt")
		if e != nil {
			t.Fatal(e)
		}
		if e = ValidateRules(b); e != nil {
			t.Fatalf("snapshot %s: %v", n, e)
		}
	}
}
