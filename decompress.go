package main

import (
	"encoding/binary"
	"log"
)

func decompress(data []byte) (out []byte) {
	// Only support binary compression and 4096 dictionary size
	if len(data) < 6 {
		log.Fatalln("Sector too small")
	}
	if data[0] != IMPLODE_BINARY {
		log.Fatalln("Only supports IMPLODE_BINARY not:", data[0])
	}
	if data[1] != IMPLODE_DICT_4K {
		log.Fatalln("Only supports dictionary size 4096 not:", data[1])
	}
	out = make([]byte, 4096)

	bitBuf := binary.LittleEndian.Uint32(data[2:])
	var bitsInBuf uint32 = 32

	var outIndex, copy_off, i uint32
	dataIndex := 6

	for dataIndex < len(data) {
		// Fill buffer if needed
		for bitsInBuf < 16 {
			bitBuf += uint32(data[dataIndex]) << bitsInBuf
			dataIndex++
			bitsInBuf += 8
		}

		if bitBuf&1 == 1 {
			// Not Literal
			// Shift one bit extra in the switch statments below

			// Old Code
			// bitBuf >>= 1
			// bitsInBuf -= 1
			//
			// for i = 0; i <= 0x0F; i++ {
			// 	   if ((bitBuf) & ((1 << (s_LenBits[i])) - 1)) == uint32(s_LenCode[i]) {
			// 		   break
			//     }
			// }
			//
			// bitBuf >>= s_LenBits[i]
			// bitsInBuf -= uint(s_LenBits[i])

			// Use 2 bits and skip 1 bit from Literal byte check
			if bitBuf&0x06 == 0x06 {
				bitBuf >>= 3
				bitsInBuf -= 3
				i = 1
				goto setCopyLen
			}

			// Use 3 bits and skip 1 bit from Literal byte check
			switch bitBuf & 0x0E {
			case 0x0a:
				bitBuf >>= 4
				bitsInBuf -= 4
				i = 0
				goto setCopyLen
			case 0x02:
				bitBuf >>= 4
				bitsInBuf -= 4
				i = 2
				goto setCopyLen
			case 0x0c:
				bitBuf >>= 4
				bitsInBuf -= 4
				i = 3
				goto setCopyLen
			}

			// Use 4 bits and skip 1 bit from Literal byte check
			switch bitBuf & 0x1E {
			case 0x14:
				bitBuf >>= 5
				bitsInBuf -= 5
				i = 4
				goto setCopyLen
			case 0x04:
				bitBuf >>= 5
				bitsInBuf -= 5
				i = 5
				goto setCopyLen
			case 0x18:
				bitBuf >>= 5
				bitsInBuf -= 5
				i = 6
				goto setCopyLen
			}

			// Use 5 bits and skip 1 bit from Literal byte check
			switch bitBuf & 0x3E {
			case 0x28:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 7
				goto setCopyLen
			case 0x08:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 8
				goto setCopyLen
			case 0x30:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 9
				goto setCopyLen
			case 0x10:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 10
				goto setCopyLen
			}

			// Use 6 bits and skip 1 bit from Literal byte check
			switch bitBuf & 0x7E {
			case 0x60:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 11
				goto setCopyLen
			case 0x20:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 12
				goto setCopyLen
			case 0x40:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 13
				goto setCopyLen
			}

			// Use 7 bits and skip 1 bit from Literal byte check
			switch bitBuf & 0xFE {
			case 0x80:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 14
			case 0x00:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 15
			}

		setCopyLen:
			// ###TODO### potential gain of unpacking shift, and setting copy_len is 225ms for diabdat.mpq
			copy_len := s_LenBase[i] + ((bitBuf) & ((1 << (s_ExLenBits[i])) - 1))
			bitBuf >>= s_ExLenBits[i]
			bitsInBuf -= s_ExLenBits[i]

			if copy_len == 519 {
				return
			}

			// fill buffer if needed
			for bitsInBuf < 14 {
				bitBuf += uint32(data[dataIndex]) << bitsInBuf
				dataIndex++
				bitsInBuf += 8
			}

			// Old version
			// Find most significant 6 bits of offset into the dictionary
			//for i = 0; i <= 0x3f; i++ {
			//	if ((bitBuf) & ((1 << (s_OffsBits[i])) - 1)) == uint32(s_OffsCode[i]) {
			//		break
			//	}
			//}
			//bitBuf >>= s_OffsBits[i]
			//bitsInBuf -= uint(s_OffsBits[i])

			// 2 bits
			if bitBuf&0x03 == 0x03 {
				bitBuf >>= 2
				bitsInBuf -= 2
				i = 0
				goto setCopyOff
			}

			// 4 bits
			switch bitBuf & 0x0F {
			case 0x0d:
				bitBuf >>= 4
				bitsInBuf -= 4
				i = 1
				goto setCopyOff
			case 0x05:
				bitBuf >>= 4
				bitsInBuf -= 4
				i = 2
				goto setCopyOff
			}

			// 5 bits
			switch bitBuf & 0x1F {
			case 0x19:
				bitBuf >>= 5
				bitsInBuf -= 5
				i = 3
				goto setCopyOff
			case 0x09:
				bitBuf >>= 5
				bitsInBuf -= 5
				i = 4
				goto setCopyOff
			case 0x11:
				bitBuf >>= 5
				bitsInBuf -= 5
				i = 5
				goto setCopyOff
			case 0x01:
				bitBuf >>= 5
				bitsInBuf -= 5
				i = 6
				goto setCopyOff
			}

			// 6 bits
			switch bitBuf & 0x3F {
			case 0x3e:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 7
				goto setCopyOff
			case 0x1e:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 8
				goto setCopyOff
			case 0x2e:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 9
				goto setCopyOff
			case 0x0e:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 10
				goto setCopyOff
			case 0x36:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 11
				goto setCopyOff
			case 0x16:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 12
				goto setCopyOff
			case 0x26:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 13
				goto setCopyOff
			case 0x06:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 14
				goto setCopyOff
			case 0x3a:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 15
				goto setCopyOff
			case 0x1a:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 16
				goto setCopyOff
			case 0x2a:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 17
				goto setCopyOff
			case 0x0a:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 18
				goto setCopyOff
			case 0x32:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 19
				goto setCopyOff
			case 0x12:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 20
				goto setCopyOff
			case 0x22:
				bitBuf >>= 6
				bitsInBuf -= 6
				i = 21
				goto setCopyOff
			}

			// 7 bits
			switch bitBuf & 0x7F {
			case 0x42:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 22
				goto setCopyOff
			case 0x02:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 23
				goto setCopyOff
			case 0x7c:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 24
				goto setCopyOff
			case 0x3c:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 25
				goto setCopyOff
			case 0x5c:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 26
				goto setCopyOff
			case 0x1c:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 27
				goto setCopyOff
			case 0x6c:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 28
				goto setCopyOff
			case 0x2c:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 29
				goto setCopyOff
			case 0x4c:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 30
				goto setCopyOff
			case 0x0c:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 31
				goto setCopyOff
			case 0x74:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 32
				goto setCopyOff
			case 0x34:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 33
				goto setCopyOff
			case 0x54:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 34
				goto setCopyOff
			case 0x14:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 35
				goto setCopyOff
			case 0x64:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 36
				goto setCopyOff
			case 0x24:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 37
				goto setCopyOff
			case 0x44:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 38
				goto setCopyOff
			case 0x04:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 39
				goto setCopyOff
			case 0x78:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 40
				goto setCopyOff
			case 0x38:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 41
				goto setCopyOff
			case 0x58:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 42
				goto setCopyOff
			case 0x18:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 43
				goto setCopyOff
			case 0x68:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 44
				goto setCopyOff
			case 0x28:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 45
				goto setCopyOff
			case 0x48:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 46
				goto setCopyOff
			case 0x08:
				bitBuf >>= 7
				bitsInBuf -= 7
				i = 47
				goto setCopyOff
			}

			// 8 bits
			switch bitBuf & 0xFF {
			case 0xf0:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 48
			case 0x70:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 49
			case 0xb0:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 50
			case 0x30:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 51
			case 0xd0:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 52
			case 0x50:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 53
			case 0x90:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 54
			case 0x10:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 55
			case 0xe0:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 56
			case 0x60:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 57
			case 0xa0:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 58
			case 0x20:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 59
			case 0xc0:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 60
			case 0x40:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 61
			case 0x80:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 62
			case 0x00:
				bitBuf >>= 8
				bitsInBuf -= 8
				i = 63
			}

		setCopyOff:
			if copy_len == 2 {
				copy_off = outIndex - 1 - ((i << 2) + (bitBuf & 0x03))
				bitBuf >>= 2
				bitsInBuf -= 2
			} else {
				copy_off = outIndex - 1 - ((i << 6) + (bitBuf & 0x3F))
				bitBuf >>= 6
				bitsInBuf -= 6
			}

			// ###TODO### find best magic number:
			// len 8 and more matches 635857 cases in dabdat.mpq.
			// len 16 and more matches 171379 cases in dabdat.mpq.
			// copy has an overhead however is faster on big copys and we can also skip 2n inc of copy_off and outIndex
			if copy_len > 8 && outIndex-copy_off >= copy_len {
				copy(out[outIndex:outIndex+copy_len], out[copy_off:copy_off+copy_len])
				outIndex += copy_len
			} else {
				for i = 0; i < copy_len; i++ {
					out[outIndex] = out[copy_off]
					copy_off++
					outIndex++
				}
			}
		} else {
			// Literal
			out[outIndex] = byte((bitBuf >> 1) & 0xFF)
			outIndex++
			bitBuf >>= 9
			bitsInBuf -= 9
		}
	}
	return
}
