package base64

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"
)

var testVectors = []struct {
	name string
	data []byte
}{
	{"empty", []byte{}},
	{"single", []byte{0}},
	{"two", []byte{0, 255}},
	{"three", []byte{0, 255, 128}},
	{"hello", []byte("hello world")},
	{"binary", []byte{0x00, 0x01, 0x02, 0xFE, 0xFF}},
	{"pad1", []byte{0x61}},
	{"pad2", []byte{0x61, 0x62}},
	{"pad3", []byte{0x61, 0x62, 0x63}},
	{"nibbles", []byte("\x00\x10\x20\x30\x40\x50\x60\x70\x80\x90\xA0\xB0\xC0\xD0\xE0\xF0")},
}

func TestStdRoundtrip(t *testing.T) {
	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			enc := StdEncoding.Encode(tv.data)
			dec, err := StdEncoding.Decode(enc)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !bytes.Equal(tv.data, dec) {
				t.Fatalf("mismatch:\ngot:  %x\nwant: %x", dec, tv.data)
			}
		})
	}
}

func TestURLRoundtrip(t *testing.T) {
	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			enc := URLEncoding.Encode(tv.data)
			dec, err := URLEncoding.Decode(enc)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !bytes.Equal(tv.data, dec) {
				t.Fatalf("mismatch:\ngot:  %x\nwant: %x", dec, tv.data)
			}
		})
	}
}

func TestRawURLRoundtrip(t *testing.T) {
	for _, tv := range testVectors {
		t.Run(tv.name, func(t *testing.T) {
			if len(tv.data) == 0 {
				return
			}
			enc := RawURLEncoding.Encode(tv.data)
			for _, b := range enc {
				if b == '=' {
					t.Fatal("RawURLEncoding should not produce padding")
				}
			}
			dec, err := RawURLEncoding.Decode(enc)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !bytes.Equal(tv.data, dec) {
				t.Fatalf("mismatch:\ngot:  %x\nwant: %x", dec, tv.data)
			}
		})
	}
}

func TestURLAlphabet(t *testing.T) {
	if strings.ContainsAny(URLEncoding.Alphabet(), "+/") {
		t.Fatal("URL alphabet should not contain + or /")
	}
	if !strings.ContainsAny(URLEncoding.Alphabet(), "-_") {
		t.Fatal("URL alphabet should contain - and _")
	}
}

func TestCheck(t *testing.T) {
	valid := []string{
		"",
		"YW55",          // "any"
		"aGVsbG8=",      // "hello" with 1 pad
		"YQ==",          // "a" with 2 pad
		"YWJj",          // "abc"
		"YWJjZA==",      // "abcd" with 2 pad
		"SGVsbG8gV29ybGQ=", // "Hello World" with 1 pad
	}
	for _, c := range valid {
		if err := StdEncoding.Check([]byte(c)); err != nil {
			t.Errorf("Check(%q) = %v, want nil", c, err)
		}
	}

	invalid := []string{
		"!",
		"abc!",
		"YW5!5",
		"YW55===",  // 3 pad chars
		"Y===",     // pad starts too early
	}
	for _, c := range invalid {
		if err := StdEncoding.Check([]byte(c)); err == nil {
			t.Errorf("Check(%q) = nil, want error", c)
		}
	}
}

func TestStreamRoundtrip(t *testing.T) {
	sizes := []int{0, 1, 2, 3, 4, 5, 1024, 4096, 10000}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)

			var encBuf bytes.Buffer
			encw := NewEncoder(StdEncoding, &encBuf)
			if _, err := encw.Write(data); err != nil {
				t.Fatal(err)
			}
			if err := encw.Close(); err != nil {
				t.Fatal(err)
			}

			decr := NewDecoder(StdEncoding, &encBuf)
			decr.WithIgnoreGarbage(true)
			dec, err := io.ReadAll(decr)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data, dec) {
				t.Fatalf("stream roundtrip mismatch for size %d", size)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	data := make([]byte, 100)
	rand.Read(data)

	var buf bytes.Buffer
	encw := NewEncoder(StdEncoding, &buf)
	encw.WithWrap(20)
	if _, err := encw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := encw.Close(); err != nil {
		t.Fatal(err)
	}

	for _, line := range strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n") {
		if len(line) > 20 {
			t.Fatalf("line length %d > 20", len(line))
		}
	}

	decr := NewDecoder(StdEncoding, &buf)
	decr.WithIgnoreGarbage(true)
	dec, err := io.ReadAll(decr)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, dec) {
		t.Fatal("wrapped roundtrip mismatch")
	}
}

