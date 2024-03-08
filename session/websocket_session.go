/*
Package session  websocket链接
  - websocket.Conn 链接管理:帧的读取与写入,状态及回调;并自动回复pong帧,及定时(默认25)发送ping帧
  - Status         状态定义
  - Callbacks      回调定义:  ConnectedCallBackHandle DisConnectCallBackHandle FrameCallBackHandle
*/
package session

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/qdmc/websocket_packet/frame"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectedCallBackHandle     建立链接后的回调
type ConnectedCallBackHandle func(id int64, header http.Header)

// DisConnectCallBackHandle    断开链接后的回调
type DisConnectCallBackHandle func(id int64, status Status)

// FrameCallBackHandle         帧读取后的回调
type FrameCallBackHandle func(id int64, t byte, payload []byte)

// ConnectionDatabase     链接数据
type ConnectionDatabase struct {
	Id            int64  // sessionId
	ConnectedNano int64  // 链接开始时间
	CloseNano     int64  // 链接断开时间
	SendLength    uint64 // 发送的数据长度
	WriteLength   uint64 // 接收的数据长度
	Status        Status // 状态
}

/*
WebsocketSessionInterface                            session通用接口
  - IsServer() bool                                  是否是服务端session
  - GetId() int64                                    返回sessionId:服务端的sessionId全局唯一,客户端sessionId为0;
  - GetStatus() Status                               返回session状态
  - DoConnect(autoPingTicker ...int64)               执行conn的读取,autoPingTicker:自动发送pingFrame的ticker,>=10为有效值,默认是25秒
  - Write(frameType byte, bs []byte, keys ...uint32) 写入消息:frameType(消息类型,1,2,9,10 为有效值)
  - SetDisConnectCallBack(DisConnectCallBackHandle)        设置链接断开的回调
  - SetFrameCallBack(FrameCallBackHandle)                  设置报文帧的回调
  - DisConnect()                                     主动关闭链接
*/
type WebsocketSessionInterface interface {
	GetId() int64
	GetConn() net.Conn
	IsServer() bool
	GetStatus() ConnectionDatabase
	DoConnect(autoPingTicker ...int64)
	Write(frameType byte, bs []byte, keys ...uint32) (int, error)
	SetDisConnectCallBack(DisConnectCallBackHandle)
	SetFrameCallBack(FrameCallBackHandle)
	DisConnect(isTimeOut ...bool)
}

/*
NewSession                     生成一个 WebsocketSessionInterface
  - 这里的net.Conn默认是*net.TCPCon,不能兼容golang.org/x/net/websocket中的Conn
  - isServer 来标识客户端与服务端,但session并没有处理握手
  - 默认的是客户端session,服务端session会分配一个全局唯一的id
*/
func NewSession(c net.Conn, isServer ...bool) WebsocketSessionInterface {
	id := int64(0)
	isServ := false
	if isServer != nil && len(isServer) == 1 && isServer[0] {
		id = getSessionId()
		isServ = true
	}
	var rLen, wLen uint64
	return &websocketSession{
		id:         id,
		isServer:   isServ,
		mu:         sync.Mutex{},
		conn:       c,
		status:     Connected,
		stopChan:   make(chan struct{}, 1),
		pingTicker: nil,
		startNano:  time.Now().UnixNano(),
		readLen:    &rLen,
		writeLen:   &wLen,
	}
}

type websocketSession struct {
	id                int64
	isServer          bool
	mu                sync.Mutex
	conn              net.Conn
	status            Status
	connectedCb       ConnectedCallBackHandle
	disConnectCb      DisConnectCallBackHandle
	frameCb           FrameCallBackHandle
	stopChan          chan struct{}
	pingTicker        *time.Ticker
	continuationFrame *frame.Frame
	startNano         int64
	closeNano         int64
	readLen           *uint64
	writeLen          *uint64
}

func (s *websocketSession) GetId() int64 {
	return s.id
}
func (s *websocketSession) GetConn() net.Conn {
	return s.conn
}
func (s *websocketSession) IsServer() bool {
	return s.isServer
}

func (s *websocketSession) GetStatus() ConnectionDatabase {
	return ConnectionDatabase{
		Id:            s.id,
		ConnectedNano: s.startNano,
		CloseNano:     s.closeNano,
		SendLength:    atomic.LoadUint64(s.readLen),
		WriteLength:   atomic.LoadUint64(s.writeLen),
		Status:        s.status,
	}
}

