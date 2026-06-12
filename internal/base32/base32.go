// Package base32 implements base32 encoding as specified by RFC 4648.
package base32

import (
	"errors"
	"io"
)

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
const padByte = '='

const (
	invalidIdx = -1
	padIdx     = -2
)

const NoPadding = -1

var (
	ErrInvalidFormat = errors.New("base32: invalid input format")
	ErrInvalidLength = errors.New("base32: invalid input length")
	ErrInvalidChar   = errors.New("base32: invalid character")
)

type Encoding struct {
	alphabet string
	decMap   [256]int8
	pad      byte
	noPad    bool
}

var StdEncoding = NewEncoding(alphabet)
var HexEncoding = NewEncoding("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func NewEncoding(alphabet string) *Encoding {
	e := &Encoding{alphabet: alphabet, pad: padByte}
	for i := range e.decMap {
		e.decMap[i] = invalidIdx
	}
	for i := 0; i < len(alphabet); i++ {
		e.decMap[alphabet[i]] = int8(i)
	}
	e.decMap[padByte] = padIdx
	return e
}

func (e *Encoding) WithPadding(pad int) *Encoding {
	cp := *e
	if pad == NoPadding {
		cp.noPad = true
		cp.pad = 0
		cp.decMap[padByte] = invalidIdx
	} else {
		cp.noPad = false
		cp.pad = byte(pad)
		cp.decMap[byte(pad)] = padIdx
	}
	return &cp
}

func (e *Encoding) Alphabet() string  { return e.alphabet }
func (e *Encoding) UsesPadding() bool { return !e.noPad }

// EncodedLen returns the encoded length for n source bytes.
func (e *Encoding) EncodedLen(n int) int {
	if e.noPad {
		return (n*8 + 4) / 5
	}
	return (n + 4) / 5 * 8
}

// DecodedLen returns the maximum decoded length for n encoded bytes.
func (e *Encoding) DecodedLen(n int) int {
	return n / 8 * 5
}

func (e *Encoding) Encode(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	n := e.EncodedLen(len(src))
	dst := make([]byte, n)
	e.encode(dst, src)
	return dst
}

func (e *Encoding) EncodeToString(src []byte) string {
	return string(e.Encode(src))
}

func (e *Encoding) encode(dst, src []byte) {
	alpha := e.alphabet
	noPad := e.noPad

	srcoff := 0
	dstoff := 0

	for len(src)-srcoff >= 5 {
		b0, b1, b2, b3, b4 := src[srcoff], src[srcoff+1], src[srcoff+2], src[srcoff+3], src[srcoff+4]
		dst[dstoff] = alpha[b0>>3]
		dst[dstoff+1] = alpha[((b0<<2)|(b1>>6))&0x1F]
		dst[dstoff+2] = alpha[(b1>>1)&0x1F]
		dst[dstoff+3] = alpha[((b1<<4)|(b2>>4))&0x1F]
		dst[dstoff+4] = alpha[((b2<<1)|(b3>>7))&0x1F]
		dst[dstoff+5] = alpha[(b3>>2)&0x1F]
		dst[dstoff+6] = alpha[((b3<<3)|(b4>>5))&0x1F]
		dst[dstoff+7] = alpha[b4&0x1F]
		srcoff += 5
		dstoff += 8
	}

	remaining := len(src) - srcoff
	if remaining == 0 {
		return
	}

	var buf [5]byte
	copy(buf[:], src[srcoff:])

	switch remaining {
	case 1:
		dst[dstoff] = alpha[buf[0]>>3]
		dst[dstoff+1] = alpha[(buf[0]<<2)&0x1F]
		if !noPad {
			dst[dstoff+2] = padByte
			dst[dstoff+3] = padByte
			dst[dstoff+4] = padByte
			dst[dstoff+5] = padByte
			dst[dstoff+6] = padByte
			dst[dstoff+7] = padByte
		}
	case 2:
		dst[dstoff] = alpha[buf[0]>>3]
		dst[dstoff+1] = alpha[((buf[0]<<2)|(buf[1]>>6))&0x1F]
		dst[dstoff+2] = alpha[(buf[1]>>1)&0x1F]
		dst[dstoff+3] = alpha[(buf[1]<<4)&0x1F]
		if !noPad {
			dst[dstoff+4] = padByte
			dst[dstoff+5] = padByte
			dst[dstoff+6] = padByte
			dst[dstoff+7] = padByte
		}
	case 3:
		dst[dstoff] = alpha[buf[0]>>3]
		dst[dstoff+1] = alpha[((buf[0]<<2)|(buf[1]>>6))&0x1F]
		dst[dstoff+2] = alpha[(buf[1]>>1)&0x1F]
		dst[dstoff+3] = alpha[((buf[1]<<4)|(buf[2]>>4))&0x1F]
		dst[dstoff+4] = alpha[(buf[2]<<1)&0x1F]
		if !noPad {
			dst[dstoff+5] = padByte
			dst[dstoff+6] = padByte
			dst[dstoff+7] = padByte
		}
	case 4:
		dst[dstoff] = alpha[buf[0]>>3]
		dst[dstoff+1] = alpha[((buf[0]<<2)|(buf[1]>>6))&0x1F]
		dst[dstoff+2] = alpha[(buf[1]>>1)&0x1F]
		dst[dstoff+3] = alpha[((buf[1]<<4)|(buf[2]>>4))&0x1F]
		dst[dstoff+4] = alpha[((buf[2]<<1)|(buf[3]>>7))&0x1F]
		dst[dstoff+5] = alpha[(buf[3]>>2)&0x1F]
		dst[dstoff+6] = alpha[(buf[3]<<3)&0x1F]
		if !noPad {
			dst[dstoff+7] = padByte
		}
	}
}

