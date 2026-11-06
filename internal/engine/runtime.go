// SPDX-License-Identifier: GPL-3.0-or-later
package engine

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/Luoning77/MyAdBlock/internal/agh"
	"github.com/Luoning77/MyAdBlock/internal/android"
	"github.com/Luoning77/MyAdBlock/internal/config"
	"github.com/Luoning77/MyAdBlock/internal/network"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Ports struct {
	Boot string `json:"boot"`
	UI   int    `json:"ui"`
	AGH  int    `json:"agh"`
	DNS  int    `json:"dns"`
}
type Status struct {
	State              State             `json:"state"`
	Remaining          int               `json:"remaining"`
	Mode               string            `json:"mode"`
	Core               bool              `json:"core"`
	Protection         bool              `json:"protection"`
	IPv4               bool              `json:"ipv4"`
	IPv6               bool              `json:"ipv6"`
	IPv6Available      bool              `json:"ipv6Available"`
	DoH                []bool            `json:"doh"`
	Link               android.Link      `json:"network"`
	PrivateDNSConflict bool              `json:"privateDnsConflict"`
	Error              string            `json:"error"`
	Ports              Ports             `json:"ports"`
	Versions           map[string]string `json:"versions"`
	RulesUpdated       string            `json:"rulesUpdated"`
}
type Runtime struct {
	needsPurge           bool
	updateMu             sync.Mutex
	Mu                   sync.Mutex
	Module, Data, Binary string
	Sim                  bool
	Prefs                config.Preferences
	M                    Machine
	Status               Status
	Ports                Ports
	AGH                  *agh.Client
	Rules                network.Rules
	Private              android.PrivateDNS
	cmd                  *exec.Cmd
	done                 chan error
	lastStart            time.Time
	attempts             []time.Time
	activeUpstream       string
	lastLink             string
	listener             net.Listener
	stop                 chan struct{}
}

