package engine

import (
	"github.com/Luoning77/MyAdBlock/internal/config"
	"math"
	"os"
	"path/filepath"
	"time"
)

// Pause recovery depends on its own boot record, never on the unrelated port-state file.
// Corruption preserves a safe indefinite pause rather than silently re-enabling capture.
func restorePause(data, boot string, up float64, now time.Time) Machine {
	m := Machine{State: Bypass}
	p := filepath.Join(data, "pause.json")
	var saved struct {
		Boot     string
		Seconds  int
		Deadline float64
	}
	if e := config.Read(p, &saved); e == nil {
		valid := map[int]bool{0: true, 30: true, 60: true, 300: true, 600: true, 3600: true, 86400: true}
		if saved.Boot == "" || !valid[saved.Seconds] || saved.Deadline < 0 {
			m.Pause(now, 0)
		} else if saved.Boot != boot {
			_ = os.Remove(p)
		} else if saved.Seconds == 0 {
			m.Pause(now, 0)
		} else if saved.Deadline > up {
			left := math.Ceil(saved.Deadline - up)
			if left > 86400 {
				m.Pause(now, 0)
			} else {
				m.Pause(now, int(left))
			}
		} else {
			_ = os.Remove(p)
		}
	} else if !os.IsNotExist(e) {
		m.Pause(now, 0)
	}
	var emergency struct{ Boot string }
	ep := filepath.Join(data, "emergency.pause")
	if e := config.Read(ep, &emergency); e == nil {
		if emergency.Boot == boot || emergency.Boot == "" {
			m.Pause(now, 0)
		} else {
			_ = os.Remove(ep)
		}
	} else if !os.IsNotExist(e) {
		m.Pause(now, 0)
	}
	return m
}
