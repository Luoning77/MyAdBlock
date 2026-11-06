package network

import (
	"errors"
	"strings"
	"testing"
)

type fake struct {
	chains   map[string]bool
	jump     map[string]bool
	log      []string
	fail     string
	conflict bool
}

func (f *fake) Run(bin string, a ...string) (string, error) {
	s := strings.Join(a, " ")
	f.log = append(f.log, bin+" "+s)
	if f.fail != "" && strings.Contains(s, f.fail) {
		return "", errors.New("fault")
	}
	s = strings.TrimPrefix(s, "-w 2 -t nat ")
	switch {
	case s == "-S OUTPUT":
		if f.conflict {
			return "-A OUTPUT -j MIHOMO", nil
		}
		return "", nil
	case s == "-S "+Chain:
		if !f.chains[bin] {
			return "", errors.New("missing")
		}
	case s == "-C OUTPUT -j "+Chain:
		if !f.jump[bin] {
			return "", errors.New("missing")
		}
	case s == "-N "+Chain:
		f.chains[bin] = true
	case s == "-X "+Chain:
		delete(f.chains, bin)
	case s == "-D OUTPUT -j "+Chain:
		delete(f.jump, bin)
	case s == "-A OUTPUT -j "+Chain:
		f.jump[bin] = true
	}
	return "", nil
}
func newFake() *fake { return &fake{chains: map[string]bool{}, jump: map[string]bool{}} }
func TestRulesIdempotent(t *testing.T) {
	f := newFake()
	r := Rules{Run: f}
	for i := 0; i < 3; i++ {
		if e := r.Install("iptables", 33445); e != nil {
			t.Fatal(e)
		}
	}
	n := 0
	for _, l := range f.log {
		if strings.Contains(l, "-A OUTPUT -j") {
			n++
		}
		if strings.Contains(l, "DROP") || strings.Contains(l, "-F OUTPUT") {
			t.Fatal(l)
		}
	}
	if n != 1 {
		t.Fatal(n)
	}
	if e := r.Remove(); e != nil {
		t.Fatal(e)
	}
	if e := r.Remove(); e != nil {
		t.Fatal(e)
	}
	if len(f.jump) > 0 {
		t.Fatal(f.jump)
	}
}
func TestFaultRollsBackOnlyOwnChain(t *testing.T) {
	f := newFake()
	f.fail = "-p tcp"
	r := Rules{Run: f}
	if r.Install("ip6tables", 34567) == nil {
		t.Fatal("fault ignored")
	}
	if len(f.jump) > 0 || len(f.chains) > 0 {
		t.Fatal("rollback failed")
	}
}
func TestProxyRulesKept(t *testing.T) {
	f := newFake()
	f.conflict = true
	r := Rules{Run: f}
	if r.Install("iptables", 12345) == nil {
		t.Fatal("must bypass")
	}
	for _, l := range f.log {
		if strings.Contains(l, "-F") || strings.Contains(l, "-D") {
			t.Fatal(l)
		}
	}
}
