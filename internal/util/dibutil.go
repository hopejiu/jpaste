package util

import "encoding/binary"

// PrependBMPHeader builds a valid BMP from a DIB by prefixing BITMAPFILEHEADER.
func PrependBMPHeader(dib []byte) []byte {
	headerSize := binary.LittleEndian.Uint32(dib[0:4]) // biSize
	bitCount := binary.LittleEndian.Uint16(dib[14:16])
	clrUsed := binary.LittleEndian.Uint32(dib[32:36])

	var colorTableSize uint32
	if bitCount <= 8 {
		if clrUsed == 0 {
			colorTableSize = uint32(1<<bitCount) * 4
		} else {
			colorTableSize = clrUsed * 4
		}
	}

	offset := uint32(14 + headerSize + colorTableSize)
	fileSize := uint32(14 + len(dib))

	buf := make([]byte, 14+len(dib))
	binary.LittleEndian.PutUint16(buf[0:2], 0x4D42) // 'BM'
	binary.LittleEndian.PutUint32(buf[2:6], fileSize)
	binary.LittleEndian.PutUint32(buf[10:14], offset)
	copy(buf[14:], dib)

	return buf
}
