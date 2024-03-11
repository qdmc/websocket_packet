package frame

import (
	"errors"
	"fmt"
	"io"
)

/*
CodecInterface    解码器通用接口
  - ReadOnceFrame(r io.Reader)         阻塞模式下读取一个 Frame
  - ReadBuffer(buf []byte)             读取缓存冲区的字节流,返回:帖列表,剩余字节,error
  - WriteFrame(*Frame, io.Writer)      一个frame帧写入Connection,
  - WriteBytes([]byte, io.Writer)      写入字节流
*/
type CodecInterface interface {
	ReadOnce(r io.Reader) (*Frame, error)            // 阻塞模式下读取一个 Frame
	ReadBuffer(buf []byte) ([]*Frame, []byte, error) // 读取缓存冲区的字节流
	WriteFrame(*Frame, io.Writer) (int, error)       // 一个frame帧写入
	WriteBytes([]byte, io.Writer) (int, error)       // 写入字节流
}

func NewCodec() CodecInterface {
	return new(defaultCodec)
}

type defaultCodec struct{}

func (defaultCodec) ReadOnce(r io.Reader) (*Frame, error) {
	f := new(Frame)
	_, readStatus := f.read(r)

	return f, StatusToError(readStatus)
}

func (defaultCodec) ReadBuffer(bs []byte) ([]*Frame, []byte, error) {
	list, lastBS, readStatus := ReadStreamBufferBytes(bs)
	return list, lastBS, StatusToError(readStatus)
}
func (defaultCodec) WriteFrame(f *Frame, c io.Writer) (writeLen int, err error) {
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
func (defaultCodec) WriteBytes(bs []byte, c io.Writer) (writeLen int, err error) {
	if c == nil {
		return writeLen, errors.New("connection is empty")
	}
	if bs == nil && len(bs) == 0 {
		return writeLen, errors.New("frames bytes  is empty")
	}
	writeLen, err = c.Write(bs)
	return
}

// CloseStatus   关闭状态码,一般是关闭帧负载的前两位
type CloseStatus uint16

const (
	CloseNormalClosure           CloseStatus = 1000 // 表示端点“离开”，例如服务器关闭或浏览器导航到其他页面
	CloseGoingAway               CloseStatus = 1001 // 表示端点“离开”，例如服务器关闭或浏览器导航到其他页面
	CloseProtocolError           CloseStatus = 1002 // 表示端点因为协议错误而终止连接
	CloseUnsupportedData         CloseStatus = 1003 // 表示端点由于它收到了不能接收的数据类型
	CloseNoStatusReceived        CloseStatus = 1005 // 1005 是一个保留值，且不能由端点在关闭控制帧中设置此状态码。它被指定用在期待一个用于表示没有状态码是实际存在的状态码的应用中
	CloseAbnormalClosure         CloseStatus = 1006 // 是一个保留值，且不能由端点在关闭控制帧中设置此状态码。它被指定用在期待一个用于表示连接异常关闭的状态码的应用中
	CloseInvalidFramePayloadData CloseStatus = 1007 // 表示端点因为消息中接收到的数据是不符合消息类型而终止连接（比如，文本消息中存在非UTF-8 [RFC3629]数据）
	ClosePolicyViolation         CloseStatus = 1008 // 表示端点因为接收到的消息违反其策略而终止连接。这是一个当没有其它合适状态码（例如1003或1009）或如果需要隐藏策略的具体细节时能被返回的通用状态码
	CloseMessageTooBig           CloseStatus = 1009 // 表示端点因接收到的消息对它的处理来说太大而终止连接
	CloseMandatoryExtension      CloseStatus = 1010 // 表示端点（客户端）因为它期望服务器协商一个或多个扩展，但服务器没有在WebSocket握手响应消息中返回它们而终止连接。所需要的扩展列表应该出现在关闭帧的/reason/部分。注意，这个状态码不能被服务器端使用，因为它可以使WebSocket握手失败
	CloseInternalServerErr       CloseStatus = 1011 // 表示服务器端因为遇到了一个不期望的情况使它无法满足请求而终止连接
	CloseServiceRestart          CloseStatus = 1012 // 服务端重新启动
	CloseTryAgainLater           CloseStatus = 1013 // 请稍后重试，表示服务器暂时无法处理连接，请求客户端稍后再试
	CloseTLSHandshake            CloseStatus = 1015 // 是一个保留值，且不能由端点在关闭帧中被设置为状态码。它被指定用在期待一个用于表示连接由于执行TLS握手失败而关闭的状态码的应用中（比如，服务器证书不能验证）

	SessionClientCreate    CloseStatus = 3000 //  新增自定义状态:客户端建立
	SessionClientReconnect CloseStatus = 3001 //  新增自定义状态:客户端重新链接
	SessionConnected       CloseStatus = 3002 //  新增自定义状态:正常连接。
	SessionHartTimeOut     CloseStatus = 3004 //  新增自定义状态:心跳超时
	SessionWriteConnFailed CloseStatus = 3006 //  新增自定义状态:写入失败
)

// StatusToError    状态转换成 error
func StatusToError(s CloseStatus) error {
	switch s {
	case SessionClientCreate:
		return nil
	case SessionClientReconnect:
		return nil
	case SessionConnected:
		return nil
	case CloseNormalClosure:
		return nil
	case 1001:
		return errors.New("1001: Going Away")
	case 1002:
		return errors.New("1002: Protocol error")
	case 1003:
		return errors.New("1003: Unsupported Data")
	case 1005:
		return errors.New("1005: No Status Rcvd ")
	case 1006:
		return errors.New("1006: Abnormal Closure")
	case 1007:
		return errors.New("1007: Invalid frame payload data")
	case 1008:
		return errors.New("1008: Policy Violation")
	case 1009:
		return errors.New("1009: Message Too Big")
	case 1010:
		return errors.New("1010:  Mandatory Ext.")
	case 1011:
		return errors.New("1011: Internal Server Error")
	case 1012:
		return errors.New("1012: Service Restart")
	case 1013:
		return errors.New("1013: Try Again Later")
	case 1015:
		return errors.New("1015: TLS handshake")
	case SessionHartTimeOut:
		return errors.New(fmt.Sprintf("%d: Heartbeat Timeout,close connection", SessionHartTimeOut))
	case SessionWriteConnFailed:
		return errors.New(fmt.Sprintf("%d: Write to connection failed,close connection", SessionWriteConnFailed))
	}
	return errors.New("unknown errors")
}
