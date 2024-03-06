package frame

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/qdmc/websocket_packet/uity"
	"io"
	"unicode/utf8"
)

var PayloadLengthError = errors.New(fmt.Sprintf("PayloadLength >  %d", PayloadMaxLength))

const PayloadMaxLength = 0x7FFFFFFFFFFFFFFF

type Frame struct {
	Fin           byte   // 1 bit ,1表示最后一个消息帧
	Rsv1          byte   // 1 bit ,如果没有协商则必须为 0
	Rsv2          byte   // 1 bit ,如果没有协商则必须为 0
	Rsv3          byte   // 1 bit ,如果没有协商则必须为 0
	Opcode        byte   // 4 bit ,到这为第一字节;1:文本帧;2:二进制帧;3-7:保留给将来的非控制帧;8:连接关闭;9:ping 帧;A:pong 帧;B-F:保留给将来的控制帧
	Masked        byte   // 1 bit ,0:不添加掩码;1:添加掩码;定义“有效负载数据”是否添加掩码。如果设置为 1，那么掩码的键值存在于 Masking-Key 中,这个一般用于解码“有效负载数据”。所有的从客户端发送到服务端的帧都需要设置这个 bit 位为 1
	PayloadLength uint64 // 7 或者 7+16 或者 7+64 bit: 以字节为单位的“有效负载数据”长度，如果值为 0-125，那么就表示负载数据的长度。如果是 126，那么接下来的 2 个 bytes 解释为 16bit 的无符号整形作为负载数据的长度。如果是 127，那么接下来的 8 个 bytes 解释为一个 64bit 的无符号整形（最高位的 bit 必须为 0）作为负载数据的长度
	MaskingKey    uint32 // 32 bit
	PayloadData   []byte
}

func (f *Frame) ToString() string {
	t := `
Fin:%s,
Rsv1:%s,
Rsv2:%s,
Rsv3:%s,
Opcode:%s,
Masked:%s,
PayloadLength:%d,
MaskingKey:%d,
PayloadData: %s,
`
	return fmt.Sprintf(t,
		uity.ByteToHexadecimal(f.Fin),
		uity.ByteToHexadecimal(f.Rsv1),
		uity.ByteToHexadecimal(f.Rsv2),
		uity.ByteToHexadecimal(f.Rsv3),
		uity.ByteToHexadecimal(f.Opcode),
		uity.ByteToHexadecimal(f.Masked),
		f.PayloadLength,
		f.MaskingKey,
		string(f.PayloadData),
	)
}
func (f *Frame) SetPayload(data []byte) {
	if data != nil && len(data) > 0 {
		f.PayloadData = data
	}
}
func (f *Frame) SetFin(b byte) {
	if b == 0x00 || b == 0x01 {
		f.Fin = b
	}
}
func (f *Frame) SetOpcode(b byte) {
	if b == 0x00 || b == 0x01 || b == 0x08 || b == 0x09 || b == 0x0A {
		f.Opcode = b
	}

}
func (f *Frame) ToBytes() ([]byte, error) {
	var payloadLength = uint64(0)
	var frameBytes, lengthBytes, data, enData []byte
	firstByte := f.Fin<<7 + f.Rsv1<<6 + f.Rsv2<<5 + f.Rsv3<<4 + f.Opcode<<4>>4
	frameBytes = []byte{firstByte}
	if f.PayloadData != nil && len(f.PayloadData) > 0 {
		payloadLength = uint64(len(f.PayloadData))
		data = f.PayloadData
	} else {
		data = []byte{}
	}
	if payloadLength <= 125 {
		lengthBytes = []byte{f.Masked<<7 + uint8(payloadLength)}
	} else if payloadLength > 125 && payloadLength <= 65535 {
		lengthBytes = []byte{f.Masked<<7 + 0x7E}
		lengthBytes = append(lengthBytes, uity.EnCodeUint16(uint16(payloadLength))...)
	} else if payloadLength > 65535 && payloadLength <= PayloadMaxLength {
		lengthBytes = []byte{f.Masked<<7 + 0x7F}
		lengthBytes = append(lengthBytes, uity.EnCodeUin64(payloadLength)...)
	} else {
		return nil, PayloadLengthError
	}
	frameBytes = append(frameBytes, lengthBytes...)
	if f.Masked == 0x01 {
		frameBytes = append(frameBytes, uity.EnCodeUint32(f.MaskingKey)...)
		enData = MasKingData(data, f.MaskingKey)
	} else {
		enData = data
	}
	frameBytes = append(frameBytes, enData...)
	return frameBytes, nil
}
func (f *Frame) SetMaskingKey(key uint32) {
	f.Masked = 1
	f.MaskingKey = key
}

