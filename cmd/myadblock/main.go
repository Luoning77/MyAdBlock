// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/Luoning77/MyAdBlock/internal/android"
	"github.com/Luoning77/MyAdBlock/internal/bridge"
	"github.com/Luoning77/MyAdBlock/internal/config"
	"github.com/Luoning77/MyAdBlock/internal/engine"
	"github.com/Luoning77/MyAdBlock/internal/network"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func fatal(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func lock(p string) (*os.File, error) {
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		f.Close()
		return nil, errors.New("进程已运行")
	}
	return f, nil
}
func owns(pid int, word, data string) bool {
	if pid <= 1 {
		return false
	}
	b, e := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	return e == nil && strings.Contains(string(b), word) && strings.Contains(string(b), data)
}
func stopPrevious(data string) {
	var p struct {
		Pid          int
		Binary, Data string
	}
	if config.Read(filepath.Join(data, "agh-process.json"), &p) == nil && p.Data == data && owns(p.Pid, p.Binary, data) {
		_ = syscall.Kill(p.Pid, syscall.SIGTERM)
		for i := 0; i < 30; i++ {
			if !owns(p.Pid, p.Binary, data) {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		if owns(p.Pid, p.Binary, data) {
			_ = syscall.Kill(p.Pid, syscall.SIGKILL)
		}
	}
}
func main() {
	if len(os.Args) < 2 {
		fmt.Println("myadblock daemon | grant | emergency | stop | uninstall | status | watchdog")
		return
	}
	command := os.Args[1]
	f := flag.NewFlagSet(command, flag.ExitOnError)
	module := f.String("module", "", "模块目录")
	data := f.String("data", "/data/adb/myadblock", "数据目录")
	aghBin := f.String("agh", "", "AGH binary")
	sim := f.Bool("simulate", false, "不执行 Android/iptables 操作")
	_ = f.Parse(os.Args[2:])
	if *module == "" {
		exe, _ := os.Executable()
		*module = filepath.Dir(filepath.Dir(exe))
	}
	*module, _ = filepath.Abs(*module)
	*data, _ = filepath.Abs(*data)
	if *aghBin == "" {
		*aghBin = filepath.Join(*module, "bin/AdGuardHome")
	}
	if *sim {
		if *data == "/data/adb/myadblock" {
			fatal(errors.New("模拟模式必须指定独立 --data 目录，不能使用设备真实数据目录"))
		}
		var existing struct {
			Pid       int
			Simulated bool
		}
		if config.Read(filepath.Join(*data, "daemon.json"), &existing) == nil && !existing.Simulated {
			fatal(errors.New("拒绝对真实或未知守护的数据目录执行模拟操作"))
		}
	}
	if !*sim && os.Geteuid() != 0 {
		fatal(errors.New("需要 root"))
	}
	rules := network.Rules{Run: network.Exec{}, LockPath: filepath.Join(*data, "network.lock"), Guard: filepath.Join(*data, "emergency.pause")}
	switch command {
	case "status":
		b, e := os.ReadFile(filepath.Join(*data, "status.json"))
		fatal(e)
		fmt.Println(string(b))
	case "grant":
		var p engine.Ports
		fatal(config.Read(filepath.Join(*data, "ports.json"), &p))
		if p.UI < 20000 || p.UI >= 60000 || p.Boot != engine.BootID() {
			fatal(errors.New("端口状态过期，请重启服务"))
		}
		token := config.Random()
		fatal(config.Save(filepath.Join(*data, "grant.json"), map[string]any{"Token": token, "Expires": time.Now().Add(time.Minute).Unix()}))
		target := fmt.Sprintf("http://127.0.0.1:%d/#grant=%s", p.UI, token)
		if *sim {
			fmt.Println("设备授权已生成（未打印秘密）")
			return
		}
		cmd := exec.Command("/system/bin/am", "start", "--user", "current", "-a", "android.intent.action.VIEW", "-d", target)
		cmd.Stdout = nil
		cmd.Stderr = nil
		fatal(cmd.Run())
		fmt.Println("已打开 MyAdBlock；登录后获得 5 分钟操作授权")
	case "emergency", "stop", "uninstall":
		fatal(config.Save(filepath.Join(*data, "emergency.pause"), map[string]any{"Boot": engine.BootID()}))
		var cleanupErr error
		if !*sim {
			had := rules.Installed("iptables") || rules.Installed("ip6tables")
			cleanupErr = rules.Remove()
			var ports engine.Ports
			if had && config.Read(filepath.Join(*data, "ports.json"), &ports) == nil {
				cleanupErr = errors.Join(cleanupErr, network.PurgeDNS(ports.DNS))
			}
		}
		var p struct{ Pid int }
		_ = config.Read(filepath.Join(*data, "daemon.json"), &p)
		if owns(p.Pid, "myadblock", *data) {
			sig := syscall.SIGUSR1
			if command != "emergency" {
				sig = syscall.SIGTERM
			}
			_ = syscall.Kill(p.Pid, sig)
		}
		if command != "emergency" {
			for i := 0; i < 60 && owns(p.Pid, "myadblock", *data); i++ {
				time.Sleep(100 * time.Millisecond)
			}
			if owns(p.Pid, "myadblock", *data) {
				fatal(errors.New("守护进程未停止；保留数据，禁止不安全卸载"))
			}
			stopPrevious(*data)
			var guardian struct{ Pid int }
			_ = config.Read(filepath.Join(*data, "watchdog-process.json"), &guardian)
			if owns(guardian.Pid, "watchdog", *data) {
				_ = syscall.Kill(guardian.Pid, syscall.SIGTERM)
			}
			if !*sim {
				fatal(rules.Remove())
				fatal(android.Restore(network.Exec{}, filepath.Join(*data, "private-dns.json")))
			}
		}
		if cleanupErr != nil {
			fatal(fmt.Errorf("进程已通知停止，但接管清理未完整完成；保留数据: %w", cleanupErr))
		}
		if command == "uninstall" {
			if *data != "/data/adb/myadblock" {
				fatal(errors.New("拒绝清理非标准数据目录"))
			}
			if b, e := os.ReadFile(filepath.Join(*data, ".myadblock-owned")); e != nil || string(b) != "1\n" {
				fatal(errors.New("所有权标识缺失，保留数据"))
			}
			fatal(os.RemoveAll(*data))
		}
		fmt.Println("已撤销本模块接管；原网络本身仍需可用")
	case "watchdog":
		w, e := lock(filepath.Join(*data, "watchdog.lock"))
		fatal(e)
		defer w.Close()
		_ = config.Save(filepath.Join(*data, "watchdog-process.json"), map[string]any{"Pid": os.Getpid()})
		for {
			if _, e := os.Stat(filepath.Join(*data, ".myadblock-owned")); e != nil {
				return
			}
			var lease struct {
				Boot   string
				Uptime float64
			}
			e := config.Read(filepath.Join(*data, "lease.json"), &lease)

			if e != nil || lease.Boot != engine.BootID() || engine.Uptime()-lease.Uptime > 15 {
				var cleanup error
				if !*sim {
					var status engine.Status
					_ = config.Read(filepath.Join(*data, "status.json"), &status)
					had := rules.Installed("iptables") || rules.Installed("ip6tables") || status.IPv4 || status.IPv6
					cleanup = rules.Remove()
					var ports engine.Ports
					if had && config.Read(filepath.Join(*data, "ports.json"), &ports) == nil {
						cleanup = errors.Join(cleanup, network.PurgeDNS(ports.DNS))
					}
				}
				var daemon struct{ Pid int }
				_ = config.Read(filepath.Join(*data, "daemon.json"), &daemon)
				if !owns(daemon.Pid, "myadblock", *data) && cleanup == nil {
					stopPrevious(*data)
					return
				}
			}

			time.Sleep(time.Second)
		}
	case "daemon":
		own, e := lock(filepath.Join(*data, "daemon.lock"))
		fatal(e)
		defer own.Close()
		if !*sim {
			var oldStatus engine.Status
			_ = config.Read(filepath.Join(*data, "status.json"), &oldStatus)
			had := rules.Installed("iptables") || rules.Installed("ip6tables") || oldStatus.IPv4 || oldStatus.IPv6
			fatal(rules.Remove())
			var oldPorts engine.Ports
			if config.Read(filepath.Join(*data, "ports.json"), &oldPorts) != nil {
				oldPorts = oldStatus.Ports
			}
			if oldPorts.Boot == engine.BootID() && oldPorts.DNS >= 1024 && oldPorts.DNS <= 65535 {
				if ce := network.PurgeDNS(oldPorts.DNS); ce != nil && had {
					fatal(errors.New("旧 DNS 连接映射清理失败，保持旁路；请应急恢复或重启"))
				}
			} else if had && oldPorts.Boot == engine.BootID() {
				fatal(errors.New("无法验证旧 DNS 端口，保持旁路；请应急恢复或重启"))
			}
			stopPrevious(*data)
		}
		r, e := engine.New(*module, *data, *aghBin, *sim)
		fatal(e)
		r.Rules = rules
		var guardian *exec.Cmd
		defer func() {
			r.Shutdown()
			if guardian != nil && guardian.Process != nil {
				_ = guardian.Process.Kill()
			}
		}()
		fatal(config.Save(filepath.Join(*data, "daemon.json"), map[string]any{"Pid": os.Getpid(), "Simulated": *sim}))
		fatal(r.StartCore())
		if !*sim {
			_ = config.Save(filepath.Join(*data, "lease.json"), map[string]any{"boot": engine.BootID(), "uptime": engine.Uptime()})
			self, _ := os.Executable()
			wd := exec.Command(self, "watchdog", "--module", *module, "--data", *data)
			wd.Stdout = nil
			wd.Stderr = nil
			fatal(wd.Start())
			guardian = wd
			go func() { _ = wd.Wait() }()
		}
		server := bridge.New(r).HTTP()
		serverErr := make(chan error, 1)
		go func() { serverErr <- server.Serve(r.Listener()) }()
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		daily := time.NewTicker(24 * time.Hour)
		defer daily.Stop()
		sig := make(chan os.Signal, 4)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGUSR1)
		defer signal.Stop(sig)
		fmt.Println("MyAdBlock 本地服务已启动；状态以 status.json 为准")
		for {
			select {
			case <-tick.C:
				r.Tick()
			case <-daily.C:
				go func() { _ = r.UpdateRules() }()
			case v := <-sig:
				if v == syscall.SIGUSR1 {
					_ = r.Pause(0)
					continue
				}
				ctx, c := context.WithTimeout(context.Background(), time.Second)
				_ = server.Shutdown(ctx)
				c()
				return
			case e := <-serverErr:
				if e != nil && e != http.ErrServerClosed {
					fmt.Fprintln(os.Stderr, "bridge 停止；撤销 DNS 接管")
				}
				return
			}
		}
	case "capture-check":
		if f.NArg() != 1 {
			fatal(errors.New("需要 DNS 服务器 IP"))
		}
		ip := f.Arg(0)
		if os.Geteuid() != 0 {
			fatal(errors.New("capture probe must start as root"))
		}
		fatal(syscall.Setgroups([]int{3003}))
		fatal(syscall.Setgid(9999))
		fatal(syscall.Setuid(9999))
		if os.Geteuid() != 9999 {
			fatal(errors.New("probe UID drop failed"))
		}
		if !network.Capture(net.JoinHostPort(ip, "53")) {
			os.Exit(1)
		}
	default:
		fatal(errors.New("未知命令 " + strconv.Quote(command)))
	}
}
