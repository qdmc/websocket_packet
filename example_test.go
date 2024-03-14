package websocket_packet

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/qdmc/websocket_packet/frame"
	"github.com/qdmc/websocket_packet/session"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

var msgCount *uint64

func init() {
	count := uint64(0)
	msgCount = &count
}

func ExampleNewClient() {
	// 配置一个请求的header,将在服务的链接成功回调中获取到
	dialHeader := http.Header{}
	dialHeader.Add("token", "my_token_xxx")
	client := NewClient(&ClientOptions{
		ReConnectMaxNum:   3,
		ReConnectInterval: 5,
		ConnectedCallback: func() {
			// do connected
		},
		DisConnectCallback: func(e error, db *session.ConnectionDatabase) {
			// do disconnect
		},
		MessageCallback: func(t byte, bs []byte) {
			// do frame message
		},
		RequestHeader: dialHeader,
		RequestTime:   30,
		PingTime:      25,
		IsStatistics:  false,
	})
	err := client.Dial("ws://127.0.0.1:8080")
	if err != nil {
		fmt.Println("dialErr: ", err.Error())
	}
}

func ExampleNewServerHandle() {
	serv := NewServerHandle()
	// 配置回调
	serv.SetCallbacks(&CallbackHandles{
		ConnectedCallBackHandle: func(id int64, header http.Header) {
			// do connected
		},
		DisConnectCallBackHandle: func(id int64, s frame.CloseStatus, db *session.ConnectionDatabase) {
			// do disconnect
		},
		FrameCallBackHandle: func(id int64, t byte, bs []byte) {
			// do frame message
		},
	})
	serv.SetHandshakeCheckHandle(func(req *http.Request) error {
		// do Handshake
		return nil
	})
	// 配置超时
	serv.SetTimeOut(60)
	// 配置自动发送ping帧
	serv.SetPingTime(10)
	// 配置打开数据统计
	serv.SetStatistics(true)
	// ServerHandlerInterface 可以做为一个Http.HandlerFunc使用
	http.Handle("/websocket", serv)
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
	go func() {
		for {
			time.Sleep(5 * time.Second)
			fmt.Println("***clientNum: ", server.Len())
		}
	}()
	server.SetCallbacks(cbs)
	server.SetTimeOut(60)
	server.SetPingTime(-1)
	server.SetStatistics(true)
	http.Handle("/websocket", server)
	addr := ":8081"
	println("http start at ", addr, " ......")
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		t.Fatal("listenErr: ", err.Error())
	}

}

func clientConnectedCallback() {
}

func clientDisConnectCallback(e error, db *session.ConnectionDatabase) {
}

func clientMsgCallback(t byte, bs []byte) {
}

func connectedCb(id int64, header http.Header) {
	//println("connected: ", id)
	//go func() {
	//	//time.Sleep(25 * time.Second)
	//	//fmt.Println("--- Disconnect ", id)
	//	//err := server.DisConnect(id)
	//	//if err != nil {
	//	//	fmt.Println("err: ", err)
	//	//	os.Exit(1)
	//	//}
	//}()
}

func disconnectCb(id int64, s frame.CloseStatus, db *session.ConnectionDatabase) {
	//println("disconnect: ", id)
	//err := frame.StatusToError(s)
	//if err != nil {
	//	println("disconnectErr:  ", err.Error())
	//}
}

func msgCb(id int64, t byte, bs []byte) {
	//fmt.Println("********************************************")
	//fmt.Println("id: ", id, " t: ", t, " msg: ", string(bs))
	//fmt.Println("********************************************")
	//_, err := server.SendMessage(id, t, []byte("ok"))
	//if err != nil {
	//	fmt.Println("server.SendMessageErr: ", err.Error())
	//}
	val := atomic.AddUint64(msgCount, 1)
	if val > 1000 && val/1000 == 0 {
		fmt.Println("count: ", val)
	}
}

func Test_ping(t *testing.T) {
	//hexStr := "09840000000470696e63"
	//bs, err := hex.DecodeString(hexStr)
	//if err != nil {
	//	t.Fatal("hexErr: ", err.Error())
	//}
	bs := []byte{0x89, 0x05, 0x48, 0x65, 0x6c, 0x6c, 0x6f}
	fmt.Println("hexString: ", hex.EncodeToString(bs))
	_, f, readErr := frame.ReadOnceFrame(bytes.NewReader(bs))
	if frame.StatusToError(readErr) != nil {
		t.Fatal("ReadOnceFrame: ", frame.StatusToError(readErr).Error())
	}
	fmt.Println("s1 :", f.ToString())
	if f.Opcode != 9 {
		t.Fatal("is not ping frame")
	}
	fmt.Println(string(f.PayloadData))
	f2 := frame.NewPingFrame([]byte("Hello"))
	fmt.Println(f2.ToString())
	bs2, err := f2.ToBytes()
	if err != nil {
		t.Fatal("toBytesErr: ", err.Error())
	}
	fmt.Println("hexStr1 :", hex.EncodeToString(bs))
	fmt.Println("hexStr2 :", hex.EncodeToString(bs2))
}

func Test_Pong(t *testing.T) {
	bs := []byte{0x8a, 0x85, 0x37, 0xfa, 0x21, 0x3d, 0x7f, 0x9f, 0x4d, 0x51, 0x58}
	_, f, readErr := frame.ReadOnceFrame(bytes.NewReader(bs))
	if frame.StatusToError(readErr) != nil {
		t.Fatal("ReadOnceFrame: ", frame.StatusToError(readErr).Error())
	}
	fmt.Println("s1 :", f.ToString())
	if f.Opcode != 10 {
		t.Fatal("is not pong frame")
	}
	f2 := frame.NewPongFrame([]byte("Hello"), 939139389)
	bs2, err := f2.ToBytes()
	if err != nil {
		t.Fatal("toBytesErr: ", err.Error())
	}
	fmt.Println("hexStr1 :", hex.EncodeToString(bs))
	fmt.Println("hexStr2 :", hex.EncodeToString(bs2))
}

func Test_BinaryMsg(t *testing.T) {
	bs, err := frame.AutoBinaryFramesBytes([]byte("test"))
	if err != nil {
		t.Fatal("err: ", err.Error())
	}
	_, f, status := frame.ReadOnceFrame(bytes.NewReader(bs))
	if frame.StatusToError(status) != nil {
		t.Fatal("ReadOnceFrameErr: ")
	}
	fmt.Println(f.ToString())
}
