// SPDX-License-Identifier: GPL-3.0-or-later
package network

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const Chain = "MYADBLOCK_DNS"

type Runner interface {
	Run(string, ...string) (string, error)
}
type Exec struct{}

func (Exec) Run(n string, a ...string) (string, error) {
	ctx, c := context.WithTimeout(context.Background(), 4*time.Second)
	defer c()
	b, e := exec.CommandContext(ctx, n, a...).CombinedOutput()
	return string(b), e
}

type Rules struct {
	Run             Runner
	LockPath, Guard string
}

func (r Rules) lock() (func(), error) {
	if r.LockPath == "" {
		return func() {}, nil
	}
	f, e := os.OpenFile(r.LockPath, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); e != nil {
		f.Close()
		return nil, e
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
}
func (r Rules) call(bin string, args ...string) (string, error) {
	return r.Run.Run(bin, append([]string{"-w", "2", "-t", "nat"}, args...)...)
}
func (r Rules) RemoveFamily(bin string) error {
	unlock, e := r.lock()
	if e != nil {
		return e
	}
	defer unlock()
	return r.removeFamily(bin)
}
func (r Rules) removeFamily(bin string) error {
	// Exact jumps only. Never flush a shared chain or table.
	for i := 0; i < 16; i++ {
		if _, e := r.call(bin, "-C", "OUTPUT", "-j", Chain); e != nil {
			break
		}
		if _, e := r.call(bin, "-D", "OUTPUT", "-j", Chain); e != nil {
			return e
		}
	}
	if _, e := r.call(bin, "-S", Chain); e != nil {
		return nil
	}
	if _, e := r.call(bin, "-F", Chain); e != nil {
		return e
	}
	_, e := r.call(bin, "-X", Chain)
	return e
}
func (r Rules) Remove() error {
	e4 := r.RemoveFamily("iptables")
	e6 := r.RemoveFamily("ip6tables")
	return errors.Join(e4, e6)
}
func (r Rules) Conflict(bin string) (bool, error) {
	s, e := r.call(bin, "-S", "OUTPUT")
	if e != nil {
		return false, e
	}
	for _, l := range strings.Split(s, "\n") {
		if strings.HasPrefix(l, "-A OUTPUT ") && !strings.Contains(l, "-j "+Chain) {
			return true, nil
		}
	}
	return false, nil
}
func (r Rules) Installed(bin string) bool {
	_, e := r.call(bin, "-C", "OUTPUT", "-j", Chain)
	return e == nil
}
func (r Rules) Install(bin string, port int) error {
	unlock, e := r.lock()
	if e != nil {
		return e
	}
	defer unlock()
	if r.Guard != "" {
		if _, e := os.Stat(r.Guard); e == nil {
			return errors.New("应急暂停已启用")
		}
	}
	if port < 1024 || port > 65535 {
		return errors.New("非法 DNS 端口")
	}
	if c, e := r.Conflict(bin); e != nil || c {
		return errors.New("NAT OUTPUT 存在未知规则或能力不可用；保留原网络")
	}
	if r.Installed(bin) {
		return nil
	}
	if e := r.removeFamily(bin); e != nil {
		return e
	}
	if _, e := r.call(bin, "-N", Chain); e != nil {
		return e
	}
	ok := false
	defer func() {
		if !ok {
			_ = r.removeFamily(bin)
		}
	}()
	// AGH, bootstrap and probes all run as root; UID exemption prevents recursion even with plaintext system DNS.
	if _, e := r.call(bin, "-A", Chain, "-m", "owner", "--uid-owner", "0", "-j", "RETURN"); e != nil {
		return e
	}
	for _, p := range []string{"udp", "tcp"} {
		if _, e := r.call(bin, "-A", Chain, "-p", p, "--dport", "53", "-j", "REDIRECT", "--to-ports", strconv.Itoa(port)); e != nil {
			return e
		}
	}
	if _, e := r.call(bin, "-A", "OUTPUT", "-j", Chain); e != nil {
		return fmt.Errorf("接管跳转失败: %w", e)
	}
	ok = true
	return nil
}
