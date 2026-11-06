//go:build linux

// SPDX-License-Identifier: GPL-3.0-or-later
package network

import (
	"encoding/binary"
	"errors"
	"fmt"
	"syscall"
	"time"
)

func ctSocket() (int, error) {
	fd, e := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW|syscall.SOCK_CLOEXEC, 12)
	if e != nil {
		return -1, e
	}
	if e = syscall.Bind(fd, &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK}); e != nil {
		syscall.Close(fd)
		return -1, e
	}
	tv := syscall.NsecToTimeval((2 * time.Second).Nanoseconds())
	if e = syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv); e != nil {
		syscall.Close(fd)
		return -1, e
	}
	return fd, nil
}
func ctSend(fd int, op, flags uint16, seq uint32, payload []byte) error {
	b := make([]byte, 16+len(payload))
	binary.LittleEndian.PutUint32(b, uint32(len(b)))
	binary.LittleEndian.PutUint16(b[4:], 0x100|op)
	binary.LittleEndian.PutUint16(b[6:], flags)
	binary.LittleEndian.PutUint32(b[8:], seq)
	copy(b[16:], payload)
	return syscall.Sendto(fd, b, 0, &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK})
}
func ctRead(fd int, seq uint32, visit func([]byte) error) error {
	buf := make([]byte, 65536)
	deadline := time.Now().Add(3 * time.Second)
	total := 0
	for time.Now().Before(deadline) {
		n, _, flags, from, e := syscall.Recvmsg(fd, buf, nil, 0)
		if e != nil {
			return e
		}
		if flags&syscall.MSG_TRUNC != 0 {
			return errors.New("conntrack dump truncated")
		}
		peer, ok := from.(*syscall.SockaddrNetlink)
		if !ok || peer.Pid != 0 {
			return errors.New("non-kernel netlink sender")
		}
		total += n
		if total > 16<<20 {
			return errors.New("conntrack dump too large; bypass")
		}
		msgs, e := syscall.ParseNetlinkMessage(buf[:n])
		if e != nil {
			return e
		}
		for _, m := range msgs {
			if m.Header.Seq != seq {
				continue
			}
			if m.Header.Flags&0x10 != 0 {
				return errors.New("conntrack dump interrupted")
			}
			switch m.Header.Type {
			case syscall.NLMSG_DONE:
				if len(m.Data) >= 4 && int32(binary.LittleEndian.Uint32(m.Data)) != 0 {
					return errors.New("conntrack dump failed")
				}
				return nil
			case syscall.NLMSG_ERROR:
				if len(m.Data) < 4 {
					return errors.New("short netlink ack")
				}
				errno := int32(binary.LittleEndian.Uint32(m.Data))
				if errno == 0 {
					return nil
				}
				return syscall.Errno(-errno)
			default:
				if visit != nil {
					if e = visit(m.Data); e != nil {
						return e
					}
				}
			}
		}
	}
	return errors.New("conntrack read deadline")
}
func ctDump(family byte, port int) ([][]byte, error) {
	fd, e := ctSocket()
	if e != nil {
		return nil, e
	}
	defer syscall.Close(fd)
	if e = ctSend(fd, 1, 1|0x300, 1, []byte{family, 0, 0, 0}); e != nil {
		return nil, e
	}
	found := [][]byte{}
	e = ctRead(fd, 1, func(p []byte) error {
		if port == 0 {
			return nil
		}
		v, e := ownedTuple(p, port)
		if e != nil {
			return e
		}
		if v != nil {
			found = append(found, v)
		}
		if len(found) > 4096 {
			return errors.New("too many module DNS sessions")
		}
		return nil
	})
	return found, e
}
func ConntrackAvailable() error { _, e := ctDump(syscall.AF_INET, 0); return e }
func PurgeDNS(port int) error {
	if port < 1024 || port > 65535 {
		return errors.New("invalid conntrack DNS port")
	}
	var errs []error
	for _, family := range []byte{syscall.AF_INET, syscall.AF_INET6} {
		entries, e := ctDump(family, port)
		if e != nil {
			if family == syscall.AF_INET6 && (e == syscall.EAFNOSUPPORT || e == syscall.EPROTONOSUPPORT) {
				continue
			}
			errs = append(errs, e)
			continue
		}
		for _, payload := range entries {
			fd, e := ctSocket()
			if e != nil {
				errs = append(errs, e)
				break
			}
			e = ctSend(fd, 2, 1|4, 2, payload)
			if e == nil {
				e = ctRead(fd, 2, nil)
			}
			syscall.Close(fd)
			if e != nil && e != syscall.ENOENT {
				errs = append(errs, fmt.Errorf("targeted DNS conntrack cleanup: %w", e))
			}
		}
	}
	return errors.Join(errs...)
}
