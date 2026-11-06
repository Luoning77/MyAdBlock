// SPDX-License-Identifier: GPL-3.0-or-later
package network

import (
	"encoding/binary"
	"errors"
	"net"
)

// Constants follow Linux UAPI nfnetlink_conntrack.h (v6.12); no global conntrack flush.
func align4(n int) int { return (n + 3) &^ 3 }
func attrs(b []byte) (map[uint16][]byte, error) {
	m := map[uint16][]byte{}
	for len(b) > 0 {
		if len(b) < 4 {
			return nil, errors.New("short netlink attribute")
		}
		n := int(binary.LittleEndian.Uint16(b))
		kind := binary.LittleEndian.Uint16(b[2:]) & 0x3fff
		if n < 4 || n > len(b) || align4(n) > len(b) {
			return nil, errors.New("invalid netlink attribute")
		}
		m[kind] = b[4:n]
		b = b[align4(n):]
	}
	return m, nil
}
func attr(kind uint16, p []byte) []byte {
	b := make([]byte, align4(len(p)+4))
	binary.LittleEndian.PutUint16(b, uint16(len(p)+4))
	binary.LittleEndian.PutUint16(b[2:], kind)
	copy(b[4:], p)
	return b
}

// ownedTuple selects only NAT sessions whose reply endpoint is our unique loopback DNS listener.
// Tuple and kernel ID are carried into DELETE to avoid deleting a reused tuple (ABA race).
func ownedTuple(payload []byte, port int) ([]byte, error) {
	if len(payload) < 4 {
		return nil, errors.New("short nfgenmsg")
	}
	m, e := attrs(payload[4:])
	if e != nil {
		return nil, e
	}
	orig, e := attrs(m[1])
	if e != nil {
		return nil, e
	}
	reply, e := attrs(m[2])
	if e != nil {
		return nil, e
	}
	op, e := attrs(orig[2])
	if e != nil {
		return nil, e
	}
	rp, e := attrs(reply[2])
	if e != nil {
		return nil, e
	}
	ip, e := attrs(reply[1])
	if e != nil {
		return nil, e
	}
	if len(op[1]) != 1 || len(op[3]) != 2 || len(rp[2]) != 2 {
		return nil, nil
	}
	proto := op[1][0]
	if proto != 6 && proto != 17 {
		return nil, nil
	}
	if binary.BigEndian.Uint16(op[3]) != 53 || int(binary.BigEndian.Uint16(rp[2])) != port {
		return nil, nil
	}
	src := ip[1]
	if payload[0] == 10 {
		src = ip[3]
	}
	if !net.IP(src).IsLoopback() {
		return nil, nil
	}
	if len(m[12]) != 4 {
		return nil, errors.New("conntrack id missing")
	}
	// Send full original tuple, exact ID, and original zone when present.
	out := append([]byte{}, payload[:4]...)
	out = append(out, attr(1|0x8000, m[1])...)
	out = append(out, attr(12, m[12])...)
	if zone, ok := m[18]; ok {
		out = append(out, attr(18, zone)...)
	}
	return out, nil
}
