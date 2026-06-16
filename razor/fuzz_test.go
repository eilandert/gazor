package razor

import "testing"

// FuzzSignatures throws arbitrary bytes at the full offline pipeline (prep_mail
// + MIME split + preproc + signature engines). Email is untrusted input, so the
// pipeline must never panic regardless of how malformed the message is.
func FuzzSignatures(f *testing.F) {
	f.Add([]byte("Subject: x\n\nhello http://a.example.com/ body\n"))
	f.Add([]byte("Content-Type: multipart/alternative; boundary=\"B\"\n\n--B\nContent-Type: text/html\n\n<HTML>&amp;<A HREF=http://x.example.org/>z</A>\n--B--\n"))
	f.Add([]byte("Content-Transfer-Encoding: base64\n\nQUJDRA==\n"))
	f.Add([]byte(""))
	f.Add([]byte("\x00\x01\x02--notaboundary\r\n=3D=0A"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		_ = Signatures(raw)
	})
}
