package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
)

const spamMail = "From: spammer@example.com\n" +
	"To: victim@example.org\n" +
	"Subject: Cheap meds online\n" +
	"Content-Type: text/plain\n\n" +
	"Buy cheap pills now at http://pills.example.com/deal and save big!\n" +
	"Visit www.bonus.example.net for more offers.\n" +
	"Limited time only. Act fast.\n"

func runCLI(t *testing.T, args []string, stdin string) (int, string, string) {
	t.Helper()
	var out, errb strings.Builder
	code := run(args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

// TestCLISig is offline and deterministic: the sig output must match the golden
// VR4/VR8 signatures (same vectors as razor/testdata/protocol.tsv).
func TestCLISig(t *testing.T) {
	code, out, errb := runCLI(t, []string{"sig"}, spamMail)
	if code != 0 {
		t.Fatalf("sig exit %d, err=%q", code, errb)
	}
	for _, want := range []string{
		"part 0 e4: VeqrM3rqOTDa3IRFG3vSd8QwlhUA",
		"part 0 e8: 95xTxuWzyA0A",
		"part 0 e8: A1X0JdhQyA0A",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("sig output missing %q\ngot:\n%s", want, out)
		}
	}
}

// fakeRazor is a minimal local razor server: greeting, state, then a fixed
// spam/ham verdict for every check query in a batch.
func fakeRazor(t *testing.T, spam bool) (host string, port int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveFake(conn, spam)
		}
	}()
	h, p, _ := net.SplitHostPort(ln.Addr().String())
	n, _ := strconv.Atoi(p)
	return h, n
}

func serveFake(conn net.Conn, spam bool) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	_, _ = conn.Write([]byte("sn=C&srl=1&ep4=7542-10\r\n"))
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		switch {
		case strings.HasPrefix(line, "a=g&pm=state"):
			_, _ = conn.Write([]byte("se=88&ac=50&bql=4&bqs=128\r\n.\r\n"))
		case strings.HasPrefix(line, "a=reg"):
			// Registration reply is a single line (no ".\r\n" terminator):
			// Client.Register parses resp[0] directly, and readResponse returns
			// it on the idle/EOF after the client stops sending.
			_, _ = conn.Write([]byte("res=1&user=newuser&pass=newpass\r\n"))
		case strings.HasPrefix(line, "a=q"):
			return
		case strings.HasPrefix(line, "-"):
			// consume the rest of the batch up to the ".\r\n" terminator;
			// count query lines (every "\r\n" except the trailing dot line).
			n := strings.Count(line, "\r\n")
			for !strings.HasSuffix(line, ".\r\n") {
				more, e := br.ReadString('\n')
				if e != nil {
					break
				}
				n += strings.Count(more, "\r\n")
				line = more
			}
			queries := n - 1 // subtract the ".\r\n" terminator line
			if queries < 1 {
				queries = 1
			}
			p := "0"
			if spam {
				p = "1"
			}
			var sb strings.Builder
			for i := 0; i < queries; i++ {
				if i == 0 {
					sb.WriteString("-")
				}
				sb.WriteString(fmt.Sprintf("p=%s&cf=99\r\n", p))
			}
			sb.WriteString(".\r\n")
			_, _ = conn.Write([]byte(sb.String()))
		}
	}
}

func TestCLICheckSpam(t *testing.T) {
	h, p := fakeRazor(t, true)
	code, out, errb := runCLI(t, []string{"--server", h, "--port", strconv.Itoa(p), "--timeout", "3s", "check"}, spamMail)
	if code != 0 {
		t.Fatalf("expected spam (exit 0), got %d; out=%q err=%q", code, out, errb)
	}
}

func TestCLICheckHam(t *testing.T) {
	h, p := fakeRazor(t, false)
	code, _, errb := runCLI(t, []string{"--server", h, "--port", strconv.Itoa(p), "--timeout", "3s", "check"}, spamMail)
	if code != 1 {
		t.Fatalf("expected not-spam (exit 1), got %d; err=%q", code, errb)
	}
}

func TestCLIUnknownOp(t *testing.T) {
	code, _, errb := runCLI(t, []string{"bogus"}, "")
	if code != 2 || !strings.Contains(errb, "unknown op") {
		t.Errorf("expected usage error, code=%d err=%q", code, errb)
	}
}

func TestCLIUsage(t *testing.T) {
	if code, _, _ := runCLI(t, nil, ""); code != 2 {
		t.Errorf("no args should exit 2, got %d", code)
	}
}

func TestCLIVersion(t *testing.T) {
	code, out, _ := runCLI(t, []string{"--version"}, "")
	if code != 0 || !strings.Contains(out, "gazor") {
		t.Errorf("version: code=%d out=%q", code, out)
	}
}