// Read         读取一个webSocket帧,返回读取的长度
func (f *Frame) Read(r io.Reader) (int, error) {
	var n int
	firstByte, err := uity.ReadByte(r)
	if err != nil {
		return n, errors.New(fmt.Sprintf("read first Byte error: %s", err.Error()))
	}
	n += 1
	f.Fin = firstByte >> 7
	f.Rsv1 = firstByte << 1 >> 7
	f.Rsv2 = firstByte << 2 >> 7
	f.Rsv3 = firstByte << 3 >> 7
	f.Opcode = firstByte << 4 >> 4
	secondByte, err := uity.ReadByte(r)
	if err != nil {
		return n, errors.New(fmt.Sprintf("read second Byte error: %s", err.Error()))
	}
	n += 1
	f.Masked = secondByte >> 7
	length := secondByte << 1 >> 1
	//fmt.Println("length: ", length)
	if length <= 0x7D {
		f.PayloadLength = uint64(length)
	} else if length == 0x7E {
		length32, u16Err := uity.ReadUint16(r)
		if u16Err != nil {
			return n, errors.New(fmt.Sprintf("read uint16 length error: %s", u16Err.Error()))
		}
		n += 2
		f.PayloadLength = uint64(length32)
	} else if length == 0x7F {
		length64, u64Err := uity.ReadUint64(r)
		if u64Err != nil {
			return n, errors.New(fmt.Sprintf("read uint64 length error: %s", u64Err.Error()))
		}
		if length64 > PayloadMaxLength {
			return n, errors.New(fmt.Sprintf("read uint64 length >  %d", PayloadMaxLength))
		}
		n += 8
		f.PayloadLength = length64
	} else {
		return n, errors.New("frame PayloadLength is error")
	}
	if f.Masked == 0x01 {
		key, keyErr := uity.ReadUint32(r)
		if keyErr != nil {
			return n, errors.New(fmt.Sprintf("read MaskingKey error: %s", keyErr.Error()))
		}
		n += 4
		f.MaskingKey = key
	}
	if f.PayloadLength > 0 {
		dataBytes, payloadErr := uity.ReadBytes(r, int(f.PayloadLength))
		if payloadErr != nil {
			return n, errors.New(fmt.Sprintf("read Payload error: %s", payloadErr.Error()))
		}
		n += int(f.PayloadLength)
		if f.Masked == 0x01 {
			f.PayloadData = MasKingData(dataBytes, f.MaskingKey)
		} else {
			f.PayloadData = dataBytes
		}
	}
	return n, nil
}

func ReadBytes(framesBytes []byte) ([]*Frame, []byte, error) {
	if framesBytes == nil && len(framesBytes) < 1 {
		return nil, nil, errors.New("frames bytes is empty")
	}
	var list []*Frame
	var err error
	var frameLength, maskedLen int
	var payloadLen, masked byte
	for {
		if len(framesBytes) < 2 {
			return list, framesBytes, nil
		}
		masked = framesBytes[1] >> 7
		if masked == 0x01 {
			maskedLen = 4
		} else {
			maskedLen = 0
		}
		payloadLen = framesBytes[1] << 1 >> 1
		if payloadLen <= 0x7D {
			frameLength = 2 + int(payloadLen) + maskedLen
		} else if payloadLen == 0x7E {
			if len(framesBytes) < 4 {
				return list, framesBytes, nil
			}
			frameLength = 2 + int(binary.BigEndian.Uint64([]byte{framesBytes[2], framesBytes[3]})) + maskedLen
		} else if payloadLen == 0x7F {
			if len(framesBytes) < 10 {
				return list, framesBytes, nil
			}
			frameLength = 2 + int(binary.BigEndian.Uint64(framesBytes[2:10])) + maskedLen
		}
		if len(framesBytes) < frameLength {
			return list, framesBytes, nil
		}
		frameBS := framesBytes[0:frameLength]
		f := new(Frame)
		_, err = f.Read(bytes.NewReader(frameBS))
		if err != nil {
			return list, framesBytes, err
		}
		list = append(list, f)
		framesBytes = framesBytes[frameLength:]
		if len(framesBytes) < 1 {
			return list, framesBytes, nil
		} else {
			continue
		}
	}
}

func MasKingData(payloadData []byte, key uint32) []byte {
	if payloadData == nil || len(payloadData) < 1 {
		return payloadData
	}
	keyBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(keyBytes, key)
	var bs []byte
	dataLen := len(payloadData)
	for i := 0; i < dataLen; i++ {
		j := i % 4
		bs = append(bs, payloadData[i]^keyBytes[j])
	}
	return bs
}

