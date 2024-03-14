package websocket_packet

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/qdmc/websocket_packet/frame"
	"github.com/qdmc/websocket_packet/session"
	"net"
	"net/http"

	"net/url"
	"time"
)

// Session        websocket链接通用接口
type Session = session.WebsocketSessionInterface

// SessionDb      链接信息
type SessionDb = session.ConnectionDatabase

// ClientStatus     客户端状态
type ClientStatus = session.Status

/*
ClientOptions               客户端配置
  - ReConnectMaxNum         非正常断开后的重链次数,<0:不重链;0:一直重链;>0:重链接的最大次数;默认是5次
  - ReConnectInterval       重链间隔(秒),默认是5秒,最小为一秒
  - ConnectedCallback       链接成功后的回调
  - DisConnectCallback      断开后的回调
  - MessageCallback         接收到消息的回调
  - RequestHeader           发送请求时携带额外的请求头
  - RequestTime             发送请求的最大时长(秒),默认:10;最小:3;最大:60
  - PingTime                自动发送pingFrame的时间(秒)配置, <1:关闭(默认值); 1~~25:都会配置为25秒; >120:都会配置为120秒
  - IsStatistics            是否开启流量统计,默认为false
*/
type ClientOptions struct {
	ReConnectMaxNum    int
	ReConnectInterval  int64
	ConnectedCallback  func()
	DisConnectCallback func(err error, db *session.ConnectionDatabase)
	MessageCallback    func(byte, []byte)
	RequestHeader      http.Header
	RequestTime        int64
	PingTime           int64
	IsStatistics       bool
}

// NewClientOption      生成一个新的客户端配置
func NewClientOption() *ClientOptions {
	return &ClientOptions{
		ReConnectMaxNum:    5,
		ReConnectInterval:  5,
		ConnectedCallback:  nil,
		DisConnectCallback: nil,
		MessageCallback:    nil,
		RequestHeader:      http.Header{},
		RequestTime:        10,
		PingTime:           0,
		IsStatistics:       false,
	}
}

/*
SetReConnect         配置重连参数
  - maxNum           重链次数,<0:不重链;0:一直重链;>0:重链接的最大次数;默认是5次
  - interval         重链间隔(秒),默认是5秒,最小为一秒
*/
func (o *ClientOptions) SetReConnect(maxNum int, interval int64) *ClientOptions {
	o.ReConnectMaxNum = maxNum
	o.ReConnectInterval = interval
	return o
}

// SetConnectedCb       配置链接成功后的回调
func (o *ClientOptions) SetConnectedCb(f func()) *ClientOptions {
	o.ConnectedCallback = f
	return o
}

// SetDisconnectCb      配置断开后的回调
func (o *ClientOptions) SetDisconnectCb(f func(err error, db *session.ConnectionDatabase)) *ClientOptions {
	o.DisConnectCallback = f
	return o
}

// SetMessageCb         配置接收到消息的回调
func (o *ClientOptions) SetMessageCb(f func(byte, []byte)) *ClientOptions {
	o.MessageCallback = f
	return o
}

// SetHeader             配置发送请求时携带额外的请求头
func (o *ClientOptions) SetHeader(h http.Header) {
	o.RequestHeader = h
}

// NewClient        生成一个客户端
func NewClient(opt *ClientOptions) *Client {
	if opt == nil {
		opt = NewClientOption()
	}
	if opt.ReConnectInterval < 1 {
		opt.ReConnectInterval = 5
	}
	return &Client{
		opt:    opt,
		status: session.ClientCreate,
	}
}

// Client                 websocket客户端,只有一个 Session,并自动发送ping帧(25秒)与自动回复pong帧
type Client struct {
	//mu            sync.Mutex
	opt    *ClientOptions
	s      Session
	status ClientStatus
	url    *url.URL
	//protocol, origin string
}

// GetOptions                返回配置项
func (c *Client) GetOptions() *ClientOptions {
	return c.opt
}

// SetOptions               设置配置项
func (c *Client) SetOptions(opt *ClientOptions) {
	if opt != nil && c.status == session.ClientCreate {
		c.opt = opt
	}
}

// Dial          链接到服务端
func (c *Client) Dial(url string) error {
	if c.status != session.ClientCreate {
		return errors.New("client status is not ClientCreate")
	}
	err := c.parseUrl(url)
	if err != nil {
		return errors.New(fmt.Sprintf("urlErr: %s", err.Error()))
	}
	err = c.dialToServer()
	if err != nil {
		return errors.New(fmt.Sprintf("dialToServerErr: %s", err.Error()))
	}
	return nil
}

/*
SendMessage            发送消息到服务端
  - frameType           消息类型,1:text;2:binary;9:ping;10:pong; 如是close消息,请调用Disconnect()方法
  - payload             消息负载
  - keys                消息加密key[可选]
*/
func (c *Client) SendMessage(frameType byte, payload []byte, keys ...uint32) (int, error) {
	if c.s != nil {
		return c.s.Write(frameType, payload)
	} else {
		return 0, errors.New("not dial to server")
	}
}

