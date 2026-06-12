package base64

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
	alphabet := e.alphabet
	noPad := e.noPad

	srcoff := 0
	dstoff := 0
	remaining := len(src)

	for remaining >= 12 {
		for i := 0; i < 4; i++ {
			b0, b1, b2 := src[srcoff], src[srcoff+1], src[srcoff+2]
			dst[dstoff] = alphabet[b0>>2]
			dst[dstoff+1] = alphabet[((b0<<4)|(b1>>4))&0x3F]
			dst[dstoff+2] = alphabet[((b1<<2)|(b2>>6))&0x3F]
			dst[dstoff+3] = alphabet[b2&0x3F]
			srcoff += 3
			dstoff += 4
		}
		remaining -= 12
	}

	for remaining >= 3 {
		b0, b1, b2 := src[srcoff], src[srcoff+1], src[srcoff+2]
		dst[dstoff] = alphabet[b0>>2]
		dst[dstoff+1] = alphabet[((b0<<4)|(b1>>4))&0x3F]
		dst[dstoff+2] = alphabet[((b1<<2)|(b2>>6))&0x3F]
		dst[dstoff+3] = alphabet[b2&0x3F]
		srcoff += 3
		dstoff += 4
		remaining -= 3
	}

	if remaining == 1 {
		b0 := src[srcoff]
		dst[dstoff] = alphabet[b0>>2]
		dst[dstoff+1] = alphabet[(b0<<4)&0x3F]
		if !noPad {
			dst[dstoff+2] = padChar
			dst[dstoff+3] = padChar
		}
	} else if remaining == 2 {
		b0, b1 := src[srcoff], src[srcoff+1]
		dst[dstoff] = alphabet[b0>>2]
		dst[dstoff+1] = alphabet[((b0<<4)|(b1>>4))&0x3F]
		dst[dstoff+2] = alphabet[(b1<<2)&0x3F]
		if !noPad {
			dst[dstoff+3] = padChar
		}
	}
}

func encodeBlock(dst []byte, src []byte, alpha string) {
	dst[0] = alpha[src[0]>>2]
	dst[1] = alpha[((src[0]<<4)|(src[1]>>4))&0x3F]
	dst[2] = alpha[((src[1]<<2)|(src[2]>>6))&0x3F]
	dst[3] = alpha[src[2]&0x3F]
}

func encodeFinalBlock(dst []byte, src []byte, alpha string, noPad bool) {
	switch len(src) {
	case 1:
		dst[0] = alpha[src[0]>>2]
		dst[1] = alpha[(src[0]<<4)&0x3F]
		if !noPad {
			dst[2] = padChar
			dst[3] = padChar
		}
	case 2:
		dst[0] = alpha[src[0]>>2]
		dst[1] = alpha[((src[0]<<4)|(src[1]>>4))&0x3F]
		dst[2] = alpha[(src[1]<<2)&0x3F]
		if !noPad {
			dst[3] = padChar
		}
	}
}
