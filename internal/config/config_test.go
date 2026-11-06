package config

import (
	"golang.org/x/crypto/bcrypt"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultCredentialsAndNormalization(t *testing.T) {
	m := Fresh("/data/adb/myadblock")
	u := m["users"].([]any)[0].(map[string]any)
	if u["name"] != "root" || bcrypt.CompareHashAndPassword([]byte(u["password"].(string)), []byte("root")) != nil {
		t.Fatal("credentials")
	}
	m["user_rules"] = []string{"@@||keep.example^"}
	if e := Normalize(m, "/data/adb/myadblock", 23456, 34567, true, "private"); e != nil {
		t.Fatal(e)
	}
	if Section(m, "http")["address"] != "127.0.0.1:23456" {
		t.Fatal(m)
	}
	if len(m["user_rules"].([]string)) != 1 {
		t.Fatal("lost rules")
	}
	if len(m["users"].([]any)) != 2 {
		t.Fatal("management credential missing")
	}
}
func TestImportBoundary(t *testing.T) {
	for _, s := range []string{"schema_version: 34\nhttp:\n  address: 0.0.0.0:80", "schema_version: 34\ndns:\n  upstream_dns_file: /etc/shadow", "schema_version: 34\nquerylog:\n  dir_path: /sdcard/", "schema_version: 999"} {
		if _, e := Import([]byte(s), Fresh("/test")); e == nil {
			t.Fatal(s)
		}
	}
	if _, e := Import([]byte("schema_version: 34\nuser_rules:\n - '@@||keep.example^'"), Fresh("/test")); e != nil {
		t.Fatal(e)
	}
}
func TestAtomicPermissions(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	if e := Atomic(p, []byte("ok")); e != nil {
		t.Fatal(e)
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0600 {
		t.Fatal(st.Mode())
	}
}
func TestUnknownSchema(t *testing.T) {
	m := YAML{"schema_version": 99}
	if Normalize(m, "/tmp", 12345, 23456, false, "secret") == nil {
		t.Fatal("schema silently rewritten")
	}
}

func TestPersistedNestedMapsPreserveCredentialsAndSettings(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	m := Fresh("/data/adb/myadblock")
	password := Hash("custom-password")
	m["users"].([]any)[0].(map[string]any)["password"] = password
	Section(m, "dns")["cache_size"] = 12345678
	m["user_rules"] = []string{"@@||keep.example^"}
	if e := Write(p, m); e != nil {
		t.Fatal(e)
	}
	loaded, e := Load(p)
	if e != nil {
		t.Fatal(e)
	}
	if e = Normalize(loaded, "/data/adb/myadblock", 23456, 34567, true, "private"); e != nil {
		t.Fatal(e)
	}
	if loaded["users"].([]any)[0].(map[string]any)["password"] != password {
		t.Fatal("persisted custom credential reset")
	}
	if Section(loaded, "dns")["cache_size"] != 12345678 {
		t.Fatal("persisted DNS fields overwritten")
	}
	if len(loaded["user_rules"].([]any)) != 1 {
		t.Fatal("rules lost")
	}
}