func TestIgnoreGarbage(t *testing.T) {
	garbled := "aGVs\nbG8g\td29ybGQ="
	r := NewDecoder(StdEncoding, strings.NewReader(garbled))
	r.WithIgnoreGarbage(true)
	dec, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ignore-garbage: %v", err)
	}
	if string(dec) != "hello world" {
		t.Fatalf("got %q, want %q", dec, "hello world")
	}
}

func TestLargeRandom(t *testing.T) {
	for _, size := range []int{1, 10, 100, 1000, 10000, 100000} {
		t.Run(fmt.Sprintf("%d", size), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)
			enc := StdEncoding.Encode(data)
			dec, err := StdEncoding.Decode(enc)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data, dec) {
				t.Fatal("roundtrip mismatch")
			}
		})
	}
}

func TestEncodedLen(t *testing.T) {
	for _, tc := range []struct{ n, want int }{
		{0, 0}, {1, 4}, {2, 4}, {3, 4}, {4, 8}, {5, 8}, {6, 8}, {7, 12},
	} {
		if got := StdEncoding.EncodedLen(tc.n); got != tc.want {
			t.Errorf("EncodedLen(%d) = %d, want %d", tc.n, got, tc.want)
		}
	}
}

func TestDecodedLen(t *testing.T) {
	for _, tc := range []struct{ n, want int }{
		{0, 0}, {4, 3}, {8, 6}, {12, 9},
	} {
		if got := StdEncoding.DecodedLen(tc.n); got != tc.want {
			t.Errorf("DecodedLen(%d) = %d, want %d", tc.n, got, tc.want)
		}
	}
}

func TestCrossSystemBase64(t *testing.T) {
	if _, err := exec.LookPath("base64"); err != nil {
		t.Skip("system base64 not available")
	}

	data := []byte("cross-validation data: \x00\x01\x02\xFF")
	ourEnc := StdEncoding.EncodeToString(data)

	cmd := exec.Command("base64", "-d")
	cmd.Stdin = strings.NewReader(ourEnc)
	sysDec, err := cmd.Output()
	if err != nil {
		t.Fatalf("system decode: %v", err)
	}
	if !bytes.Equal(data, sysDec) {
		t.Fatalf("system decode mismatch:\nwant: %x\ngot:  %x", data, sysDec)
	}

	cmd2 := exec.Command("base64")
	cmd2.Stdin = strings.NewReader(string(data))
	sysEnc, err := cmd2.Output()
	if err != nil {
		t.Fatalf("system encode: %v", err)
	}
	sysEnc = bytes.TrimSpace(sysEnc)

	ourDec, err := StdEncoding.Decode(sysEnc)
	if err != nil {
		t.Fatalf("our decode: %v", err)
	}
	if !bytes.Equal(data, ourDec) {
		t.Fatalf("our decode of system output mismatch:\nwant: %x\ngot:  %x", data, ourDec)
	}
}

// Benchmarks

func BenchmarkEncode(b *testing.B) {
	for _, size := range []int{64, 1024, 65536} {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				StdEncoding.EncodeToString(data)
			}
		})
	}
}

func BenchmarkDecode(b *testing.B) {
	for _, size := range []int{64, 1024, 65536} {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)
			enc := StdEncoding.EncodeToString(data)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				StdEncoding.DecodeString(enc)
			}
		})
	}
}

func BenchmarkStreamEncode(b *testing.B) {
	data := make([]byte, 65536)
	rand.Read(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		encw := NewEncoder(StdEncoding, &buf)
		encw.Write(data)
		encw.Close()
	}
}

func BenchmarkStreamDecode(b *testing.B) {
	data := make([]byte, 65536)
	rand.Read(data)
	enc := StdEncoding.EncodeToString(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewDecoder(StdEncoding, strings.NewReader(enc))
		r.WithIgnoreGarbage(true)
		io.ReadAll(r)
	}
}

func BenchmarkURLEncode(b *testing.B) {
	data := make([]byte, 65536)
	rand.Read(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		URLEncoding.EncodeToString(data)
	}
}

func BenchmarkURLDecode(b *testing.B) {
	data := make([]byte, 65536)
	rand.Read(data)
	enc := URLEncoding.EncodeToString(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		URLEncoding.DecodeString(enc)
	}
}
