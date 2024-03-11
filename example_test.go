package websocket_packet

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/qdmc/websocket_packet/frame"
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
	//go func() {
	//	for {
	//		time.Sleep(5 * time.Second)
	//		fmt.Println("***clientNum: ", server.Len())
	//	}
	//}()
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

func connectedCb(id int64, header http.Header) {
	println("connected: ", id)
	go func() {

	}()
}

func disconnectCb(id int64, s frame.CloseStatus) {
	println("disconnect: ", id)
	err := frame.StatusToError(s)
	if err != nil {
		println("disconnectErr:  ", err.Error())
	}
}

func msgCb(id int64, t byte, bs []byte) {
	//println("acceptFrame: ", id)
	//println("acceptFrame: ", t)
	//println("acceptFrame: ", string(bs))
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