/*
Client: FIN=1, opcode=0x1, msg="hello" // 第一个消息在单个帧中发送
Server: (process complete message immediately) Hi.
Client: FIN=0, opcode=0x1, msg="and a"  // 第二个消息跨三个帧发送
Server: (listening, newmessage containing text started)
Client: FIN=0, opcode=0x0, msg="happy new"
Server: (listening, payload concatenated to previous message)
Client: FIN=1, opcode=0x0, msg="year!"
Server: (process complete message) Happy newyear to you too!
*/

func PongFrameBytes(bs []byte, keys ...uint32) []byte {
	if bs != nil && len(bs) > 125 {
		bs = bs[0:125]
	}
	var key uint32
	var isKey bool
	var framesBytes []byte
	if keys != nil && len(keys) == 1 {
		key = keys[0]
		isKey = true
	}
	frame := new(Frame)
	frame.SetFin(0x02)
	frame.SetOpcode(0x0A)
	if isKey {
		frame.SetMaskingKey(key)
	}
	frame.SetPayload(bs)
	framesBytes, _ = frame.ToBytes()
	return framesBytes
}

func PingFrameBytes(bs []byte, keys ...uint32) []byte {
	if bs != nil && len(bs) > 125 {
		bs = bs[0:125]
	}
	var key uint32
	var isKey bool
	var framesBytes []byte
	if keys != nil && len(keys) == 1 {
		key = keys[0]
		isKey = true
	}
	frame := new(Frame)
	frame.SetFin(0x02)
	frame.SetOpcode(0x09)
	if isKey {
		frame.SetMaskingKey(key)
	}
	frame.SetPayload(bs)
	framesBytes, _ = frame.ToBytes()
	return framesBytes
}

func CloseFrameBytes(bs []byte, keys ...uint32) []byte {
	if bs != nil && len(bs) > 125 {
		bs = bs[0:125]
	}
	var key uint32
	var isKey bool
	var framesBytes []byte
	if keys != nil && len(keys) == 1 {
		key = keys[0]
		isKey = true
	}
	frame := new(Frame)
	frame.SetFin(0x02)
	frame.SetOpcode(0x08)
	if isKey {
		frame.SetMaskingKey(key)
	}
	frame.SetPayload(bs)
	framesBytes, _ = frame.ToBytes()
	return framesBytes
}

func BinaryFrameBytes(bs []byte, keys ...uint32) []byte {
	var key uint32
	var isKey bool
	var framesBytes []byte
	if keys != nil && len(keys) == 1 {
		key = keys[0]
		isKey = true
	}
	if len(bs) > PayloadMaxLength {
		bsArr := uity.PacketBytesDivision(bs, PayloadMaxLength)
		for index, frameData := range bsArr {
			frame := new(Frame)
			if index == 0 {
				frame.SetOpcode(0x02)
			} else {
				frame.SetOpcode(0x00)
			}
			if index == len(bsArr)-1 {
				frame.SetFin(0x02)
			} else {
				frame.SetFin(0x00)
			}

			if isKey {
				frame.SetMaskingKey(key)
			}
			frame.SetPayload(frameData)
			framesBs, _ := frame.ToBytes()
			framesBytes = append(framesBytes, framesBs...)
		}
	} else {
		frame := new(Frame)
		frame.SetFin(0x01)
		frame.SetOpcode(0x02)
		if isKey {
			frame.SetMaskingKey(key)
		}
		frame.SetPayload(bs)
		framesBytes, _ = frame.ToBytes()
	}
	return framesBytes
}

func TextFrameBytes(bs []byte, keys ...uint32) ([]byte, error) {
	if bs != nil && len(bs) > 0 {
		if !utf8.Valid(bs) {
			return nil, errors.New("text bytes is not utf8")
		}
	}
	var key uint32
	var isKey bool
	var framesBytes []byte
	if keys != nil && len(keys) == 1 {
		key = keys[0]
		isKey = true
	}
	if len(bs) > PayloadMaxLength {
		bsArr := uity.PacketBytesDivision(bs, PayloadMaxLength)
		for index, frameData := range bsArr {
			frame := new(Frame)
			if index == 0 {
				frame.SetOpcode(0x01)
			} else {
				frame.SetOpcode(0x00)
			}
			if index == len(bsArr)-1 {
				frame.SetFin(0x01)
			} else {
				frame.SetFin(0x00)
			}

			if isKey {
				frame.SetMaskingKey(key)
			}
			frame.SetPayload(frameData)
			framesBs, _ := frame.ToBytes()
			framesBytes = append(framesBytes, framesBs...)
		}
	} else {
		frame := new(Frame)
		frame.SetFin(0x01)
		frame.SetOpcode(0x01)
		if isKey {
			frame.SetMaskingKey(key)
		}
		frame.SetPayload(bs)
		framesBytes, _ = frame.ToBytes()
	}
	return framesBytes, nil
}
