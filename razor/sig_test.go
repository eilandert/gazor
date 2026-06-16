package razor

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// TestSignatureParity is the make-or-break gate: gazor's Ephemeral and Whiplash
// signatures must be byte-identical to real razor (Razor2::Signature::*). The
// golden vectors in testdata/expected.tsv are generated from the razor perl source
// by testdata/gen_expected.pl (committed because razor's perl modules are not
// apt-installable on current Debian).
func TestSignatureParity(t *testing.T) {
	f, err := os.Open("testdata/expected.tsv")
	if err != nil {
		t.Fatalf("open expected.tsv: %v", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r\n")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cols := strings.SplitN(line, "\t", 3)
		if len(cols) != 3 {
			t.Fatalf("bad line: %q", line)
		}
		kind, file, want := cols[0], cols[1], cols[2]
		raw, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		var got string
		switch kind {
		case "EPH":
			got = Ephemeral(raw)
		case "WHIP":
			got = strings.Join(Whiplash(string(raw)), ",")
		default:
			t.Fatalf("unknown kind %q", kind)
		}
		if got != want {
			t.Errorf("%s %s: mismatch\n  got  %q\n  want %q", kind, file, got, want)
		}
		n++
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("no vectors in expected.tsv")
	}
	t.Logf("signature parity verified over %d vectors", n)
}

func TestDrand48MatchesPerl(t *testing.T) {
	// srand(42); rand() x3 from real perl (Perl_drand48).
	want := []float64{0.74452500006100664, 0.34270147871890799, 0.11108528244416149}
	d := newDrand48(42)
	for i, w := range want {
		if got := d.f(); got != w {
			t.Errorf("drand48[%d] = %.17g, want %.17g", i, got, w)
		}
	}
}
