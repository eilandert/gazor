package razor

import (
	"bufio"
	"strings"
	"testing"
)

func TestServerAddr(t *testing.T) {
	c := &Client{Port: 2703}
	cases := map[string]string{
		"host.example":      "host.example:2703", // bare host gets default port
		"host.example:9999": "host.example:9999", // explicit port preserved
		"127.0.0.1:2703":    "127.0.0.1:2703",    // the bug case: not re-wrapped
		"[::1]:2703":        "[::1]:2703",        // bracketed ipv6 with port
		"::1":               "[::1]:2703",        // bare ipv6 gets bracketed default
		"[fe80::1]":         "[fe80::1]:2703",    // bracketed ipv6 without port
	}
	for in, want := range cases {
		if got := c.serverAddr(in); got != want {
			t.Errorf("serverAddr(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckReportRespStrict(t *testing.T) {
	bad := map[string]string{
		"empty body":     ".\r\n",            // no fields → not a success
		"only other key": "foo=bar\r\n.\r\n", // no res/err success marker
	}
	for name, resp := range bad {
		if err := checkReportResp([]string{resp}); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
	}
	if err := checkReportResp(nil); err == nil {
		t.Error("nil responses should error")
	}
	// recognized successes still pass
	for _, ok := range []string{"err=230\r\n.\r\n", "res=1\r\n.\r\n"} {
		if err := checkReportResp([]string{ok}); err != nil {
			t.Errorf("%q: want nil, got %v", ok, err)
		}
	}
}

func TestReadLineLimited(t *testing.T) {
	// a line with no newline, longer than the cap, must error not allocate freely
	br := bufio.NewReader(strings.NewReader(strings.Repeat("A", 100)))
	if _, err := readLineLimited(br, 16); err == nil {
		t.Error("expected line-too-long error")
	}
	// a normal line returns with the newline
	br = bufio.NewReader(strings.NewReader("hello\r\nrest"))
	got, err := readLineLimited(br, 1024)
	if err != nil || got != "hello\r\n" {
		t.Errorf("got %q err %v", got, err)
	}
}
