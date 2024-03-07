package session

import (
	"errors"
	"github.com/qdmc/websocket_packet/frame"
	"github.com/qdmc/websocket_packet/uity/ids"
	"golang.org/x/net/websocket"
	"sync"
	"sync/atomic"
	"time"
)

type ConnectedCallBack func(id int64)
type DisConnectCallBack func(id int64, status Status)
type FrameCallBack func(id int64, t byte, payload []byte)

// ConnectionDatabase     链接数据
type ConnectionDatabase struct {
	Id            int64
	ConnectedNano int64
	CloseNano     int64
	SendLength    uint64
	WriteLength   uint64
	Status        Status
}

func (c ConnectionDatabase) GetOnlineNone() int64 {
	if c.CloseNano > 0 {
		return time.Now().UnixNano() - c.ConnectedNano
	} else {
		return c.CloseNano - c.ConnectedNano
	}
}

/*
WebsocketSessionInterface                            session通用接口
  - IsServer() bool                                  是否是服务端session
  - GetId() int64                                    返回sessionId,客户端sessionId都为0
  - GetStatus() Status                               返回session状态
  - DoConnect(autoPingTicker ...int64)               执行conn的读取,autoPingTicker:自动发送pingFrame的ticker,>=10为有效值,默认是25秒
  - Write(frameType byte, bs []byte, keys ...uint32) 写入消息:frameType(消息类型,1,2,9,10 为有效值)
  - SetDisConnectCallBack(DisConnectCallBack)        设置链接断开的回调
  - SetFrameCallBack(FrameCallBack)                  设置报文帧的回调
  - DisConnect()                                     主动关闭链接
*/
type WebsocketSessionInterface interface {
	GetId() int64
	IsServer() bool
	GetStatus() ConnectionDatabase
	DoConnect(autoPingTicker ...int64)
	Write(frameType byte, bs []byte, keys ...uint32) (int, error)
	SetDisConnectCallBack(DisConnectCallBack)
	SetFrameCallBack(FrameCallBack)
	DisConnect()
}

func NewSession(c *websocket.Conn, isServer ...bool) WebsocketSessionInterface {
	id := int64(0)
	isServ := false
	if isServer != nil && len(isServer) == 1 && isServer[0] {
		id = ids.GetId()
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
	conn              *websocket.Conn
	status            Status
	connectedCb       ConnectedCallBack
	disConnectCb      DisConnectCallBack
	frameCb           FrameCallBack
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
	s.Write(9, nil)
}
func (s *websocketSession) DoConnect(pingTickers ...int64) {
	if s.status != Connected {
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
			readLen, f, err := frame.ReadOnce(s.conn)
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
	if f.Opcode == 8 {
		s.close(CloseNormalClosure)
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
	switch frameType {
	case 1:
		frameBytes, err = frame.TextFrameBytes(bs, keys...)
	case 2:
		frameBytes = frame.BinaryFrameBytes(bs, keys...)
	case 9:
		frameBytes = frame.PingFrameBytes(bs, keys...)
	case 10:
		frameBytes = frame.PongFrameBytes(bs, keys...)
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
		atomic.AddUint64(s.writeLen, uint64(writeLen))
		return writeLen, nil
	} else {
		return writeLen, errors.New("not connected")
	}
}

func (s *websocketSession) SetDisConnectCallBack(back DisConnectCallBack) {
	if s.status != Connected {
		return
	}
	s.disConnectCb = back
}

func (s *websocketSession) SetFrameCallBack(back FrameCallBack) {
	if s.status != Connected {
		return
	}
	s.frameCb = back
}

func (s *websocketSession) DisConnect() {
	s.mu.Lock()
	s.mu.Unlock()
	if s.status == Connected {
		s.conn.Write(frame.CloseFrameBytes(nil))
		close(s.stopChan)
	}

}
func (s *websocketSession) close(status Status) {
	s.mu.Lock()
	s.mu.Unlock()
	if s.status == Connected {
		if status == CloseNormalClosure || status == CloseReadConnFailed || status == CloseWriteConnFailed {
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
