package razor

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

// --- report chunking (was: bodies over bqs silently truncated) ---

func TestBuildReportChunksMultiple(t *testing.T) {
	body := func(c byte) []byte { return []byte(strings.Repeat(string(c), 600)) }
	parts := []*part{{body: body('a')}, {body: body('b')}, {body: body('c')}}
	chunks := buildReportChunks([]byte("HDR"), parts, 1) // limit 1024: ~one body/chunk

	if len(chunks) != 3 {
		t.Fatalf("want 3 chunks (no truncation), got %d", len(chunks))
	}
	// every body must appear somewhere — nothing dropped
	all := strings.Join(chunks, "")
	for _, c := range []byte{'a', 'b', 'c'} {
		if !strings.Contains(all, strings.Repeat(string(c), 600)) {
			t.Errorf("body %q was dropped", c)
		}
	}
	for _, ch := range chunks {
		if !strings.HasPrefix(ch, "HDR") {
			t.Errorf("each chunk must start with the headers, got %.10q", ch)
		}
	}
}

func TestBuildReportChunksForwardProgress(t *testing.T) {
	// a single body larger than the limit must still be placed (no infinite loop,
	// no drop): exactly one chunk containing it.
	big := strings.Repeat("z", 5000)
	chunks := buildReportChunks([]byte("HDR"), []*part{{body: []byte(big)}}, 1)
	if len(chunks) != 1 || !strings.Contains(chunks[0], big) {
		t.Fatalf("oversized lone body: want 1 chunk containing it, got %d chunks", len(chunks))
	}
}

// --- report/revoke acknowledgement validation ---

func TestCheckReportResp(t *testing.T) {
	ok := []struct{ name, resp string }{
		{"err230 wants mail", "err=230\r\n.\r\n"},
		{"res1 accepted", "res=1\r\n.\r\n"},
	}
	for _, c := range ok {
		if err := checkReportResp([]string{c.resp}); err != nil {
			t.Errorf("%s: want nil, got %v", c.name, err)
		}
	}
	bad := []struct{ name, resp string }{
		{"err500", "err=500\r\n.\r\n"},
		{"res0 rejected", "res=0\r\n.\r\n"},
	}
	for _, c := range bad {
		if err := checkReportResp([]string{c.resp}); err == nil {
			t.Errorf("%s: want error, got nil", c.name)
		}
	}
}

// --- response cardinality (was: missing/extra responses silently ignored) ---

func TestDistributeCardinality(t *testing.T) {
	c := &Client{}
	mk := func(n int) *part {
		p := &part{}
		for i := 0; i < n; i++ {
			p.sent = append(p.sent, map[string]string{"a": "c"})
		}
		return p
	}
	// expects 3 responses, gets 1 -> error
	if err := c.distribute([]*part{mk(1), mk(2)}, []string{"p=1\r\n.\r\n"}); err == nil {
		t.Error("mismatch (1 of 3) must error")
	}
	// expects 1, gets 1 -> ok
	p := mk(1)
	if err := c.distribute([]*part{p}, []string{"p=1\r\n.\r\n"}); err != nil {
		t.Errorf("matching cardinality must not error: %v", err)
	}
	if len(p.resp) != 1 {
		t.Errorf("response not distributed: %d", len(p.resp))
	}
}

// --- MIME structure limits ---

func TestSplitMimeDepthCap(t *testing.T) {
	count := 0
	// depth already over the cap -> stop recursing, treat as a single leaf.
	out := splitMime([]byte("Content-Type: multipart/mixed; boundary=X\n\n--X\nfoo\n--X--"),
		"v", true, maxMimeDepth+1, &count)
	if len(out) != 1 {
		t.Fatalf("over-deep part must yield 1 leaf, got %d", len(out))
	}
}

func TestSplitMimePartCap(t *testing.T) {
	var b strings.Builder
	b.WriteString("Content-Type: multipart/mixed; boundary=X\n\n")
	for i := 0; i < maxMimeParts*2; i++ {
		b.WriteString("--X\nContent-Type: text/plain\n\nx\n")
	}
	b.WriteString("--X--\n")
	count := 0
	out := splitMime([]byte(b.String()), "v", false, 0, &count)
	if len(out) == 0 {
		t.Fatal("expected some parts")
	}
	if len(out) > maxMimeParts {
		t.Fatalf("part count not bounded: got %d > cap %d", len(out), maxMimeParts)
	}
}

// --- response framing: absolute deadline bounds a continuous drip ---

func TestReadResponseDeadlineBoundsDrip(t *testing.T) {
	cli, srv := net.Pipe()
	defer cli.Close()
	defer srv.Close()

	// drip one byte every 40ms forever, never sending a terminator: before the
	// absolute-deadline fix this kept resetting the 250ms idle timer indefinitely.
	go func() {
		for {
			if _, err := srv.Write([]byte("x")); err != nil {
				return
			}
			time.Sleep(40 * time.Millisecond)
		}
	}()

	done := make(chan struct{})
	go func() {
		_, _ = readResponse(cli, bufio.NewReader(cli), 200*time.Millisecond)
		close(done)
	}()
	select {
	case <-done: // returned, bounded by the absolute deadline
	case <-time.After(3 * time.Second):
		t.Fatal("readResponse did not return: drip extended the deadline indefinitely")
	}
}

func TestReadResponseTerminated(t *testing.T) {
	cli, srv := net.Pipe()
	defer cli.Close()
	go func() {
		_, _ = srv.Write([]byte("a=g&pm=state\r\n.\r\n"))
		srv.Close()
	}()
	got, err := readResponse(cli, bufio.NewReader(cli), time.Second)
	if err != nil {
		t.Fatalf("terminated response: %v", err)
	}
	if !strings.HasSuffix(got, ".\r\n") {
		t.Errorf("want terminated response, got %q", got)
	}
}
