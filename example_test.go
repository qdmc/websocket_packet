package websocket_packet

import (
	"github.com/qdmc/websocket_packet/session"
	"net/http"
	"testing"
)

func ExampleNewServerHandle() {
	// ServerHandlerInterface 可以做为一个Http.HandlerFunc使用
	http.Handle("/websocket", NewServerHandle())
	http.ListenAndServe(":8080", nil)
}

var server ServerHandlerInterface

func Test_server(t *testing.T) {
	cbs := &CallbackHandles{
		ConnectedCallBackHandle:  connectedCb,
		DisConnectCallBackHandle: disconnectCb,
		FrameCallBackHandle:      msgCb,
	}
	server = NewServerHandle()
	server.SetCallbacks(cbs)
	server.SetTimeOut(60)
	http.Handle("/websocket", server)
	addr := ":8081"
	println("http start at ", addr, " ......")
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		t.Fatal("listenErr: ", err.Error())
	}

}

func connectedCb(session Session) {
	println("connected: ", session.GetId())
	go func() {

	}()
}

func disconnectCb(id int64, s ClientStatus) {
	println("disconnect: ", id)
	err := session.StatusToError(s)
	if err != nil {
		println(err.Error())
	}
}

func msgCb(id int64, t byte, bs []byte) {
	println("acceptFrame: ", id)
	println("acceptFrame: ", t)
	println("acceptFrame: ", string(bs))
}