func (s *websocketSession) doPing() {
	fmt.Println("**** doPing")
	s.Write(9, []byte("Hello"))
}
func (s *websocketSession) DoConnect(pingTickers ...int64) {
	if s.status != Connected {
		return
	}
	err := s.conn.SetDeadline(time.Time{})
	if err != nil {
		return
	}
	var status Status = CloseNormalClosure
	defer func() {
		s.conn.Close()
		s.close(status)
	}()
	d := 25 * time.Second
	if pingTickers != nil && len(pingTickers) == 1 && pingTickers[0] >= 10 {
		d = time.Duration(pingTickers[0]) * time.Second
	}
	s.pingTicker = time.NewTicker(d)
	for {
		select {
		case <-s.stopChan:
			return
		case <-s.pingTicker.C:
			go s.doPing()
			continue
		default:
			readLen, f, err := frame.ReadOnceFrame(s.conn)
			if err != nil {
				status = CloseReadConnFailed
				return
			} else {
				atomic.AddUint64(s.readLen, uint64(readLen))
				// 处理分包合并,Fin为1时,表示最后一个分包
				if f.Fin == 0 {
					if s.continuationFrame == nil {
						s.continuationFrame = f
					} else {
						s.continuationFrame.PayloadData = append(s.continuationFrame.PayloadData, f.PayloadData...)
					}
				} else {
					// 这里合并分包,并弹出
					if s.continuationFrame != nil {
						composeFrame := new(frame.Frame)
						composeFrame.SetOpcode(f.Opcode)
						composeFrame.SetPayload(append(s.continuationFrame.PayloadData, f.PayloadData...))
						s.continuationFrame = nil
						go s.doFrameCallBack(composeFrame)
					} else {
						go s.doFrameCallBack(f)
					}

				}
				continue
			}
		}
	}

}
func (s *websocketSession) doFrameCallBack(f *frame.Frame) {
	if f == nil {
		return
	}
	fmt.Println("*** doFrameCallBack: ", f.Opcode)
	if f.Opcode == 8 {
		s.close(CloseNormalClosure)
	} else if f.Opcode == 9 {
		s.Write(10, f.PayloadData)
	} else {
		if s.frameCb != nil {
			go s.frameCb(s.GetId(), f.Opcode, f.PayloadData)
		}
	}
}
func (s *websocketSession) Write(frameType byte, bs []byte, keys ...uint32) (int, error) {
	var frameBytes []byte
	var err error
	var writeLen int
	if s.isServer == true {
		keys = nil
	}
	switch frameType {
	case 1:
		frameBytes, err = frame.AutoTextFramesBytes(bs, keys...)
	case 2:
		frameBytes, err = frame.AutoBinaryFramesBytes(bs, keys...)
	case 9:
		frameBytes, err = frame.NewPingFrame(bs, keys...).ToBytes()
	case 10:
		frameBytes, err = frame.NewPongFrame(bs, keys...).ToBytes()
	default:
		err = errors.New("frameType must be in 1,2,9,10")
	}
	if err != nil {
		return writeLen, err
	}
	if s.status == Connected {
		writeLen, err = s.conn.Write(frameBytes)
		if err != nil {
			s.close(CloseWriteConnFailed)
			return 0, err
		}
		if frameType == 9 {
			fmt.Println("hex: ", hex.EncodeToString(frameBytes))
		}
		atomic.AddUint64(s.writeLen, uint64(writeLen))
		return writeLen, nil
	} else {
		return writeLen, errors.New("not connected")
	}
}

func (s *websocketSession) SetDisConnectCallBack(back DisConnectCallBackHandle) {
	if s.status != Connected {
		return
	}
	s.disConnectCb = back
}

func (s *websocketSession) SetFrameCallBack(back FrameCallBackHandle) {
	if s.status != Connected {
		return
	}
	s.frameCb = back
}

func (s *websocketSession) DisConnect(isTimeOut ...bool) {
	if s.status == Connected {
		bs, _ := frame.NewCloseFrame(nil).ToBytes()
		s.conn.Write(bs)
		if isTimeOut != nil && len(isTimeOut) == 1 && isTimeOut[0] {
			s.close(CloseHartTimeOut)
		} else {
			s.close(CloseNormalClosure)
		}

	}

}
func (s *websocketSession) close(status Status) {
	s.mu.Lock()
	s.mu.Unlock()
	if s.status == Connected {
		if status == CloseNormalClosure || status == CloseReadConnFailed || status == CloseWriteConnFailed || CloseHartTimeOut == status {
			close(s.stopChan)
		}
		s.status = status
		if s.disConnectCb != nil {
			go s.disConnectCb(s.GetId(), s.status)
		}
		s.closeNano = time.Now().UnixNano()
	}
	if s.pingTicker != nil {
		s.pingTicker.Stop()
	}
}
