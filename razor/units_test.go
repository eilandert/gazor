package razor

import (
	"reflect"
	"sort"
	"testing"
)

func TestHexToBase64Roundtrip(t *testing.T) {
	// 40-hex SHA1-length inputs round-trip through razor's truncation rules.
	for _, h := range []string{
		"a1b2c3d4e5f60718293a4b5c6d7e8f9012345678",
		"0000000000000000000000000000000000000000",
		"ffffffffffffffffffffffffffffffffffffffff",
	} {
		if got := base64tohex(hextobase64(h)); got != h {
			t.Errorf("roundtrip %s -> %q -> %s", h, hextobase64(h), got)
		}
	}
}

func TestMakesisGolden(t *testing.T) {
	got := makesis(map[string]string{"a": "c", "e": "4", "ep4": "7542-10", "s": "AbC-_d"})
	if want := "a=c&e=4&ep4=7542-10&s=AbC-_d\r\n"; got != want {
		t.Errorf("makesis = %q, want %q", got, want)
	}
	got = makesisNue(map[string]string{"a": "c", "e": "4", "s": "x y"})
	if want := "a=c&e=4&s=x y\r\n"; got != want {
		t.Errorf("makesisNue = %q, want %q", got, want)
	}
}

func TestParsesisRoundtrip(t *testing.T) {
	in := map[string]string{"a": "c", "e": "4", "ep4": "7542-10", "s": "AbC-_d"}
	out := parsesis(makesis(in))
	if !reflect.DeepEqual(in, out) {
		t.Errorf("parsesis(makesis(x)) = %v, want %v", out, in)
	}
}

func TestToBatchedQueryGolden(t *testing.T) {
	var q []map[string]string
	for _, s := range []string{"S1", "S2", "S3", "S4", "S5"} {
		q = append(q, map[string]string{"a": "c", "e": "8", "s": s})
	}
	got := toBatchedQuery(q, 4, 128, true)
	want := []string{
		"-a=c&e=8&s=S1\r\na=c&e=8&s=S2\r\na=c&e=8&s=S3\r\na=c&e=8&s=S4\r\n.\r\n",
		"a=c&e=8&s=S5\r\n",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("toBatchedQuery =\n  %q\nwant\n  %q", got, want)
	}
	// from_batched_query must invert the multi-line batch back to 4 queries.
	back := fromBatchedQuery("-a=c&e=8&s=S1\r\na=c&e=8&s=S2\r\na=c&e=8&s=S3\r\na=c&e=8&s=S4")
	if len(back) != 4 || back[0]["s"] != "S1" || back[3]["s"] != "S4" || back[2]["e"] != "8" {
		t.Errorf("fromBatchedQuery roundtrip = %v", back)
	}
}

func TestHmacSHA1Golden(t *testing.T) {
	iv1, iv2 := xorKey("secretpass")
	if got := hmacSHA1("challenge123", iv1, iv2); got != "Mgf4tIPCd1GAe3umtAsU-Sv3_8YA" {
		t.Errorf("hmacSHA1 = %q, want %q", got, "Mgf4tIPCd1GAe3umtAsU-Sv3_8YA")
	}
}

func TestHexbits2hash(t *testing.T) {
	cases := map[string][]int{"88": {4, 8}, "14": {3, 5}}
	for hexstr, want := range cases {
		h := hexbits2hash(hexstr)
		var got []int
		for k := range h {
			got = append(got, k)
		}
		sort.Ints(got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("hexbits2hash(%s) = %v, want %v", hexstr, got, want)
		}
	}
}
