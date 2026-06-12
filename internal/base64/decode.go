package base64

func (e *Encoding) Decode(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}
	maxLen := len(src)*3/4 + 3
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
		var vals [4]uint8
		np := 0
		found := 0

		for found < 4 && i < len(src) {
			c := src[i]
			i++
			v := decMap[c]
			switch {
			case v >= 0:
				vals[found] = uint8(v)
				found++
				if np > 0 && e.strict {
					return 0, ErrInvalidFormat
				}
			case v == padIdx:
				vals[found] = 0
				found++
				np++
			case v == invalidIdx:
				if e.strict {
					return 0, ErrInvalidChar
				}
				continue
			}
		}

		if found == 0 {
			continue
		}

		originalFound := found
		if found < 4 {
			if !e.noPad {
				return 0, ErrInvalidLength
			}
			// Unpadded input: pad remaining positions with zero value
			np = 4 - originalFound
			for found < 4 {
				vals[found] = 0
				found++
			}
		}

		b24 := uint32(vals[0])<<18 | uint32(vals[1])<<12 | uint32(vals[2])<<6 | uint32(vals[3])
		nout := 3 - np

		if off+nout > len(dst) {
			return 0, ErrInvalidLength
		}

		if nout >= 1 {
			dst[off] = byte(b24 >> 16)
		}
		if nout >= 2 {
			dst[off+1] = byte(b24 >> 8)
		}
		if nout >= 3 {
			dst[off+2] = byte(b24)
		}
		off += nout
	}

	return off, nil
}

func decodeBlock(vals [4]uint8, np int) (uint32, int) {
	b24 := uint32(vals[0])<<18 | uint32(vals[1])<<12 | uint32(vals[2])<<6 | uint32(vals[3])
	nout := 3 - np
	return b24, nout
}
