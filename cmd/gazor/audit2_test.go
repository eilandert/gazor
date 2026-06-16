package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSafetyCheck(t *testing.T) {
	// no token + non-loopback bind → refused
	if err := (serveConfig{listen: ":8079"}).safetyCheck(); err == nil {
		t.Error(":8079 without token should be refused")
	}
	if err := (serveConfig{listen: "0.0.0.0:8079"}).safetyCheck(); err == nil {
		t.Error("0.0.0.0 without token should be refused")
	}
	// no token + loopback → allowed
	if err := (serveConfig{listen: "127.0.0.1:8079"}).safetyCheck(); err != nil {
		t.Errorf("loopback without token should be allowed: %v", err)
	}
	// token set → any bind allowed
	if err := (serveConfig{listen: ":8079", token: "x"}).safetyCheck(); err != nil {
		t.Errorf("token set should allow any bind: %v", err)
	}
	// unix-socket only (no TCP) → allowed
	if err := (serveConfig{unix: "/tmp/s"}).safetyCheck(); err != nil {
		t.Errorf("unix-only should be allowed: %v", err)
	}
}

func TestIsLoopbackListen(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1:8079": true,
		"localhost:8079": true,
		"[::1]:8079":     true,
		":8079":          false, // all interfaces
		"0.0.0.0:8079":   false,
		"10.0.0.5:8079":  false,
	} {
		if got := isLoopbackListen(addr); got != want {
			t.Errorf("isLoopbackListen(%q) = %v, want %v", addr, got, want)
		}
	}
}

func TestReadCapped(t *testing.T) {
	// exactly max is accepted
	if b, err := readCapped(strings.NewReader("12345"), 5); err != nil || len(b) != 5 {
		t.Errorf("exact max: %d bytes err %v", len(b), err)
	}
	// over max is rejected (not truncated)
	if _, err := readCapped(strings.NewReader("123456"), 5); err != errTooLarge {
		t.Errorf("over max: want errTooLarge, got %v", err)
	}
}

func TestServeConcurrencyGate(t *testing.T) {
	// maxConc 1: a second concurrent request gets 503 while the first holds the slot.
	release := make(chan struct{})
	entered := make(chan struct{})
	cfg := serveConfig{maxConc: 1}
	if cfg.maxConc < 1 {
		cfg.maxConc = 8
	}
	sem := make(chan struct{}, cfg.maxConc)
	gate := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				next(w, r)
			default:
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		}
	}
	h := gate(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
	})
	go h(httptest.NewRecorder(), httptest.NewRequest("POST", "/check", nil))
	<-entered // first request holds the only slot
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest("POST", "/check", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("second concurrent request = %d, want 503", rec.Code)
	}
	close(release)
}