func (e *Encoding) Decode(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}
	maxLen := len(src)*5/8 + 5
	dst := make([]byte, maxLen)
	n, err := e.decode(dst, src)
	if err != nil {
		return nil, err
	}
	return dst[:n], nil
}

func (e *Encoding) DecodeString(s string) ([]byte, error) {
	return e.Decode([]byte(s))
}

func (e *Encoding) decode(dst []byte, src []byte) (int, error) {
	decMap := &e.decMap
	off := 0
	i := 0

	for i < len(src) {
		var vals [8]uint8
		np := 0
		found := 0

		for found < 8 && i < len(src) {
			c := src[i]
			i++
			v := decMap[c]
			switch {
			case v >= 0:
				vals[found] = uint8(v)
				found++
			case v == padIdx:
				vals[found] = 0
				found++
				np++
			case v == invalidIdx:
				continue
			}
		}

		if found == 0 {
			continue
		}

		if found < 8 {
			if !e.noPad {
				return 0, ErrInvalidLength
			}
			np = 8 - found
			for found < 8 {
				vals[found] = 0
				found++
			}
		}

		var out [5]byte
		out[0] = (vals[0] << 3) | (vals[1] >> 2)
		out[1] = (vals[1] << 6) | (vals[2] << 1) | (vals[3] >> 4)
		out[2] = (vals[3] << 4) | (vals[4] >> 1)
		out[3] = (vals[4] << 7) | (vals[5] << 2) | (vals[6] >> 3)
		out[4] = (vals[6] << 5) | vals[7]

		nout := (8 - np) * 5 / 8
		if off+nout > len(dst) {
			return 0, ErrInvalidLength
		}
		for j := 0; j < nout; j++ {
			dst[off+j] = out[j]
		}
		off += nout
	}

	return off, nil
}

func (e *Encoding) Check(src []byte) error {
	decMap := &e.decMap
	i := 0
	for i < len(src) {
		vals := 0
		np := 0
		for vals < 8 && i < len(src) {
			c := src[i]
			i++
			v := decMap[c]
			switch {
			case v >= 0:
				vals++
			case v == padIdx:
				if e.noPad {
					return ErrInvalidChar
				}
				vals++
				np++
			case v == invalidIdx:
				return ErrInvalidChar
			}
		}
		if vals == 0 {
			continue
		}
		if vals < 8 && !e.noPad {
			return ErrInvalidLength
		}
	}
	return nil
}

// --- Streaming ---

type Encoder struct {
	enc   *Encoding
	w     io.Writer
	buf   [5]byte
	nbuf  int
	wrap  int
	col   int
}

func NewEncoder(enc *Encoding, w io.Writer) *Encoder {
	return &Encoder{enc: enc, w: w}
}

func (e *Encoder) WithWrap(cols int) *Encoder {
	e.wrap = cols
	return e
}

func (e *Encoder) Write(p []byte) (int, error) {
	total := len(p)
	for len(p) > 0 {
		n := copy(e.buf[e.nbuf:], p)
		e.nbuf += n
		p = p[n:]
		if e.nbuf < 5 {
			break
		}
		var out [8]byte
		e.encodeBlock(out[:], e.buf[:5])
		if err := e.flush(out[:]); err != nil {
			return total - len(p), err
		}
		e.nbuf = 0
	}
	return total, nil
}

func (e *Encoder) Close() error {
	if e.nbuf == 0 {
		return nil
	}
	var out [8]byte
	e.encodeFinalBlock(out[:], e.buf[:e.nbuf])
	n := 8
	if e.enc.noPad {
		switch e.nbuf {
		case 1:
			n = 2
		case 2:
			n = 4
		case 3:
			n = 5
		case 4:
			n = 7
		}
	}
	return e.flush(out[:n])
}

func (e *Encoder) encodeBlock(dst []byte, src []byte) {
	alpha := e.enc.alphabet
	dst[0] = alpha[src[0]>>3]
	dst[1] = alpha[((src[0]<<2)|(src[1]>>6))&0x1F]
	dst[2] = alpha[(src[1]>>1)&0x1F]
	dst[3] = alpha[((src[1]<<4)|(src[2]>>4))&0x1F]
	dst[4] = alpha[((src[2]<<1)|(src[3]>>7))&0x1F]
	dst[5] = alpha[(src[3]>>2)&0x1F]
	dst[6] = alpha[((src[3]<<3)|(src[4]>>5))&0x1F]
	dst[7] = alpha[src[4]&0x1F]
}

func (e *Encoder) encodeFinalBlock(dst []byte, src []byte) {
	// Uses Encode's final block logic directly
	tmp := e.enc.Encode(src)
	copy(dst, tmp)
}

func (e *Encoder) flush(data []byte) error {
	if e.wrap <= 0 {
		n, err := e.w.Write(data)
		e.col += n
		return err
	}
	for len(data) > 0 {
		if e.col >= e.wrap {
			if _, err := e.w.Write([]byte{'\n'}); err != nil {
				return err
			}
			e.col = 0
		}
		space := e.wrap - e.col
		if space > len(data) {
			space = len(data)
		}
		n, err := e.w.Write(data[:space])
		e.col += n
		if err != nil {
			return err
		}
		data = data[space:]
	}
	return nil
}
