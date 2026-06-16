package razor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseIdentityFile(t *testing.T) {
	in := "# razor identity\nuser = bob\npass=s3cret\nother=x\n"
	id, ok := ParseIdentityFile(strings.NewReader(in))
	if !ok {
		t.Fatal("expected ok")
	}
	if id.User != "bob" || id.Pass != "s3cret" {
		t.Errorf("got %+v", id)
	}
}

func TestParseIdentityFileIncomplete(t *testing.T) {
	if _, ok := ParseIdentityFile(strings.NewReader("user=bob\n")); ok {
		t.Error("missing pass should be not-ok")
	}
}

func TestResolveIdentityFlagWins(t *testing.T) {
	t.Setenv("GAZOR_USER", "envuser")
	t.Setenv("GAZOR_PASS", "envpass")
	id := ResolveIdentity("flaguser", "flagpass", t.TempDir())
	if id == nil || id.User != "flaguser" || id.Pass != "flagpass" {
		t.Errorf("flag should win: %+v", id)
	}
}

func TestResolveIdentityEnv(t *testing.T) {
	t.Setenv("GAZOR_USER", "envuser")
	t.Setenv("GAZOR_PASS", "envpass")
	t.Setenv("RAZOR_USER", "")
	t.Setenv("RAZOR_PASS", "")
	id := ResolveIdentity("", "", t.TempDir())
	if id == nil || id.User != "envuser" {
		t.Errorf("GAZOR_* env should resolve: %+v", id)
	}
}

func TestResolveIdentityRazorEnvFallback(t *testing.T) {
	t.Setenv("GAZOR_USER", "")
	t.Setenv("GAZOR_PASS", "")
	t.Setenv("RAZOR_USER", "razoruser")
	t.Setenv("RAZOR_PASS", "razorpass")
	id := ResolveIdentity("", "", t.TempDir())
	if id == nil || id.User != "razoruser" {
		t.Errorf("RAZOR_* env fallback: %+v", id)
	}
}

func TestResolveIdentityFile(t *testing.T) {
	for _, k := range []string{"GAZOR_USER", "GAZOR_PASS", "RAZOR_USER", "RAZOR_PASS"} {
		t.Setenv(k, "")
	}
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "identity"), []byte("user=fileuser\npass=filepass\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	id := ResolveIdentity("", "", home)
	if id == nil || id.User != "fileuser" || id.Pass != "filepass" {
		t.Errorf("identity file: %+v", id)
	}
}

func TestResolveIdentityNone(t *testing.T) {
	for _, k := range []string{"GAZOR_USER", "GAZOR_PASS", "RAZOR_USER", "RAZOR_PASS"} {
		t.Setenv(k, "")
	}
	if id := ResolveIdentity("", "", t.TempDir()); id != nil {
		t.Errorf("no creds should be nil, got %+v", id)
	}
}

func TestResolveHome(t *testing.T) {
	if got := ResolveHome("/explicit"); got != "/explicit" {
		t.Errorf("flag home = %q", got)
	}
	t.Setenv("RAZOR_HOME", "/from/env")
	if got := ResolveHome(""); got != "/from/env" {
		t.Errorf("env home = %q", got)
	}
}
