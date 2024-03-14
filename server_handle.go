package websocket_packet

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/qdmc/websocket_packet/frame"
	"github.com/qdmc/websocket_packet/session"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

var manager *sessionManager
var managerOnce sync.Once

// CallbackHandles    回调组
type CallbackHandles struct {
	session.ConnectedCallBackHandle  // 建立链接后的回调
	session.DisConnectCallBackHandle // 断开链接后的回调
	session.FrameCallBackHandle      // 帧读取后的回调
}

/*
ServerHandlerInterface 实现的业务
  - 校验websocket握手(Handshake)
  - net.http.Handler:实现ServeHTTP(w http.ResponseWriter, req *http.Request)
  - 管理session:查询,断开
  - 消息的接收与发送
*/
type ServerHandlerInterface interface {
	SetCallbacks(*CallbackHandles)                                                     // 配置回调
	Len() int                                                                          // 返回客户端(Session)总数
	GetSessionOnce(id int64) (Session, error)                                          // 获取一个 Session
	GetSessionRange(start, end uint64) []Session                                       // 获取获取 Session 列表
	GetSessionWithIds(ids ...int64) map[int64]Session                                  // 获取获取 Session 列表
	DisConnect(id int64) error                                                         // 断开一个 Session
	ServeHTTP(w http.ResponseWriter, req *http.Request)                                // 实现net.http.Handler
	SetHandshakeCheckHandle(f func(req *http.Request) error)                           // 配置一个校验的握手的handle
	SendMessage(id int64, frameType byte, payload []byte, keys ...uint32) (int, error) // 发送消息到客户端
	SetStatistics(b bool)                                                              // 是否开启流量统计,在执行ServeHTTP之前有效,默认为false
	SetPingTime(t int64)                                                               // 配置自动发送pingFrame的时间(秒),在执行ServeHTTP之前有效,<1:关闭(默认值); 1~~25:都会配置为25秒; >120:都会配置为120秒
	SetTimeOut(i int64)                                                                // 配置 Session 超时,在执行ServeHTTP之前有效
}

// NewServerHandle      生成一个全局唯一的 ServerHandlerInterface
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
			m:                    map[int64]sessionItem{},
			handshakeCheckHandle: nil,
		}
	})
	return manager
}

type sessionManager struct {
	mu                   sync.RWMutex
	cb                   *CallbackHandles
	m                    map[int64]sessionItem
	handshakeCheckHandle func(req *http.Request) error
	timeOutSecond        int64
	pingTime             int64
	isServerHttp         bool
	isStatistics         bool
}

func (s *sessionManager) SetStatistics(b bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isServerHttp {
		s.isStatistics = b
	}
}
func (s *sessionManager) SetPingTime(t int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isServerHttp {
		s.pingTime = t
	}
}
func (s *sessionManager) SendMessage(id int64, frameType byte, payload []byte, keys ...uint32) (int, error) {
	sess, err := s.GetSessionOnce(id)
	if err != nil {
		return 0, err
	}
	return sess.Write(frameType, payload, keys...)
}

