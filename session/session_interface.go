package session

type ConnectedCallBack func(id string)
type DisConnectCallBack func(id string, status Status)
type FrameCallBack func(id string, t byte, payload []byte)

/*
WebsocketSessionInterface      session通用接口
*/
type WebsocketSessionInterface interface {
	GetId() int64
	IsServer() bool
	GetStatus() Status
	DoConnect()
	Write(frameType byte, bs []byte) (int, error)
	SetConnectedCallBack(ConnectedCallBack)
	SetDisConnectCallBack(DisConnectCallBack)
	SetFrameCallBack(FrameCallBack)
	CloseAutoRes()
	DisConnect()
}
