package frame

import (
	"encoding/binary"
	"fmt"
	"io"
)

// enCodeUint16 打码uint16转成两字节
func enCodeUint16(n uint16) []byte {
	bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(bytes, n)
	return bytes
}

// enCodeUint32 解码uint32转成四字节
func enCodeUint32(n uint32) []byte {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, n)
	return bytes
}

// enCodeUin64 解码uint64转成8字节
func enCodeUin64(n uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, n)
	return bytes
}

// readUint16 读取uint16
func readUint16(r io.Reader) (uint16, error) {
	bs := make([]byte, 2)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(bs), nil
}

// readUint32   读取uint32
func readUint32(r io.Reader) (uint32, error) {
	bs := make([]byte, 4)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(bs), nil
}

// readUint64  读取uint64
func readUint64(r io.Reader) (uint64, error) {
	bs := make([]byte, 8)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(bs), nil
}

func readBytes(r io.Reader, lengths ...int) ([]byte, error) {
	length := 1
	if lengths != nil && len(lengths) == 1 && lengths[0] >= 1 {
		length = lengths[0]
	}
	bs := make([]byte, length)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func readByte(r io.Reader) (byte, error) {
	bs := make([]byte, 1)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return bs[0], nil
}

// packetBytesDivision Bytes分割
// bs                  []byte
// buffLen             分割长度
func packetBytesDivision(bs []byte, buffLen int) [][]byte {
	var list [][]byte
	if bs == nil || len(bs) < 1 {
		return list
	}
	if buffLen < 1 || len(bs) <= buffLen {
		list = append(list, bs)
		return list
	}
	for len(bs) > buffLen {
		list = append(list, bs[0:buffLen])
		bs = bs[buffLen:]
	}
	if bs != nil && len(bs) > 0 {
		list = append(list, bs)
	}
	return list
}

func byteToHexadecimal(b byte) string {
	return fmt.Sprintf("0x%02X", b)
}
