package engine

import (
	"testing"
	"time"
)

func TestPublicPriorityAndHysteresis(t *testing.T) {
	m := Machine{State: Bypass}
	now := time.Now()
	for i := 0; i < 3; i++ {
		m.Step(now, true, true, true)
	}
	if m.State != Public {
		t.Fatal(m)
	}
	for i := 0; i < 2; i++ {
		m.Step(now, true, false, true)
		if m.State != Public {
			t.Fatal("过早回退")
		}
	}
	m.Step(now, true, false, true)
	if m.State != Fallback {
		t.Fatal(m)
	}
	for i := 0; i < 2; i++ {
		m.Step(now, true, true, true)
		if m.State != Fallback {
			t.Fatal("过早恢复")
		}
	}
	m.Step(now, true, true, true)
	if m.State != Public {
		t.Fatal(m)
	}
}
func TestAllFailAndCoreCrash(t *testing.T) {
	m := Machine{State: Public}
	now := time.Now()
	for i := 0; i < 3; i++ {
		m.Step(now, true, false, false)
	}
	if m.State != Bypass {
		t.Fatal(m)
	}
	m.State = Public
	m.Step(now, false, true, true)
	if m.State != Bypass {
		t.Fatal(m)
	}
}
func TestPauseSurvivesNetworkHealing(t *testing.T) {
	now := time.Now()
	for _, sec := range []int{0, 30, 60, 300, 600, 3600, 86400} {
		m := Machine{}
		m.Pause(now, sec)
		for i := 0; i < 10; i++ {
			if m.Step(now.Add(time.Second), true, true, true) != Paused {
				t.Fatal(sec, m)
			}
		}
		if sec > 0 {
			if m.Step(now.Add(time.Duration(sec+1)*time.Second), true, false, false) != Bypass {
				t.Fatal(m)
			}
		} else if m.Step(now.Add(365*24*time.Hour), true, true, true) != Paused {
			t.Fatal("无限暂停被取消")
		}
		m.Resume()
		if m.State != Bypass {
			t.Fatal(m)
		}
	}
}
func TestLegitimateOneDoH(t *testing.T) {
	m := Machine{State: Public}
	for i := 0; i < 20; i++ {
		if m.Step(time.Now(), true, true, false) != Public {
			t.Fatal("单个公共上游可用时不可回退")
		}
	}
}

func TestFailedCoreLaunchDoesNotLookAlive(t *testing.T) {
	r := &Runtime{Binary: "", Data: t.TempDir(), Sim: true}
	if r.StartCore() == nil {
		t.Fatal("empty executable accepted")
	}
	if r.coreAlive() {
		t.Fatal("failed exec reported running; automatic retries would be blocked")
	}
}
