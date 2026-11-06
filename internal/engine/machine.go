// SPDX-License-Identifier: GPL-3.0-or-later
package engine

import "time"

type State string

const (
	Public   State = "PUBLIC_DOH"
	Fallback State = "SYSTEM_DNS_FALLBACK"
	Bypass   State = "BYPASS_SAFE"
	Paused   State = "USER_PAUSED"
)

type Machine struct {
	State         State
	Fail, Recover int
	Until         time.Time
	Infinite      bool
}

func (m *Machine) Pause(now time.Time, sec int) {
	m.State = Paused
	m.Infinite = sec == 0
	m.Until = now.Add(time.Duration(sec) * time.Second)
	m.Fail = 0
	m.Recover = 0
}
func (m *Machine) Resume() {
	m.State = Bypass
	m.Infinite = false
	m.Until = time.Time{}
	m.Fail = 0
	m.Recover = 0
}
func (m *Machine) Step(now time.Time, agh, public, system bool) State {
	if m.State == Paused {
		if m.Infinite || now.Before(m.Until) {
			return Paused
		}
		m.Resume()
	}
	if !agh {
		m.State = Bypass
		m.Recover = 0
		return m.State
	}
	if public {
		m.Fail = 0
		m.Recover++
		if m.State == Public || m.Recover >= 3 {
			m.State = Public
		}
		if m.State == Fallback && !system {
			m.State = Bypass
		}
	} else {
		m.Recover = 0
		m.Fail++
		if m.Fail >= 3 {
			if system {
				m.State = Fallback
			} else {
				m.State = Bypass
			}
		}
	}
	return m.State
}
func (m *Machine) Remaining(now time.Time) int {
	if m.State != Paused {
		return 0
	}
	if m.Infinite {
		return -1
	}
	s := int(m.Until.Sub(now).Seconds())
	if s < 0 {
		return 0
	}
	return s + 1
}