func (s *sessionManager) SetHandshakeCheckHandle(f func(req *http.Request) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.m) == 0 {
		s.handshakeCheckHandle = f
	}
}
func (s *sessionManager) SetCallbacks(callbacks *CallbackHandles) {
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
func (s *sessionManager) GetSessionRange(start, end uint64) []Session {
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

func (s *sessionManager) GetSessionWithIds(ids ...int64) map[int64]Session {
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
		go item.DisConnect(session.CloseNormalClosure)
		return nil
	} else {
		return errors.New(fmt.Sprintf("not found session with id(%b)", id))
	}

}

func (s *sessionManager) SetTimeOut(i int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isServerHttp {
		s.timeOutSecond = i
	}
}

// ServeHTTP            实现Http.HandlerFunc
func (s *sessionManager) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.isServerHttp = true
	var conn net.Conn
	var err error
	conn, err = serverUpgradeHandler(req, w, s.handshakeCheckHandle)
	if err != nil {
		httpResponseError(w, 404, err)
		return
	}
	err = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return
	}
	_, err = conn.Write(makeServerHandshakeBytes(req))
	if err != nil {
		return
	}
	err = conn.SetWriteDeadline(time.Time{})
	if err != nil {
		return
	}
	err = conn.SetDeadline(time.Time{})
	if err != nil {
		return
	}
	go s.addSession(conn, req.Header)
}
func (s *sessionManager) doTimeOut(id int64) {
	item := s.delSession(id)
	if item != nil {
		if item.t != nil {
			item.t.Stop()
		}
		// 如果是超时,给出 1008 状态码,(1008): 协议违规，表示违反了协议的约束或策略
		item.Session.DisConnect(frame.ClosePolicyViolation)
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
func (s *sessionManager) addSession(conn net.Conn, header http.Header) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := session.NewSession(conn, true, &session.ConfigureSession{
		ConnectedCallBackHandle: nil,
		DisConnectCallBack:      s.doDisConnCb,
		FrameCallBackHandle:     s.doMsgCb,
		IsStatistics:            s.isStatistics,
		AutoPingTicker:          s.pingTime,
	})
	sessionId := sess.GetId()
	item := sessionItem{
		Session: sess,
		t:       nil,
	}
	if s.timeOutSecond >= 1 {
		item.t = time.AfterFunc(time.Duration(s.timeOutSecond)*time.Second, func() {
			newManager().doTimeOut(item.GetId())
		})
	}
	s.m[sessionId] = item
	go s.doConnCb(sessionId, header)
	go item.Session.DoConnect()
}

func (s *sessionManager) doConnCb(id int64, header http.Header) {
	if s.cb != nil && s.cb.ConnectedCallBackHandle != nil {
		go s.cb.ConnectedCallBackHandle(id, header)
	}
}
func (s *sessionManager) doDisConnCb(id int64, status ClientStatus) {
	item := s.delSession(id)
	if item != nil && s.cb != nil && s.cb.DisConnectCallBackHandle != nil {
		go s.cb.DisConnectCallBackHandle(id, status)
	}
}

func (s *sessionManager) doMsgCb(id int64, t byte, payload []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if item, ok := s.m[id]; ok && item.t != nil && s.timeOutSecond >= 1 {
		item.t.Reset(time.Duration(s.timeOutSecond) * time.Second)
	}
	if s.cb != nil && s.cb.FrameCallBackHandle != nil {
		go s.cb.FrameCallBackHandle(id, t, payload)
	}
}

// serverUpgradeHandler      server端校验握手
func serverUpgradeHandler(req *http.Request, w http.ResponseWriter, otherHandle func(req *http.Request) error) (conn net.Conn, err error) {
	err = defaultUpgradeCheck(req)
	if err != nil {
		return
	}
	if otherHandle != nil {
		err = otherHandle(req)
		if err != nil {
			return
		}
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("this ResponseWriter is not Hijacker")
	}
	conn, _, err = hj.Hijack()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("HijackErr: %s", err.Error()))
	}
	return conn, nil
}

func defaultUpgradeCheck(r *http.Request) error {
	if r.Method != http.MethodGet {
		return errors.New("bad method")
	}
	if !checkHttpHeaderKeyVale(r.Header, "Connection", "upgrade") {
		return errors.New("missing or bad upgrade")
	}
	if !checkHttpHeaderKeyVale(r.Header, "Upgrade", "websocket") {
		return errors.New("missing or bad WebSocket-Protocol")
	}
	if !checkHttpHeaderKeyVale(r.Header, "Sec-Websocket-Version", "13") {
		return errors.New("bad protocol version")
	}
	if !checkSecWebsocketKey(r.Header) {
		return errors.New("bad 'Sec-WebSocket-Key'")
	}
	return nil
}

func checkHttpHeaderValue(h http.Header, key string) ([]string, bool) {
	if key == "" {
		return nil, false
	}
	if val, ok := h[key]; ok {
		return val, true
	} else {
		return nil, false
	}
}

func checkHttpHeaderKeyVale(h http.Header, key, value string) bool {
	if vals, ok := checkHttpHeaderValue(h, key); ok {
		valueStr := strings.ToLower(value)
		for _, val := range vals {
			if strings.Index(strings.ToLower(val), valueStr) != -1 {
				return true
			}
		}
		return false
	} else {
		return false
	}
}

// checkSecWebsocketKey   校验Sec-Websocket-Key
func checkSecWebsocketKey(h http.Header) bool {
	key := h.Get("Sec-Websocket-Key")
	if key == "" {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(key)
	return err == nil && len(decoded) == 16
}

// makeServerHandshakeBytes    生成服务端回复的报文
func makeServerHandshakeBytes(req *http.Request) []byte {
	key := req.Header.Get("Sec-Websocket-Key")
	var p []byte
	p = append(p, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: "...)
	p = append(p, computeAcceptKey(key)...)
	p = append(p, "\r\n"...)
	p = append(p, "\r\n"...)
	return p
}

// httpResponseError    Response回复错误,在拆解Response前使用
func httpResponseError(w http.ResponseWriter, status int, err error) {
	errStr := http.StatusText(status)
	if err != nil && err.Error() != "" {
		errStr = err.Error()
	}
	http.Error(w, errStr, status)
}

// generateChallengeKey   生成随机的websocketKey
func generateChallengeKey() (string, error) {
	p := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(p), nil
}

// computeAcceptKey     计算websocket的key
func computeAcceptKey(key string) string {
	h := sha1.New() //#nosec G401 -- (CWE-326) https://datatracker.ietf.org/doc/html/rfc6455#page-54
	h.Write([]byte(key))
	h.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
