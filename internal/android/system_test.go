package android

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

type settings struct {
	v      map[string]string
	writes []string
}

func (s *settings) Run(_ string, a ...string) (string, error) {
	if a[0] == "get" {
		if v, ok := s.v[a[2]]; ok {
			return v + "\n", nil
		}
		return "null\n", nil
	}
	s.writes = append(s.writes, strings.Join(a, " "))
	if a[0] == "delete" {
		delete(s.v, a[2])
	} else {
		s.v[a[2]] = a[3]
	}
	return "", nil
}
func TestPrivateDNSConditionalRestore(t *testing.T) {
	for _, changed := range []bool{false, true} {
		s := &settings{v: map[string]string{"private_dns_mode": "hostname", "private_dns_specifier": "dns.example"}}
		path := filepath.Join(t.TempDir(), "private.json")
		p := PrivateDNS{}
		if e := p.Disable(s, path); e != nil {
			t.Fatal(e)
		}
		if s.v["private_dns_mode"] != "off" {
			t.Fatal(s)
		}
		if changed {
			s.v["private_dns_mode"] = "opportunistic"
		}
		if e := p.Disable(s, path); e != nil {
			t.Fatal(e)
		}
		if len(s.writes) != 1 {
			t.Fatal("repeated settings overwrite")
		}
		if e := Restore(s, path); e != nil {
			t.Fatal(e)
		}
		want := "hostname"
		if changed {
			want = "opportunistic"
		}
		if s.v["private_dns_mode"] != want || s.v["private_dns_specifier"] != "dns.example" {
			t.Fatal(s)
		}
	}
}
func TestMissingSettingRestoredByDelete(t *testing.T) {
	s := &settings{v: map[string]string{}}
	path := filepath.Join(t.TempDir(), "state.json")
	p := PrivateDNS{}
	if e := p.Disable(s, path); e != nil {
		t.Fatal(e)
	}
	if e := Restore(s, path); e != nil {
		t.Fatal(e)
	}
	if _, ok := s.v["private_dns_mode"]; ok {
		t.Fatal(s)
	}
}

type failedWrite struct{ *settings }

func (s failedWrite) Run(n string, a ...string) (string, error) {
	if a[0] == "put" {
		return "", fmt.Errorf("permission denied")
	}
	return s.settings.Run(n, a...)
}
func TestFailedPrivateWriteNeverClaimsLaterUserValue(t *testing.T) {
	s := &settings{v: map[string]string{"private_dns_mode": "hostname", "private_dns_specifier": "dns.example"}}
	path := filepath.Join(t.TempDir(), "state.json")
	p := PrivateDNS{}
	if p.Disable(failedWrite{s}, path) == nil {
		t.Fatal("write failure ignored")
	}
	s.v["private_dns_mode"] = "off"
	if e := Restore(s, path); e != nil {
		t.Fatal(e)
	}
	if s.v["private_dns_mode"] != "off" || len(s.writes) != 0 {
		t.Fatal("restored setting that this module never changed")
	}
}
