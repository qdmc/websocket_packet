package uity

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// EnCodeUint16 打码uint16转成两字节
func EnCodeUint16(n uint16) []byte {
	bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(bytes, n)
	return bytes
}

// EnCodeUint32 解码uint32转成四字节
func EnCodeUint32(n uint32) []byte {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, n)
	return bytes
}

// EnCodeUin64 解码uint64转成8字节
func EnCodeUin64(n uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, n)
	return bytes
}

// ReadUint16 读取uint16
func ReadUint16(r io.Reader) (uint16, error) {
	bs := make([]byte, 2)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(bs), nil
}

// ReadUint32   读取uint32
func ReadUint32(r io.Reader) (uint32, error) {
	bs := make([]byte, 4)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(bs), nil
}

// ReadUint64  读取uint64
func ReadUint64(r io.Reader) (uint64, error) {
	bs := make([]byte, 8)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(bs), nil
}

func ReadBytes(r io.Reader, lengths ...int) ([]byte, error) {
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

func ReadByte(r io.Reader) (byte, error) {
	bs := make([]byte, 1)
	_, err := io.ReadFull(r, bs)
	if err != nil {
		return 0, err
	}
	return bs[0], nil
}

// PacketBytesDivision Bytes分割
// bs                  []byte
// buffLen             分割长度
func PacketBytesDivision(bs []byte, buffLen int) [][]byte {
	var list [][]byte
	if len(bs) < 1 {
		return list
	}
	if buffLen < 1 || len(bs) < buffLen {
		list = append(list, bs)
		return list
	}
	start := 0
	for start < len(bs) {
		end := start + buffLen
		if end >= (len(bs) - 1) {
			list = append(list, bs[start:])
		} else {
			list = append(list, bs[start:end])

		}
		start += buffLen
	}
	return list
}

func ByteToHexadecimal(b byte) string {
	return fmt.Sprintf("0x%02X", b)
}

func BytesToHexadecimal(bs []byte) string {
	if bs == nil && len(bs) < 1 {
		return ""
	}
	var strArr []string
	for _, b := range bs {
		strArr = append(strArr, ByteToHexadecimal(b))
	}
	return strings.Join(strArr, ",")
}

func ComputeAcceptKey(key string) string {
	h := sha1.New() //#nosec G401 -- (CWE-326) https://datatracker.ietf.org/doc/html/rfc6455#page-54
	h.Write([]byte(key))
	h.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
