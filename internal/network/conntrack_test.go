package network

import (
	"encoding/binary"
	"testing"
)

func fixture(port int, ip []byte, proto byte, origPort int) []byte {
	p := func(n int) []byte { b := make([]byte, 2); binary.BigEndian.PutUint16(b, uint16(n)); return b }
	o := attr(1, attr(1, []byte{192, 0, 2, 2}))
	o = append(o, attr(2, append(append(attr(1, []byte{proto}), attr(2, p(45678))...), attr(3, p(origPort))...))...)
	r := attr(1, attr(1, ip))
	r = append(r, attr(2, append(append(attr(1, []byte{proto}), attr(2, p(port))...), attr(3, p(45678))...))...)
	b := []byte{2, 0, 0, 0}
	b = append(b, attr(1, o)...)
	b = append(b, attr(2, r)...)
	return append(b, attr(12, []byte{0, 0, 0, 9})...)
}
func TestConntrackTargetsOnlyOwnDNS(t *testing.T) {
	for _, c := range []struct {
		port  int
		ip    []byte
		proto byte
		orig  int
		want  bool
	}{{33333, []byte{127, 0, 0, 1}, 17, 53, true}, {33333, []byte{127, 0, 0, 1}, 6, 53, true}, {33334, []byte{127, 0, 0, 1}, 17, 53, false}, {33333, []byte{1, 1, 1, 1}, 17, 53, false}, {33333, []byte{127, 0, 0, 1}, 17, 443, false}, {33333, []byte{127, 0, 0, 1}, 1, 53, false}} {
		v, e := ownedTuple(fixture(c.port, c.ip, c.proto, c.orig), 33333)
		if e != nil {
			t.Fatal(e)
		}
		if (v != nil) != c.want {
			t.Fatal(c)
		}
	}
}
func TestMalformedConntrack(t *testing.T) {
	for _, b := range [][]byte{nil, {2, 0}, {2, 0, 0, 0, 255, 255, 1, 0}} {
		if _, e := ownedTuple(b, 33333); e == nil {
			t.Fatal("malformed accepted")
		}
	}
}
