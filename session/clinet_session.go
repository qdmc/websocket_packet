package session

import (
	"github.com/qdmc/websocket_packet/frame"
	"golang.org/x/net/websocket"
	"sync"
)

func NewClientSession(c *websocket.Conn) WebsocketSessionInterface {
	return &clientSession{
		mu:        sync.Mutex{},
		isNotAuto: false,
		conn:      c,
		codec:     frame.NewCodec(),
		status:    Connected,
	}
}

type clientSession struct {
	mu        sync.Mutex
	isNotAuto bool
	conn      *websocket.Conn
	codec     frame.CodecInterface
	status    Status
}

func (s *clientSession) GetId() int64 {
	return 0
}

func (s *clientSession) IsServer() bool {
	return false
}

func (s *clientSession) GetStatus() Status {
	return s.status
}

func (s *clientSession) DoConnect() {
	//TODO implement me
	panic("implement me")
}

func (s *clientSession) Write(frameType byte, bs []byte) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (s *clientSession) SetConnectedCallBack(back ConnectedCallBack) {
	//TODO implement me
	panic("implement me")
}

func (s *clientSession) SetDisConnectCallBack(back DisConnectCallBack) {
	//TODO implement me
	panic("implement me")
}

func (s *clientSession) SetFrameCallBack(back FrameCallBack) {
	//TODO implement me
	panic("implement me")
}

func (s *clientSession) CloseAutoRes() {
	s.mu.Lock()
	s.mu.Unlock()
	s.isNotAuto = true
}
func (s *clientSession) DisConnect() {

}
func (s *clientSession) close(status Status) {

}
