package base64

func (e *Encoding) Check(src []byte) error {
	if len(src) == 0 {
		return nil
	}

	i := 0
	for i < len(src) {
		// Find the next group boundary (4 chars at a time for padded, partial for unpadded)
		groupEnd := i + 4
		if groupEnd > len(src) {
			if !e.noPad {
				return ErrInvalidLength
			}
			groupEnd = len(src)
		}
		group := src[i:groupEnd]

		padStart := -1
		for j, b := range group {
			v := e.decMap[b]
			switch {
			case v >= 0:
				if padStart >= 0 {
					return ErrInvalidFormat
				}
			case v == padIdx:
				if e.noPad {
					return ErrInvalidChar
				}
				if padStart < 0 {
					padStart = j
				}
				if j == 0 {
					return ErrInvalidFormat
				}
			case v == invalidIdx:
				return ErrInvalidChar
			}
		}

		if padStart >= 0 {
			np := len(group) - padStart
			if np > 2 {
				return ErrInvalidFormat
			}
			if np == 2 && padStart != 2 {
				return ErrInvalidFormat
			}
		}

		i += len(group)
	}

	return nil
}
