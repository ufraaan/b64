// Package base64 implements base64 encoding as specified by RFC 4648.
package base64

import "errors"

const (
	stdAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	urlAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	padChar     = byte('=')
)

const (
	invalidIdx int8 = -1
	padIdx     int8 = -2
)

const NoPadding = -1

var (
	ErrInvalidFormat = errors.New("base64: invalid input format")
	ErrInvalidLength = errors.New("base64: invalid input length")
	ErrInvalidChar   = errors.New("base64: invalid character")
)

// Encoding represents a base64 encoding scheme defined by a 64-character alphabet.
type Encoding struct {
	alphabet string
	decMap   [256]int8
	pad      byte
	noPad    bool
	strict   bool
}

var (
	StdEncoding    = NewEncoding(stdAlphabet)
	URLEncoding    = NewEncoding(urlAlphabet)
	RawURLEncoding = URLEncoding.WithPadding(NoPadding)
)

func NewEncoding(alphabet string) *Encoding {
	e := &Encoding{alphabet: alphabet, pad: padChar}
	for i := range e.decMap {
		e.decMap[i] = invalidIdx
	}
	for i := 0; i < len(alphabet); i++ {
		e.decMap[alphabet[i]] = int8(i)
	}
	e.decMap[padChar] = padIdx
	return e
}

func (e *Encoding) WithPadding(pad int) *Encoding {
	cp := *e
	if pad == NoPadding {
		cp.noPad = true
		cp.pad = 0
		cp.decMap[padChar] = invalidIdx
	} else {
		cp.noPad = false
		cp.pad = byte(pad)
		cp.decMap[byte(pad)] = padIdx
	}
	return &cp
}

func (e *Encoding) Strict() *Encoding {
	cp := *e
	cp.strict = true
	return &cp
}

func (e *Encoding) Alphabet() string     { return e.alphabet }
func (e *Encoding) UsesPadding() bool    { return !e.noPad }

func (e *Encoding) EncodedLen(n int) int {
	if e.noPad {
		return (n*8 + 5) / 6
	}
	return (n + 2) / 3 * 4
}

func (e *Encoding) DecodedLen(n int) int {
	return n / 4 * 3
}
