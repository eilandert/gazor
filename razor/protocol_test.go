package razor

import (
	"bufio"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestProtocolParity verifies the protocol layer (hextobase64, the prep_mail +
// preproc pipeline, and the VR4/VR8 wire signatures) against golden vectors
// generated from real razor by testdata/gen_protocol.pl. This is the end-to-end
// gate: a raw email must yield the exact signatures razor's agent would send.
func TestProtocolParity(t *testing.T) {
	f, err := os.Open("testdata/protocol.tsv")
	if err != nil {
		t.Fatalf("open protocol.tsv: %v", err)
	}
	defer f.Close()

	c := &Client{}
	c.engines = map[int]bool{4: true, 8: true}

	cache := map[string][]*part{}
	load := func(file string) []*part {
		if p, ok := cache[file]; ok {
			return p
		}
		raw, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		parts := c.prepare(raw)
		c.computeSigs(parts, "7542-10")
		cache[file] = parts
		return parts
	}
	partAt := func(file string, idxStr string) *part {
		parts := load(file)
		idx, _ := strconv.Atoi(idxStr)
		if idx < 0 || idx >= len(parts) {
			t.Fatalf("%s: part index %d out of range (%d parts)", file, idx, len(parts))
		}
		return parts[idx]
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n := 0
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r\n")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cols := strings.Split(line, "\t")
		switch cols[0] {
		case "B64":
			if got := hextobase64(cols[1]); got != cols[2] {
				t.Errorf("hextobase64(%s) = %q, want %q", cols[1], got, cols[2])
			}
		case "CLEAN4":
			if got := hex.EncodeToString(partAt(cols[1], cols[2]).cleaned); got != cols[3] {
				t.Errorf("%s part %s VR4 cleaned mismatch\n  got  %s\n  want %s", cols[1], cols[2], got, cols[3])
			}
		case "CLEAN8":
			if got := hex.EncodeToString(partAt(cols[1], cols[2]).cleanedVR8); got != cols[3] {
				t.Errorf("%s part %s VR8 cleaned mismatch\n  got  %s\n  want %s", cols[1], cols[2], got, cols[3])
			}
		case "MAIL":
			p := partAt(cols[1], cols[2])
			var sig4 string
			if s := p.sigs[4]; len(s) > 0 {
				sig4 = s[0]
			}
			if sig4 != cols[3] {
				t.Errorf("%s part %s VR4 sig = %q, want %q", cols[1], cols[2], sig4, cols[3])
			}
			sig8 := strings.Join(p.sigs[8], ",")
			if sig8 != cols[4] {
				t.Errorf("%s part %s VR8 sigs = %q, want %q", cols[1], cols[2], sig8, cols[4])
			}
		default:
			t.Fatalf("unknown kind %q", cols[0])
		}
		n++
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("no vectors in protocol.tsv")
	}
	t.Logf("protocol parity verified over %d vectors", n)
}
