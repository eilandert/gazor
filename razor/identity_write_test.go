package razor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteIdentityFileExplicitOut(t *testing.T) {
	out := filepath.Join(t.TempDir(), "creds")
	got, err := WriteIdentityFile("", out, Identity{User: "bob@example.com", Pass: "s3cret"})
	if err != nil {
		t.Fatal(err)
	}
	if got != out {
		t.Fatalf("path = %s, want %s", got, out)
	}
	if runtime.GOOS != "windows" {
		fi, _ := os.Stat(out)
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Fatalf("perm = %o, want 600", perm)
		}
	}
	f, _ := os.Open(out)
	id, ok := ParseIdentityFile(f)
	_ = f.Close()
	if !ok || id.User != "bob@example.com" || id.Pass != "s3cret" {
		t.Fatalf("round-trip = %+v ok=%v", id, ok)
	}
}

func TestWriteIdentityFileDefaultAndNoClobber(t *testing.T) {
	home := t.TempDir()
	// First write lands at <home>/identity.
	got, err := WriteIdentityFile(home, "", Identity{User: "u", Pass: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(home, "identity") {
		t.Fatalf("first write path = %s", got)
	}
	// Second write must NOT clobber the active identity — lands at identity-<user>.
	got2, err := WriteIdentityFile(home, "", Identity{User: "new@x", Pass: "np"})
	if err != nil {
		t.Fatal(err)
	}
	if got2 != filepath.Join(home, "identity-new@x") {
		t.Fatalf("no-clobber path = %s", got2)
	}
	b, _ := os.ReadFile(filepath.Join(home, "identity"))
	if !strings.Contains(string(b), "user=u") {
		t.Fatalf("active identity was clobbered: %q", b)
	}
}

func TestWriteIdentityFileRejectsEmpty(t *testing.T) {
	if _, err := WriteIdentityFile(t.TempDir(), "", Identity{User: "u"}); err == nil {
		t.Fatal("expected error for empty pass")
	}
}

func TestSanitizeUser(t *testing.T) {
	if got := sanitizeUser("a b/c..d@e"); got != "a_b_c..d@e" {
		t.Errorf("got %q", got)
	}
}