func Uptime() float64 {
	b, e := os.ReadFile("/proc/uptime")
	if e != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(fields[0], 64)
	return v
}
func BootID() string {
	b, e := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if e != nil {
		return "simulation"
	}
	return strings.TrimSpace(string(b))
}
func reserve(host, kind string) (int, io.Closer, error) {
	for i := 0; i < 100; i++ {
		var b [4]byte
		if _, e := rand.Read(b[:]); e != nil {
			return 0, nil, e
		}
		p := 20000 + int(binary.BigEndian.Uint32(b[:])%40000)
		a := net.JoinHostPort(host, strconv.Itoa(p))
		if kind == "tcp" {
			l, e := net.Listen("tcp", a)
			if e == nil {
				return p, l, nil
			}
		} else {
			l, e := net.ListenPacket("udp", a)
			if e == nil {
				return p, l, nil
			}
		}
	}
	return 0, nil, errors.New("无法分配随机端口")
}
func New(module, data, binaryPath string, sim bool) (*Runtime, error) {
	if !sim && (BootID() == "simulation" || Uptime() <= 0) {
		return nil, errors.New("无法可靠读取开机标识或单调时钟；保持旁路")
	}
	r := &Runtime{Module: module, Data: data, Binary: binaryPath, Sim: sim, Prefs: config.Defaults(), Rules: network.Rules{Run: network.Exec{}}, M: Machine{State: Bypass}, stop: make(chan struct{})}
	if e := os.MkdirAll(data, 0700); e != nil {
		return nil, e
	}
	if e := os.Chmod(data, 0700); e != nil {
		return nil, e
	}
	_ = config.Atomic(filepath.Join(data, ".myadblock-owned"), []byte("1\n"))
	if e := os.MkdirAll(filepath.Join(data, "agh-data"), 0700); e != nil {
		return nil, e
	}

	prefPath := filepath.Join(data, "preferences.json")
	if e := config.Read(prefPath, &r.Prefs); e != nil && !os.IsNotExist(e) {
		return nil, e
	}
	if r.Prefs.Schema != config.Schema {
		return nil, errors.New("未知控制配置版本，拒绝覆盖")
	}
	if e := ValidatePrefs(r.Prefs); e != nil {
		return nil, e
	}
	if e := config.Save(prefPath, r.Prefs); e != nil {
		return nil, e
	}
	r.Status.Versions = map[string]string{}
	if b, e := os.ReadFile(filepath.Join(module, "version.properties")); e == nil {
		for _, l := range strings.Split(string(b), "\n") {
			kv := strings.SplitN(l, "=", 2)
			if len(kv) == 2 {
				r.Status.Versions[kv[0]] = kv[1]
			}
		}
	}
	ui, l, e := reserve("127.0.0.1", "tcp")
	if e != nil {
		return nil, e
	}
	r.listener = l.(net.Listener)
	hp, h, e := reserve("127.0.0.1", "tcp")
	if e != nil {
		l.Close()
		return nil, e
	}
	defer h.Close()
	dp, d, e := reserve("127.0.0.1", "udp")
	if e != nil {
		l.Close()
		return nil, e
	}
	defer d.Close()
	t, e := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(dp)))
	if e != nil {
		l.Close()
		return nil, e
	}
	defer t.Close()
	ipv6 := false
	if v, e := net.ListenPacket("udp6", net.JoinHostPort("::1", strconv.Itoa(dp))); e == nil {
		v.Close()
		if v, e := net.Listen("tcp6", net.JoinHostPort("::1", strconv.Itoa(dp))); e == nil {
			v.Close()
			ipv6 = true
		}
	}
	if ui == hp || hp == dp || ui == dp {
		l.Close()
		return nil, errors.New("端口分配碰撞，请重试")
	}
	r.Ports = Ports{BootID(), ui, hp, dp}
	r.Status.Ports = r.Ports
	r.Status.IPv6Available = ipv6
	secretPath := filepath.Join(data, "management.secret")
	secret, e := os.ReadFile(secretPath)
	if os.IsNotExist(e) {
		secret = []byte(config.Random())
		e = config.Atomic(secretPath, secret)
	}
	if e != nil {
		l.Close()
		return nil, e
	}
	r.AGH = agh.New(hp, "myadblock-controller", string(secret))
	cfgPath := filepath.Join(data, "AdGuardHome.yaml")
	m, e := config.Load(cfgPath)
	if os.IsNotExist(e) {
		m = config.Fresh(data)
		if personal, pe := os.ReadFile(filepath.Join(module, "presets/personal.yaml")); pe == nil {
			m, e = config.Import(personal, m)
			if e != nil {
				l.Close()
				return nil, e
			}
		}
	} else if e != nil {
		l.Close()
		return nil, e
	}
	// Configuration changes occur before AGH is started, never while it can overwrite YAML.
	if e = config.Normalize(m, data, hp, dp, ipv6, string(secret)); e != nil {
		l.Close()
		return nil, e
	}
	if e = r.prepareSnapshots(); e != nil {
		l.Close()
		return nil, e
	}
	if e = r.CommitConfig(m); e != nil {
		l.Close()
		return nil, e
	}
	if e = config.Save(filepath.Join(data, "ports.json"), r.Ports); e != nil {
		l.Close()
		return nil, e
	}
	_ = config.Read(filepath.Join(data, "private-dns.json"), &r.Private)
	r.M = restorePause(data, BootID(), Uptime(), time.Now())
	return r, nil
}
func ValidatePrefs(p config.Preferences) error {
	if p.Mode != "daily" && p.Mode != "strict" {
		return errors.New("模式必须为 daily/strict")
	}
	if len(p.DoH) != 2 || len(p.Bootstrap) < 1 || len(p.Bootstrap) > 8 {
		return errors.New("需要两个公共 DoH 和 1–8 个 bootstrap IP")
	}
	for _, u := range p.DoH {
		if e := network.SafeHTTPS(u); e != nil {
			return e
		}
	}
	for _, s := range p.Bootstrap {
		ip := net.ParseIP(s)
		if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() {
			return errors.New("无效 bootstrap IP")
		}
	}
	return nil
}
func (r *Runtime) prepareSnapshots() error {
	for i := 1; i <= 3; i++ {
		p := filepath.Join(r.Data, "subscriptions", fmt.Sprintf("%d.txt", i))
		if _, e := os.Stat(p); os.IsNotExist(e) {
			b, e := os.ReadFile(filepath.Join(r.Module, "presets/snapshots", fmt.Sprintf("%d.txt", i)))
			if e != nil {
				return e
			}
			if e = config.Atomic(p, b); e != nil {
				return e
			}
		}
		cached := filepath.Join(r.Data, "agh-data", "data", "filters", fmt.Sprintf("%d.txt", i))
		if _, e := os.Stat(cached); os.IsNotExist(e) {
			b, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			if e = config.Atomic(cached, b); e != nil {
				return e
			}
		}
	}
	return nil
}