// Disconnect          断开与服务器链接
func (c *Client) Disconnect() {
	if c.s != nil {
		c.s.DisConnect(session.CloseNormalClosure)
	}
}

// connCb        链接到服务端回调方法
func (c *Client) connCb() {
	if c.opt != nil && c.opt.ConnectedCallback != nil {
		go c.opt.ConnectedCallback()
	}
}

// disConnCb     断开回调方法
func (c *Client) disConnCb(id int64, s ClientStatus, db *session.ConnectionDatabase) {
	c.status = s
	if c.opt != nil && c.opt.DisConnectCallback != nil {
		go c.opt.DisConnectCallback(frame.StatusToError(s), db)
	}
	if s == session.CloseNormalClosure {
		c.status = session.ClientCreate
		return
	} else {
		c.status = session.ClientReconnect
		go c.reConnect()
	}

}

// msgCb         消息回调方法
func (c *Client) msgCb(id int64, t byte, payload []byte) {
	if c.opt != nil && c.opt.MessageCallback != nil {
		go c.opt.MessageCallback(t, payload)
	}
}

// reConnect     重连方法
func (c *Client) reConnect() {
	// 不是读写错误时,不执行reConnect
	if c.status != session.ClientReconnect {
		return
	}

	if c.opt == nil || c.opt.ReConnectMaxNum < 0 {
		return
	}
	go func() {
		for i := 0; i <= c.opt.ReConnectMaxNum; i++ {
			if c.opt.ReConnectMaxNum == 0 {
				i = 0
			}
			time.Sleep(time.Duration(c.opt.ReConnectInterval) * time.Second)
			err := c.dialToServer()
			if err != nil {
				continue
			} else {
				return
			}
		}
	}()

}

// dialToServer    链接到服务端
func (c *Client) dialToServer() error {
	if c.status != session.ClientReconnect && c.status != session.ClientCreate {
		return errors.New("client status is error")
	}

	if c.opt == nil {
		c.opt = NewClientOption()
	}
	if c.opt.RequestTime < 3 || c.opt.RequestTime > 60 {
		c.opt.RequestTime = 10
	}
	conn, err := net.DialTimeout("tcp", c.url.Host, time.Duration(c.opt.RequestTime)*time.Second)
	if err != nil {
		return err
	}
	req := c.makeRequest()
	err = req.Write(conn)
	if err != nil {
		return err
	}
	bufRead := bufio.NewReaderSize(conn, 4096)
	bufRead.Reset(conn)
	defer func() {
		if err != nil {
			conn.Close()
			c.status = session.ClientConnectFailed
		} else {
			c.status = session.Connected
		}
	}()
	resp, err := http.ReadResponse(bufRead, req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return errors.New("response statusCode is not 101")
	}
	if !checkHttpHeaderKeyVale(resp.Header, "Upgrade", "websocket") {
		return errors.New("response header.Upgrade is not websocket")
	}
	if !checkHttpHeaderKeyVale(resp.Header, "Connection", "upgrade") {
		return errors.New("response header.Connection is not upgrade")
	}
	if resp.Header.Get("Sec-Websocket-Accept") != computeAcceptKey(req.Header.Get("Sec-WebSocket-Key")) {
		return errors.New("response header.Sec-Websocket-Accept is error")
	}
	err = conn.SetDeadline(time.Time{})
	if err != nil {
		return err
	}

	go c.connCb()
	c.s = session.NewSession(conn, false, &session.ConfigureSession{
		ConnectedCallBackHandle: nil,
		DisConnectCallBack:      c.disConnCb,
		FrameCallBackHandle:     c.msgCb,
		IsStatistics:            c.opt.IsStatistics,
		AutoPingTicker:          c.opt.PingTime,
	})
	go c.s.DoConnect()
	return nil
}

// parseUrl      解析url
func (c *Client) parseUrl(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	var scheme string
	if u.Scheme == "ws" {
		scheme = "http"
	} else if u.Scheme == "wss" {
		scheme = "https"
	} else {
		return errors.New("scheme must be ws or wss")
	}
	u.Scheme = scheme
	c.url = u
	return nil
}
func (c *Client) makeRequest() *http.Request {
	req := &http.Request{
		Method:     http.MethodGet,
		URL:        c.url,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       c.url.Host,
	}
	key, _ := generateChallengeKey()
	req.Header["Upgrade"] = []string{"websocket"}
	req.Header["Connection"] = []string{"Upgrade"}
	req.Header.Add("Sec-WebSocket-Key", key)
	req.Header["Sec-WebSocket-Version"] = []string{"13"}
	if c.opt != nil && c.opt.RequestHeader != nil {
		for headKey, headValue := range c.opt.RequestHeader {
			req.Header[headKey] = headValue
		}
	}
	return req
}
