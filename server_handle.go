package websocket_packet

import (
	"errors"
	"fmt"
	"github.com/qdmc/websocket_packet/session"
	"github.com/qdmc/websocket_packet/uity"
	"golang.org/x/net/websocket"
	"net/http"
	"sort"
	"sync"
	"time"
)

var manager *sessionManager
var managerOnce sync.Once

type ConnectedCallBack = session.ConnectedCallBack
type DisConnectCallBack = session.DisConnectCallBack
type MessageCallBack = session.FrameCallBack
type Callbacks struct {
	ConnectedCallBack
	DisConnectCallBack
	MessageCallBack
}

/*
server实现的业务
 - 校验websocket握手(Handshake)
 - net.http.Handler:实现ServeHTTP(w http.ResponseWriter, req *http.Request)
 - 管理session:查询,断开
 - 消息的接收与发送
*/

type ServerHandlerInterface interface {
	SetCallbacks(*Callbacks)
	Len() int
	GetSessionOnce(id int64) (Session, error)
	GetClientsRange(start, end uint64) []Session
	GetClientsWithIds(ids ...int64) map[int64]Session
	DisConnect(id int64) error
	SetTimeOut(i int64)
	ServeHTTP(w http.ResponseWriter, req *http.Request)
}

func NewServerHandle() ServerHandlerInterface {
	return newManager()
}

type itemKeys []int64

func (ks itemKeys) Len() int {
	return len(ks)
}

func (ks itemKeys) Less(i, j int) bool {
	return ks[i] > ks[j]
}

func (ks itemKeys) Swap(i, j int) {
	ks[i], ks[j] = ks[j], ks[i]
}

type sessionItem struct {
	Session
	t *time.Timer
}

func (i sessionItem) timerReset(d time.Duration) {
	if i.t != nil {
		i.t.Reset(d)
	}
}

func newManager() *sessionManager {
	managerOnce.Do(func() {
		manager = &sessionManager{
			mu:                   sync.RWMutex{},
			cb:                   nil,
			s:                    nil,
			m:                    map[int64]sessionItem{},
			handshakeCheckHandle: nil,
		}
		webServ := &websocket.Server{
			Handshake: func(config *websocket.Config, request *http.Request) error {
				err := uity.DefaultUpgradeCheck(request)
				if err != nil {
					return err
				}
				if newManager().handshakeCheckHandle != nil {
					return newManager().handshakeCheckHandle(request)
				}
				return nil
			},
			Handler: func(conn *websocket.Conn) {
				manager.addSession(conn)
			},
		}
		manager.s = webServ
	})
	return manager
}

type sessionManager struct {
	mu                   sync.RWMutex
	cb                   *Callbacks
	s                    *websocket.Server
	m                    map[int64]sessionItem
	handshakeCheckHandle func(req *http.Request) error
	timeOutSecond        int64
}

func (s *sessionManager) SetHandshakeCheckHandle(f func(req *http.Request) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.m) == 0 {
		s.handshakeCheckHandle = f
	}
}
func (s *sessionManager) SetCallbacks(callbacks *Callbacks) {
	s.cb = callbacks
}

func (s *sessionManager) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}
func (s *sessionManager) GetSessionOnce(id int64) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if item, ok := s.m[id]; ok {
		return item.Session, nil
	} else {
		return nil, errors.New(fmt.Sprintf("not found session with id(%b)", id))
	}
}
func (s *sessionManager) GetClientsRange(start, end uint64) []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []Session
	if end == 0 || start >= end {
		return list
	}
	var keys itemKeys
	for k := range s.m {
		keys = append(keys, k)
	}
	sort.Sort(keys)
	for i, key := range keys {
		if uint64(i) == end {
			break
		}
		if uint64(i) >= start && uint64(i) < end {
			if item, ok := s.m[key]; ok {
				list = append(list, item.Session)
			}

		}
	}
	return list
}

func (s *sessionManager) GetClientsWithIds(ids ...int64) map[int64]Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := map[int64]Session{}
	if ids == nil || len(ids) == 0 {
		return res
	}
	for _, id := range ids {
		if item, ok := s.m[id]; ok {
			res[item.GetId()] = item.Session
		}
	}
	return res
}

func (s *sessionManager) DisConnect(id int64) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if item, ok := s.m[id]; ok {
		go item.DisConnect()
		return nil
	} else {
		return errors.New(fmt.Sprintf("not found session with id(%b)", id))
	}

}

func (s *sessionManager) SetTimeOut(i int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.m) == 0 {
		s.timeOutSecond = i
	}
}

func (s *sessionManager) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.s.ServeHTTP(w, req)
}
func (s *sessionManager) doTimeOut(id int64) {
	item := s.delSession(id)
	if item != nil {
		if item.t != nil {
			item.t.Stop()
		}
		item.Session.DisConnect(true)
	}
}
func (s *sessionManager) delSession(id int64) *sessionItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item, ok := s.m[id]; ok {
		delete(s.m, id)
		if item.t != nil {
			item.t.Stop()
		}
		return &item
	}
	return nil
}
func (s *sessionManager) addSession(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := sessionItem{
		Session: session.NewSession(conn, true),
		t:       nil,
	}
	item.Session.SetFrameCallBack(s.doMsgCb)
	item.Session.SetDisConnectCallBack(s.doDisConnCb)
	if s.timeOutSecond >= 1 {
		item.t = time.AfterFunc(time.Duration(s.timeOutSecond)*time.Second, func() {
			newManager().doTimeOut(item.GetId())
		})
	}
	s.m[item.GetId()] = item
	go s.doConnCb(item.Session)
}

func (s *sessionManager) doConnCb(sess Session) {
	if s.cb != nil && s.cb.ConnectedCallBack != nil {
		go s.cb.ConnectedCallBack(sess)
	}
}
func (s *sessionManager) doDisConnCb(id int64, status Status) {
	item := s.delSession(id)
	if item != nil && s.cb != nil && s.cb.DisConnectCallBack != nil {
		go s.cb.DisConnectCallBack(id, status)
	}
}

func (s *sessionManager) doMsgCb(id int64, t byte, payload []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if item, ok := s.m[id]; ok && item.t != nil && s.timeOutSecond >= 1 {
		item.t.Reset(time.Duration(s.timeOutSecond) * time.Second)
	}
	if s.cb != nil && s.cb.MessageCallBack != nil {
		go s.cb.MessageCallBack(id, t, payload)
	}
}