func (r *Runtime) CommitConfig(m config.YAML) error {
	candidate := filepath.Join(r.Data, "candidate.yaml")
	if e := config.Write(candidate, m); e != nil {
		return e
	}
	defer os.Remove(candidate)
	ctx, c := context.WithTimeout(context.Background(), 10*time.Second)
	defer c()
	cmd := exec.CommandContext(ctx, r.Binary, "--check-config", "-c", candidate, "-w", filepath.Join(r.Data, "agh-data"))
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if e := cmd.Run(); e != nil {
		return errors.New("官方 AGH --check-config 校验失败；原配置保留")
	}
	p := filepath.Join(r.Data, "AdGuardHome.yaml")
	if b, e := os.ReadFile(p); e == nil {
		if e = config.Atomic(filepath.Join(r.Data, "backups", "last-good.yaml"), b); e != nil {
			return e
		}
	}
	return config.Write(p, m)
}
func (r *Runtime) StartCore() error {
	now := time.Now()
	kept := []time.Time{}
	for _, t := range r.attempts {
		if now.Sub(t) < 5*time.Minute {
			kept = append(kept, t)
		}
	}
	r.attempts = kept
	if len(kept) >= 3 {
		return errors.New("AGH 5 分钟内启动失败 3 次，已限流；保持旁路")
	}
	r.attempts = append(r.attempts, now)
	r.lastStart = now
	r.cmd = exec.Command(r.Binary, "--no-check-update", "-c", filepath.Join(r.Data, "AdGuardHome.yaml"), "-w", filepath.Join(r.Data, "agh-data"))
	r.cmd.Env = append(os.Environ(), "SSL_CERT_DIR=/apex/com.android.conscrypt/cacerts:/system/etc/security/cacerts")
	r.cmd.Stdout = io.Discard
	r.cmd.Stderr = io.Discard
	if e := r.cmd.Start(); e != nil {
		r.cmd = nil
		r.done = nil
		return e
	}
	r.done = make(chan error, 1)
	go func(c *exec.Cmd, ch chan error) { ch <- c.Wait() }(r.cmd, r.done)
	_ = config.Save(filepath.Join(r.Data, "agh-process.json"), map[string]any{"pid": r.cmd.Process.Pid, "binary": r.Binary, "data": r.Data})
	r.activeUpstream = ""
	return nil
}
func (r *Runtime) StopCore() {
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Signal(os.Interrupt)
		select {
		case <-r.done:
		case <-time.After(4 * time.Second):
			_ = r.cmd.Process.Kill()
			<-r.done
		}
	}
	r.cmd = nil
}
func (r *Runtime) coreAlive() bool {
	if r.cmd == nil {
		return false
	}
	select {
	case <-r.done:
		r.cmd = nil
		return false
	default:
		return true
	}
}
func (r *Runtime) Listener() net.Listener { return r.listener }
func (r *Runtime) remove() error {
	if r.Sim {
		r.Status.IPv4 = false
		r.Status.IPv6 = false
		return nil
	}
	had := r.Status.IPv4 || r.Status.IPv6 || r.Rules.Installed("iptables") || r.Rules.Installed("ip6tables")
	r.needsPurge = r.needsPurge || had
	e := r.Rules.Remove()
	if r.needsPurge {
		ce := network.PurgeDNS(r.Ports.DNS)
		if ce == nil {
			r.needsPurge = false
		}
		e = errors.Join(e, ce)
	}
	r.Status.IPv4 = r.Rules.Installed("iptables")
	r.Status.IPv6 = r.Rules.Installed("ip6tables")
	return e
}
func (r *Runtime) safe(msg string) {
	r.M.State = Bypass
	r.activeUpstream = ""
	r.Status.Error = msg
	if e := r.remove(); e != nil {
		r.Status.Error += "；撤销规则失败，请使用应急入口并检查 Root 权限"
	}
}
func (r *Runtime) Tick() {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	now := time.Now()
	r.renewLease()
	r.Status.Error = ""
	if _, e := os.Stat(filepath.Join(r.Module, "disable")); e == nil {
		r.M.Pause(now, 0)
	}
	if _, e := os.Stat(filepath.Join(r.Data, "emergency.pause")); e == nil {
		r.M.Pause(now, 0)
	}
	if !r.coreAlive() {
		_ = r.remove()
		if now.Sub(r.lastStart) > 10*time.Second {
			if e := r.StartCore(); e != nil {
				r.Status.Error = e.Error()
			}
		}
	}
	var core map[string]any
	ready := r.AGH.Call("GET", "/status", nil, &core) == nil
	r.Status.Core = ready
	r.Status.Protection = core["protection_enabled"] == true
	if r.M.State == Paused && (r.M.Infinite || now.Before(r.M.Until)) {
		if !r.Sim {
			r.Status.PrivateDNSConflict = r.Private.Conflict(r.Rules.Run)
			if r.Status.PrivateDNSConflict {
				r.Status.Error = "暂停期间私人 DNS 存在更改或无法读取；不会覆写，请检查系统设置"
			}
		}
		if e := r.remove(); e != nil {
			r.Status.Error = "暂停时规则撤销失败"
		}
		if ready && r.Status.Protection {
			if e := r.AGH.Protection(false); e != nil {
				r.Status.Error = "DNS 接管已撤销，但 AGH 过滤暂停失败"
			} else {
				r.Status.Protection = false
			}
		}
		r.publish()
		return
	}
	if r.M.State == Paused {
		r.M.Resume()
		_ = os.Remove(filepath.Join(r.Data, "pause.json"))
	}
	if !ready {
		r.safe("AGH 尚未就绪，DNS 接管已撤销")
		r.Status.Core = false
		r.publish()
		return
	}
	if r.Sim {
		r.Status.Link = android.Link{Reliable: false}
		r.safe("模拟运行：未执行 Android DNS 发现、私人 DNS 或网络接管")
		r.publish()
		return
	}
	r.Status.Link = android.Discover(r.Rules.Run, r.Module)
	l := r.Status.Link
	if l.VPN {
		r.safe("检测到 VPN/TUN；保留代理路径，部分流量未接管")
		r.publish()
		return
	}
	if !r.Sim && !l.Reliable {
		r.safe("无法可靠读取底层 LinkProperties，保留原网络路径")
		r.publish()
		return
	}
	linkKey := fmt.Sprint(l.Interface, l.DNS, l.VPN)
	if r.lastLink != linkKey {
		_ = r.remove()
		r.activeUpstream = ""
		r.lastLink = linkKey
		r.M.State = Bypass
		r.M.Fail = 0
		r.M.Recover = 0
	}
	bootstrap := network.MergeBootstrap(r.Prefs.Bootstrap, l.DNS)
	good := make([]bool, len(r.Prefs.DoH))
	var wg sync.WaitGroup
	for i, u := range r.Prefs.DoH {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			ctx, c := context.WithTimeout(context.Background(), 3*time.Second)
			defer c()
			good[i] = network.DoH(ctx, u, bootstrap)
		}(i, u)
	}
	wg.Wait()
	r.Status.DoH = good
	r.renewLease()
	healthy := []string{}
	for i, ok := range good {
		if ok {
			healthy = append(healthy, r.Prefs.DoH[i])
		}
	}
	system := []string{}
	systemGood := make([]bool, len(l.DNS))
	for i, ip := range l.DNS {
		wg.Add(1)
		go func(i int, ip string) {
			defer wg.Done()
			systemGood[i] = network.DNS(net.JoinHostPort(ip, "53"), false)
		}(i, ip)
	}
	wg.Wait()
	for i, ok := range systemGood {
		if ok {
			system = append(system, l.DNS[i])
		}
	}

	state := r.M.Step(now, true, len(healthy) > 0, len(system) > 0)
	if state != Public && state != Fallback {
		r.safe("等待上游连续 3 次确认，或所有上游不可用")
		r.publish()
		return
	}
	up := healthy
	mode := "load_balance"
	if r.Prefs.Mode == "strict" {
		mode = "parallel"
	}
	if state == Fallback {
		up = system
		mode = "load_balance"
	}
	if len(up) == 0 {
		r.safe("当前上游无真实可用服务器，等待防抖后回退")
		r.publish()
		return
	}
	key := fmt.Sprint(up, mode, bootstrap)
	if key != r.activeUpstream {
		_ = r.remove()
		if e := r.AGH.Upstreams(up, bootstrap, mode); e != nil {
			r.safe(e.Error())
			r.publish()
			return
		}
		_ = r.AGH.Call("POST", "/cache_clear", nil, nil)
		r.activeUpstream = key
	}
	if !network.DNS(net.JoinHostPort("127.0.0.1", strconv.Itoa(r.Ports.DNS)), false) || !network.DNS(net.JoinHostPort("127.0.0.1", strconv.Itoa(r.Ports.DNS)), true) {
		r.safe("AGH DNS UDP/TCP 查询校验失败")
		r.publish()
		return
	}
	if !r.Sim {
		if e := network.ConntrackAvailable(); e != nil {
			r.safe("内核缺少受限 conntrack 清理能力；为保证完整暂停，保持旁路")
			r.publish()
			return
		}
		r.Status.PrivateDNSConflict = r.Private.Conflict(r.Rules.Run)
		if r.Status.PrivateDNSConflict {
			r.safe("用户已修改私人 DNS；不抢夺设置，请关闭私人 DNS 后重试")
			r.publish()
			return
		}

		if !r.Status.Protection {
			if e := r.AGH.Protection(true); e != nil {
				r.safe("AGH 过滤恢复失败")
				r.publish()
				return
			}
			r.Status.Protection = true
		}
		self, _ := os.Executable()
		check := func(bin, destination string) bool {
			r.renewLease()
			existed := r.Rules.Installed(bin)
			if e := r.Rules.Install(bin, r.Ports.DNS); e != nil {
				return false
			}
			if existed {
				return true
			}
			ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
			defer c()
			cmd := exec.CommandContext(ctx, self, "capture-check", destination)
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			if e := cmd.Run(); e != nil {
				_ = r.Rules.RemoveFamily(bin)
				_ = network.PurgeDNS(r.Ports.DNS)
				return false
			}
			return true
		}
		r.Status.IPv4 = check("iptables", "192.0.2.53")
		if r.Prefs.IPv6 && r.Status.IPv6Available {
			r.Status.IPv6 = check("ip6tables", "2001:db8::53")
		} else {
			_ = r.Rules.RemoveFamily("ip6tables")
			r.Status.IPv6 = false
		}
		if !r.Status.IPv4 && !r.Status.IPv6 {
			r.safe("无可验证的 DNS 接管路径；可能为 NAT / conntrack / 路由或代理冲突")
			r.publish()
			return
		}
		if e := r.Private.Disable(r.Rules.Run, filepath.Join(r.Data, "private-dns.json")); e != nil {
			r.safe("私人 DNS 备份/关闭失败")
			r.publish()
			return
		}
		if !r.Status.IPv4 || !r.Status.IPv6 {
			r.Status.Error = "接管覆盖不完整；不支持的地址族保持放行"
		}

	}
	r.publish()
}
func (r *Runtime) renewLease() {
	_ = config.Save(filepath.Join(r.Data, "lease.json"), map[string]any{"boot": BootID(), "uptime": Uptime()})
}
func (r *Runtime) publish() {
	r.Status.State = r.M.State
	r.Status.Remaining = r.M.Remaining(time.Now())
	r.Status.Mode = r.Prefs.Mode
	_ = config.Save(filepath.Join(r.Data, "status.json"), r.Status)
	r.renewLease()
}
func (r *Runtime) Snapshot() Status {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	s := r.Status
	s.State = r.M.State
	s.Remaining = r.M.Remaining(time.Now())
	s.Mode = r.Prefs.Mode
	return s
}
func (r *Runtime) Pause(sec int) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	valid := map[int]bool{0: true, 30: true, 60: true, 300: true, 600: true, 3600: true, 86400: true}
	if !valid[sec] {
		return errors.New("不支持的暂停时长")
	}
	r.M.Pause(time.Now(), sec)
	if e := config.Save(filepath.Join(r.Data, "pause.json"), map[string]any{"Boot": BootID(), "Seconds": sec, "Deadline": Uptime() + float64(sec)}); e != nil {
		_ = r.remove()
		return e
	}
	e := r.remove()
	ae := r.AGH.Protection(false)
	r.Status.Protection = ae != nil && r.Status.Protection
	r.publish()
	return errors.Join(e, ae)
}
func (r *Runtime) Resume() error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	_ = os.Remove(filepath.Join(r.Data, "emergency.pause"))
	_ = os.Remove(filepath.Join(r.Data, "pause.json"))
	r.M.Resume()
	r.activeUpstream = ""
	r.publish()
	return nil
}
func (r *Runtime) Shutdown() {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	_ = r.remove()
	r.StopCore()
	r.listener.Close()
}
