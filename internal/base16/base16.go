// Package base16 implements hex encoding (RFC 4648 §8).
package base16

import (
	"errors"
	"io"
)

const hexChars = "0123456789ABCDEF"

var (
	ErrInvalidChar = errors.New("base16: invalid character")
)

type Encoding struct {
	encTable [16]byte
	decTable [256]int8
	upper    bool
}

var (
	StdEncoding = NewEncoding(true)
	LowerCase   = NewEncoding(false)
)

func NewEncoding(upper bool) *Encoding {
	e := &Encoding{upper: upper}
	chars := hexChars
	if !upper {
		chars = "0123456789abcdef"
	}
	for i := range e.decTable {
		e.decTable[i] = -1
	}
	for i := 0; i < 16; i++ {
		e.encTable[i] = chars[i]
		e.decTable[chars[i]] = int8(i)
	}
	// Accept both cases on decode
	for i := 0; i < 6; i++ {
		e.decTable[byte('a'+i)] = int8(10 + i)
		e.decTable[byte('A'+i)] = int8(10 + i)
	}
	for i := 0; i < 10; i++ {
		e.decTable[byte('0'+i)] = int8(i)
	}
	return e
}

func (e *Encoding) EncodedLen(n int) int { return n * 2 }

func (e *Encoding) DecodedLen(n int) int { return n / 2 }

func (e *Encoding) Encode(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	dst := make([]byte, len(src)*2)
	e.encode(dst, src)
	return dst
}

func (e *Encoding) EncodeToString(src []byte) string {
	return string(e.Encode(src))
}

func (e *Encoding) encode(dst, src []byte) {
	for i, b := range src {
		dst[i*2] = e.encTable[b>>4]
		dst[i*2+1] = e.encTable[b&0x0F]
	}
}

func (e *Encoding) Decode(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}
	if len(src)%2 != 0 {
		return nil, errors.New("base16: odd input length")
	}
	dst := make([]byte, len(src)/2)
	for i := 0; i < len(src); i += 2 {
		hi := e.decTable[src[i]]
		lo := e.decTable[src[i+1]]
		if hi < 0 || lo < 0 {
			return nil, ErrInvalidChar
		}
		dst[i/2] = byte(hi<<4) | byte(lo)
	}
	return dst, nil
}

func (e *Encoding) DecodeString(s string) ([]byte, error) {
	return e.Decode([]byte(s))
}

func (e *Encoding) Check(src []byte) error {
	if len(src)%2 != 0 {
		return errors.New("base16: odd input length")
	}
	for _, b := range src {
		if e.decTable[b] < 0 {
			return ErrInvalidChar
		}
	}
	return nil
}

// Encoder streams hex encoding to a writer.
type Encoder struct {
	enc *Encoding
	w   io.Writer
}

func NewEncoder(enc *Encoding, w io.Writer) *Encoder {
	return &Encoder{enc: enc, w: w}
}

func (e *Encoder) Write(p []byte) (int, error) {
	encoded := e.enc.Encode(p)
	if len(encoded) == 0 {
		return len(p), nil
	}
	if _, err := e.w.Write(encoded); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (e *Encoder) Close() error { return nil }
