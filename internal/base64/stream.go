package base64

import "io"

// Encoder streams base64-encoded output to an underlying writer.
type Encoder struct {
	enc  *Encoding
	w    io.Writer
	buf  [3]byte
	nbuf int
	col  int
	wrap int
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
	alpha := e.enc.alphabet

	for len(p) > 0 {
		n := copy(e.buf[e.nbuf:], p)
		e.nbuf += n
		p = p[n:]

		if e.nbuf < 3 {
			break
		}

		var out [4]byte
		encodeBlock(out[:], e.buf[:3], alpha)
		if err := e.flushWrapped(out[:]); err != nil {
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
	var out [4]byte
	encodeFinalBlock(out[:], e.buf[:e.nbuf], e.enc.alphabet, e.enc.noPad)

	n := 4
	if e.enc.noPad {
		n = 3
		if e.nbuf == 1 {
			n = 2
		}
	}
	if err := e.flushWrapped(out[:n]); err != nil {
		return err
	}
	return nil
}

func (e *Encoder) flushWrapped(data []byte) error {
	if e.wrap <= 0 {
		n, err := e.w.Write(data)
		e.col += n
		return err
	}
	for len(data) > 0 {
		if e.col > 0 && e.col >= e.wrap {
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

// Decoder streams base64-decoded output from an underlying reader.
type Decoder struct {
	enc           *Encoding
	r             io.Reader
	inBuf         [4]byte
	nbuf          int
	outBuf        []byte
	outOff        int
	ignoreGarbage bool
	eof           bool
	err           error
	rdBuf         [8192]byte
}

func NewDecoder(enc *Encoding, r io.Reader) *Decoder {
	return &Decoder{enc: enc, r: r}
}

func (d *Decoder) WithIgnoreGarbage(ignore bool) *Decoder {
	d.ignoreGarbage = ignore
	return d
}

func (d *Decoder) Read(p []byte) (int, error) {
	for {
		if d.outOff < len(d.outBuf) {
			n := copy(p, d.outBuf[d.outOff:])
			d.outOff += n
			return n, nil
		}

		if d.eof {
			if d.nbuf > 0 {
				d.decodeFinalBlock()
				d.nbuf = 0
			}
			if len(d.outBuf) > d.outOff {
				n := copy(p, d.outBuf[d.outOff:])
				d.outOff += n
				return n, nil
			}
			if d.err != nil {
				return 0, d.err
			}
			return 0, io.EOF
		}

		n, err := d.r.Read(d.rdBuf[:])
		if err != nil && err != io.EOF {
			d.err = err
			d.eof = true
			continue
		}
		if err == io.EOF {
			d.eof = true
		}

		d.processChunk(d.rdBuf[:n])

		if d.err != nil {
			d.eof = true
		}
	}
}

func (d *Decoder) processChunk(chunk []byte) {
	decMap := &d.enc.decMap

	for _, b := range chunk {
		// Always skip whitespace
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			continue
		}

		v := decMap[b]
		switch {
		case v >= 0:
			d.inBuf[d.nbuf] = b
			d.nbuf++
		case v == padIdx:
			d.inBuf[d.nbuf] = b
			d.nbuf++
		case v == invalidIdx:
			if d.ignoreGarbage {
				continue
			}
			d.err = ErrInvalidChar
			return
		}

		if d.nbuf == 4 {
			d.decodeBlock()
			d.nbuf = 0
		}
	}
}

func (d *Decoder) decodeBlock() {
	var vals [4]uint8
	np := 0
	for i := 0; i < 4; i++ {
		v := d.enc.decMap[d.inBuf[i]]
		if v == padIdx {
			np++
			vals[i] = 0
		} else {
			vals[i] = uint8(v)
		}
	}

	b24, nout := decodeBlock(vals, np)
	switch nout {
	case 3:
		d.outBuf = append(d.outBuf, byte(b24>>16), byte(b24>>8), byte(b24))
	case 2:
		d.outBuf = append(d.outBuf, byte(b24>>16), byte(b24>>8))
	case 1:
		d.outBuf = append(d.outBuf, byte(b24>>16))
	}
}

func (d *Decoder) decodeFinalBlock() {
	var fullBuf [4]byte
	copy(fullBuf[:], d.inBuf[:d.nbuf])

	// Count actual '=' padding chars in the buffer
	padCount := 0
	for i := d.nbuf - 1; i >= 0; i-- {
		if d.enc.decMap[fullBuf[i]] == padIdx {
			padCount++
		} else {
			break
		}
	}

	// For unpadded input, the number of missing chars is the padding
	if d.enc.noPad && padCount == 0 {
		padCount = 4 - d.nbuf
	}

	for i := d.nbuf; i < 4; i++ {
		fullBuf[i] = d.enc.alphabet[0]
	}

	var vals [4]uint8
	for i := 0; i < 4; i++ {
		v := d.enc.decMap[fullBuf[i]]
		if v < 0 {
			v = 0
		}
		vals[i] = uint8(v)
	}

	b24 := uint32(vals[0])<<18 | uint32(vals[1])<<12 | uint32(vals[2])<<6 | uint32(vals[3])
	nout := 3 - padCount
	switch nout {
	case 3:
		d.outBuf = append(d.outBuf, byte(b24>>16), byte(b24>>8), byte(b24))
	case 2:
		d.outBuf = append(d.outBuf, byte(b24>>16), byte(b24>>8))
	case 1:
		d.outBuf = append(d.outBuf, byte(b24>>16))
	}
}
