package razor

import (
	"bufio"
	"context"
	"strings"
	"testing"
	"time"
)

func TestReadLineLimitedFragments(t *testing.T) {
	// a line longer than the bufio buffer must reassemble correctly (ReadSlice
	// returns ErrBufferFull fragments) and still return on '\n'.
	long := strings.Repeat("x", 9000) + "\r\n"
	br := bufio.NewReaderSize(strings.NewReader(long+"next\r\n"), 4096)
	got, err := readLineLimited(br, maxResponseBytes)
	if err != nil || got != long {
		t.Fatalf("long line: err=%v len=%d want %d", err, len(got), len(long))
	}
	got2, err := readLineLimited(br, maxResponseBytes)
	if err != nil || got2 != "next\r\n" {
		t.Errorf("second line: %q err=%v", got2, err)
	}
	// over the cap → error
	if _, err := readLineLimited(bufio.NewReader(strings.NewReader(strings.Repeat("y", 100))), 16); err == nil {
		t.Error("expected line-too-long error")
	}
}

func TestBudgetIsTotalAndShrinks(t *testing.T) {
	c := &Client{Timeout: time.Second}
	// no op in progress → budget is the full timeout
	if got := c.budget(); got != time.Second {
		t.Errorf("idle budget = %s, want 1s", got)
	}
	c.beginOp(context.Background())
	if got := c.budget(); got <= 0 || got > time.Second {
		t.Errorf("op budget = %s, want (0,1s]", got)
	}
	// an earlier ctx deadline wins
	c.endOp()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	c.beginOp(ctx)
	if got := c.budget(); got > 150*time.Millisecond {
		t.Errorf("ctx-bounded budget = %s, want <=~100ms", got)
	}
	c.endOp()
	if got := c.budget(); got != time.Second {
		t.Errorf("after endOp budget = %s, want full 1s", got)
	}
}

func TestCheckContextCancelledIsPrompt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := &Client{Server: "127.0.0.1:1", Timeout: 10 * time.Second} // explicit server, never dialled
	start := time.Now()
	if _, err := c.CheckContext(ctx, []byte("From: a@b\r\n\r\nhello world test body here today\r\n")); err == nil {
		t.Fatal("cancelled context should fail")
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("cancelled check took %s, want prompt", d)
	}
}

func TestDiscCache(t *testing.T) {
	key := "catalogue\x00disc.example"
	discCacheDel(key)
	if _, ok := discCacheGet(key); ok {
		t.Fatal("empty cache should miss")
	}
	discCachePut(key, []string{"a:2703", "b:2703"})
	got, ok := discCacheGet(key)
	if !ok || len(got) != 2 || got[0] != "a:2703" {
		t.Errorf("get = %v ok=%v", got, ok)
	}
	discCacheDel(key)
	if _, ok := discCacheGet(key); ok {
		t.Error("after del should miss")
	}
}

func TestDiscCacheExpiry(t *testing.T) {
	key := "nomination\x00x"
	discMu.Lock()
	discCache[key] = discEntry{servers: []string{"s"}, exp: time.Now().Add(-time.Second)} // already expired
	discMu.Unlock()
	if _, ok := discCacheGet(key); ok {
		t.Error("expired entry should miss")
	}
}
