package engine

import (
	"github.com/Luoning77/MyAdBlock/internal/config"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPauseRecoveryIgnoresMissingOrCorruptPorts(t *testing.T) {
	d := t.TempDir()
	now := time.Now()
	_ = config.Save(filepath.Join(d, "pause.json"), map[string]any{"Boot": "same-boot", "Seconds": 300, "Deadline": 400.25})
	_ = config.Atomic(filepath.Join(d, "ports.json"), []byte("corrupted"))
	m := restorePause(d, "same-boot", 100, now)
	if m.State != Paused || m.Infinite || m.Remaining(now) < 300 {
		t.Fatal("pause lost with port state", m)
	}
}
func TestPauseAndEmergencyClearOnlyOnNewBoot(t *testing.T) {
	d := t.TempDir()
	_ = config.Save(filepath.Join(d, "pause.json"), map[string]any{"Boot": "old", "Seconds": 0, "Deadline": 0})
	_ = config.Save(filepath.Join(d, "emergency.pause"), map[string]any{"Boot": "old"})
	m := restorePause(d, "new", 100, time.Now())
	if m.State != Bypass {
		t.Fatal(m)
	}
	if _, e := os.Stat(filepath.Join(d, "emergency.pause")); !os.IsNotExist(e) {
		t.Fatal("old emergency marker not cleared")
	}
}
func TestCorruptPauseNeverReenablesCapture(t *testing.T) {
	d := t.TempDir()
	_ = config.Atomic(filepath.Join(d, "pause.json"), []byte("bad JSON"))
	m := restorePause(d, "boot", 100, time.Now())
	if m.State != Paused || !m.Infinite {
		t.Fatal(m)
	}
}
