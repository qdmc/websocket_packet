package frame

import (
	"errors"
	"io"
	"net"
)

/*
CodecInterface    解码器通用接口
  - ReadOnce(r io.Reader)         阻塞模式下读取一个 Frame
  - ReadBuffer(buf []byte)        读取缓存冲区的字节流,返回:帖列表,剩余字节,error
  - WriteFrame(*Frame, net.Conn)  一个frame帧写入Connection,
  - WriteBytes([]byte, net.Conn)  写入字节流
*/
type CodecInterface interface {
	ReadOnce(r io.Reader) (*Frame, error)
	ReadBuffer(buf []byte) ([]*Frame, []byte, error)
	WriteFrame(*Frame, net.Conn) (int, error)
	WriteBytes([]byte, net.Conn) (int, error)
}

func NewCodec() CodecInterface {
	return new(defaultCodec)
}

type defaultCodec struct{}

func (defaultCodec) ReadOnce(r io.Reader) (*Frame, error) {
	f := new(Frame)
	_, err := f.Read(r)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (defaultCodec) ReadBuffer(bs []byte) ([]*Frame, []byte, error) {
	return ReadBytes(bs)
}
func (defaultCodec) WriteFrame(f *Frame, c net.Conn) (writeLen int, err error) {
	if f == nil {
		return writeLen, errors.New("frame is empty")
	}
	if c == nil {
		return writeLen, errors.New("connection is empty")
	}
	bs, err := f.ToBytes()
	if err != nil {
		return writeLen, err
	}
	writeLen, err = c.Write(bs)
	return
}
func (defaultCodec) WriteBytes(bs []byte, c net.Conn) (writeLen int, err error) {
	if c == nil {
		return writeLen, errors.New("connection is empty")
	}
	if bs == nil && len(bs) == 0 {
		return writeLen, errors.New("frames bytes  is empty")
	}
	writeLen, err = c.Write(bs)
	return
}
