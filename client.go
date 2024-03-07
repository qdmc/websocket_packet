package websocket_packet

import (
	"errors"
	"fmt"
	"github.com/qdmc/websocket_packet/session"
	"golang.org/x/net/websocket"
	"net/url"
	"time"
)

// Session        websocket链接通用接口
type Session = session.WebsocketSessionInterface

// SessionDb      链接信息
type SessionDb = session.ConnectionDatabase

// Status     客户端状态
type Status = session.Status

/*
ClientOptions               客户端配置
  - ReConnectMaxNum         非正常断开后的重链次数,<0:不重链;0:一直重链;>0:重链接的最大次数;默认是5次
  - ReConnectInterval       重链间隔(秒),默认是5秒,最小为一秒
  - ConnectedCallback       链接成功后的回调
  - DisConnectCallback      断开后的回调
  - MessageCallback         接收到消息的回调
*/
type ClientOptions struct {
	ReConnectMaxNum    int
	ReConnectInterval  int64
	ConnectedCallback  func()
	DisConnectCallback func(err error)
	MessageCallback    func(byte, []byte)
}

func NewClientOption() *ClientOptions {
	return &ClientOptions{
		ReConnectMaxNum:    5,
		ReConnectInterval:  5,
		ConnectedCallback:  nil,
		DisConnectCallback: nil,
		MessageCallback:    nil,
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
func (o *ClientOptions) SetDisconnectCb(f func(err error)) *ClientOptions {
	o.DisConnectCallback = f
	return o
}

// SetMessageCb         配置接收到消息的回调
func (o *ClientOptions) SetMessageCb(f func(byte, []byte)) *ClientOptions {
	o.MessageCallback = f
	return o
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

// Client                 websocket客户端
type Client struct {
	//mu            sync.Mutex
	opt                   *ClientOptions
	s                     Session
	status                Status
	url, protocol, origin string
}

// Options                返回配置项
func (c *Client) Options() *ClientOptions {
	return c.opt
}

// Dial          链接到服务端
func (c *Client) Dial(url string) error {
	if c.status != session.ClientCreate {
		return errors.New("client status is not ClientCreate")
	}
	err := c.parseUrl(url)
	if err != nil {
		return err
	}
	err = c.dialToServer()
	if err != nil {
		return err
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
		c.s.DisConnect()
	}
}

// connCb        链接到服务端回调方法
func (c *Client) connCb() {
	if c.opt != nil && c.opt.ConnectedCallback != nil {
		go c.opt.ConnectedCallback()
	}
}

// disConnCb     断开回调方法
func (c *Client) disConnCb(id int64, s Status) {
	c.status = s
	if c.opt != nil && c.opt.DisConnectCallback != nil {
		go c.opt.DisConnectCallback(session.StatusToError(s))
	}
	if s == session.CloseNormalClosure {
		c.status = session.ClientCreate
		return
	}
	go c.reConnect()
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
	if c.status != session.CloseReadConnFailed && c.status != session.CloseWriteConnFailed {
		return
	}
	if c.opt == nil || c.opt.ReConnectMaxNum < 0 {
		return
	}
	go func() {
		c.status = session.ClientReconnect
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
	if c.opt == nil {
		c.opt = NewClientOption()
	}
	conn, err := websocket.Dial(c.url, c.protocol, c.origin)
	if err != nil {
		return err
	}
	c.status = session.Connected
	go c.connCb()
	c.s = session.NewSession(conn)
	c.s.SetDisConnectCallBack(c.disConnCb)
	c.s.SetFrameCallBack(c.msgCb)
	go c.s.DisConnect()
	return nil
}

// parseUrl      解析url
func (c *Client) parseUrl(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	var origin string
	if u.Scheme == "ws" {
		origin = fmt.Sprintf("http://%s", u.Host)
	} else if u.Scheme == "wss" {
		origin = fmt.Sprintf("https://%s", u.Host)
	} else {
		return errors.New("scheme must be in ws or wss")
	}
	c.url = u.String()
	c.protocol = u.Scheme
	c.origin = origin
	return nil
}
